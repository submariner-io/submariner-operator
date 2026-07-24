/*
SPDX-License-Identifier: Apache-2.0

Copyright Contributors to the Submariner project.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ciliumcm

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"slices"
	"time"

	"github.com/pkg/errors"
)

const (
	// DefaultBundleValidity is used when GenerateBundle/ReissueLeaves get validity <= 0.
	DefaultBundleValidity = 10 * 365 * 24 * time.Hour

	// DefaultRenewBefore triggers leaf (or full) renewal this long before NotAfter.
	DefaultRenewBefore = 30 * 24 * time.Hour

	// CertCheckRequeue is how often the Submariner reconciler re-checks TLS when
	// NetworkPlugin is cilium (even without other events).
	CertCheckRequeue = 24 * time.Hour
)

// Bundle holds PEM-encoded TLS material for the publisher (server) and Cilium agents (client).
type Bundle struct {
	CACert     []byte
	CAKey      []byte
	ServerCert []byte
	ServerKey  []byte
	ClientCert []byte
	ClientKey  []byte
}

// RenewAction is the reconcile decision for an existing TLS Secret.
type RenewAction int

const (
	// RenewNone — Secret is valid; leave it alone.
	RenewNone RenewAction = iota
	// RenewLeaves — CA is healthy; re-issue server+client only (no CA change).
	RenewLeaves
	// RenewFull — missing/broken CA or no ca.key for leaf renew; new CA+leaves.
	RenewFull
)

// GenerateBundle creates a long-lived lab/alpha CA, server cert (SAN 127.0.0.1/localhost),
// and client cert suitable for Cilium ClusterMesh etcd client files.
func GenerateBundle(validity time.Duration) (*Bundle, error) {
	if validity <= 0 {
		validity = DefaultBundleValidity
	}

	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "generate CA key")
	}

	caSerial, err := randomSerial()
	if err != nil {
		return nil, err
	}

	now := time.Now().Add(-time.Minute)
	caTemplate := &x509.Certificate{
		SerialNumber:          caSerial,
		Subject:               pkix.Name{CommonName: "submariner-cilium-cm-ca"},
		NotBefore:             now,
		NotAfter:              now.Add(validity),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return nil, errors.Wrap(err, "create CA certificate")
	}

	caKeyPEM, err := marshalECPrivateKey(caKey)
	if err != nil {
		return nil, err
	}

	caCertPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})

	leaves, err := issueLeaves(caDER, caKey, validity, now)
	if err != nil {
		return nil, err
	}

	return &Bundle{
		CACert:     caCertPEM,
		CAKey:      caKeyPEM,
		ServerCert: leaves.serverCert,
		ServerKey:  leaves.serverKey,
		ClientCert: leaves.clientCert,
		ClientKey:  leaves.clientKey,
	}, nil
}

// ReissueLeaves re-issues server and client certificates signed by the existing CA.
// CA PEM material is preserved unchanged.
func ReissueLeaves(caCertPEM, caKeyPEM []byte, validity time.Duration) (*Bundle, error) {
	if validity <= 0 {
		validity = DefaultBundleValidity
	}

	caCert, err := parseCertPEM(caCertPEM)
	if err != nil {
		return nil, errors.Wrap(err, "parse CA certificate")
	}

	caKey, err := parseECPrivateKeyPEM(caKeyPEM)
	if err != nil {
		return nil, errors.Wrap(err, "parse CA key")
	}

	now := time.Now().Add(-time.Minute)

	leaves, err := issueLeaves(caCert.Raw, caKey, validity, now)
	if err != nil {
		return nil, err
	}

	return &Bundle{
		CACert:     append([]byte(nil), caCertPEM...),
		CAKey:      append([]byte(nil), caKeyPEM...),
		ServerCert: leaves.serverCert,
		ServerKey:  leaves.serverKey,
		ClientCert: leaves.clientCert,
		ClientKey:  leaves.clientKey,
	}, nil
}

type leafPEMs struct {
	serverCert []byte
	serverKey  []byte
	clientCert []byte
	clientKey  []byte
}

func issueLeaves(caDER []byte, caKey *ecdsa.PrivateKey, validity time.Duration, now time.Time) (*leafPEMs, error) {
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		return nil, errors.Wrap(err, "parse CA DER")
	}

	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "generate server key")
	}

	serverSerial, err := randomSerial()
	if err != nil {
		return nil, err
	}

	serverTemplate := &x509.Certificate{
		SerialNumber: serverSerial,
		Subject:      pkix.Name{CommonName: "submariner-cilium-cm"},
		NotBefore:    now,
		NotAfter:     now.Add(validity),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		DNSNames:     []string{"localhost"},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}

	serverDER, err := x509.CreateCertificate(rand.Reader, serverTemplate, caCert, &serverKey.PublicKey, caKey)
	if err != nil {
		return nil, errors.Wrap(err, "create server certificate")
	}

	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, errors.Wrap(err, "generate client key")
	}

	clientSerial, err := randomSerial()
	if err != nil {
		return nil, err
	}

	clientTemplate := &x509.Certificate{
		SerialNumber: clientSerial,
		Subject:      pkix.Name{CommonName: "remote"},
		NotBefore:    now,
		NotAfter:     now.Add(validity),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	clientDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		return nil, errors.Wrap(err, "create client certificate")
	}

	serverKeyPEM, err := marshalECPrivateKey(serverKey)
	if err != nil {
		return nil, err
	}

	clientKeyPEM, err := marshalECPrivateKey(clientKey)
	if err != nil {
		return nil, err
	}

	return &leafPEMs{
		serverCert: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: serverDER}),
		serverKey:  serverKeyPEM,
		clientCert: pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientDER}),
		clientKey:  clientKeyPEM,
	}, nil
}

// SecretData returns the map used in corev1.Secret.Data for TLSSecretName.
func (b *Bundle) SecretData() map[string][]byte {
	return map[string][]byte{
		CACertKey:     b.CACert,
		CAKeyKey:      b.CAKey,
		TLSCertKey:    b.ServerCert,
		TLSKeyKey:     b.ServerKey,
		ClientCertKey: b.ClientCert,
		ClientKeyKey:  b.ClientKey,
	}
}

// RequiredSecretKeys are always required in submariner-cilium-cm-tls.
func RequiredSecretKeys() []string {
	return []string{CACertKey, TLSCertKey, TLSKeyKey, ClientCertKey, ClientKeyKey}
}

// AssessSecretData decides whether to leave, renew leaves, or fully regenerate the bundle.
func AssessSecretData(data map[string][]byte, now time.Time, renewBefore time.Duration) (RenewAction, string) {
	if renewBefore <= 0 {
		renewBefore = DefaultRenewBefore
	}

	if data == nil {
		return RenewFull, "secret data is empty"
	}

	if key := missingRequiredSecretKey(data); key != "" {
		return RenewFull, fmt.Sprintf("missing key %q", key)
	}

	caCert, err := parseCertPEM(data[CACertKey])
	if err != nil {
		return RenewFull, "invalid CA certificate: " + err.Error()
	}

	if !caCert.IsCA {
		return RenewFull, "CA certificate is not marked as CA"
	}

	if err := certCurrentlyValid(caCert, now); err != nil {
		return RenewFull, "CA " + err.Error()
	}

	return assessLeafSecretData(data, caCert, now, renewBefore)
}

func missingRequiredSecretKey(data map[string][]byte) string {
	for _, key := range RequiredSecretKeys() {
		if len(data[key]) == 0 {
			return key
		}
	}

	return ""
}

func assessLeafSecretData(data map[string][]byte, caCert *x509.Certificate, now time.Time,
	renewBefore time.Duration,
) (RenewAction, string) {
	serverCert, err := parseCertPEM(data[TLSCertKey])
	if err != nil {
		return leafOrFull(data, "invalid server certificate: "+err.Error())
	}

	clientCert, err := parseCertPEM(data[ClientCertKey])
	if err != nil {
		return leafOrFull(data, "invalid client certificate: "+err.Error())
	}

	if err := verifyKeyPair(data[TLSCertKey], data[TLSKeyKey]); err != nil {
		return leafOrFull(data, "server key/cert mismatch: "+err.Error())
	}

	if err := verifyKeyPair(data[ClientCertKey], data[ClientKeyKey]); err != nil {
		return leafOrFull(data, "client key/cert mismatch: "+err.Error())
	}

	if err := verifySignedBy(serverCert, caCert); err != nil {
		return leafOrFull(data, "server cert not signed by CA: "+err.Error())
	}

	if err := verifySignedBy(clientCert, caCert); err != nil {
		return leafOrFull(data, "client cert not signed by CA: "+err.Error())
	}

	if err := checkServerSANs(serverCert); err != nil {
		return leafOrFull(data, err.Error())
	}

	if !hasEKU(serverCert, x509.ExtKeyUsageServerAuth) {
		return leafOrFull(data, "server cert missing ServerAuth EKU")
	}

	if !hasEKU(clientCert, x509.ExtKeyUsageClientAuth) {
		return leafOrFull(data, "client cert missing ClientAuth EKU")
	}

	if err := certRenewWindow(serverCert, now, renewBefore); err != nil {
		return leafOrFull(data, "server "+err.Error())
	}

	if err := certRenewWindow(clientCert, now, renewBefore); err != nil {
		return leafOrFull(data, "client "+err.Error())
	}

	if len(data[CAKeyKey]) == 0 {
		// Still usable until renew; leaf renew needs ca.key.
		return RenewNone, "ok (ca.key absent; full regen required on next renew)"
	}

	if _, err := parseECPrivateKeyPEM(data[CAKeyKey]); err != nil {
		return RenewFull, "invalid CA key: " + err.Error()
	}

	return RenewNone, "ok"
}

func leafOrFull(data map[string][]byte, reason string) (RenewAction, string) {
	if len(data[CAKeyKey]) == 0 {
		return RenewFull, reason + " (no ca.key for leaf renew)"
	}

	if _, err := parseECPrivateKeyPEM(data[CAKeyKey]); err != nil {
		return RenewFull, reason + " (unusable ca.key)"
	}

	return RenewLeaves, reason
}

// certCurrentlyValid reports only hard validity failures (used for CA — we do not
// auto-rotate CA merely because it is approaching NotAfter).
func certCurrentlyValid(cert *x509.Certificate, now time.Time) error {
	if now.Before(cert.NotBefore) {
		return fmt.Errorf("not yet valid (NotBefore=%s)", cert.NotBefore.UTC().Format(time.RFC3339))
	}

	if !now.Before(cert.NotAfter) {
		return fmt.Errorf("expired (NotAfter=%s)", cert.NotAfter.UTC().Format(time.RFC3339))
	}

	return nil
}

func certRenewWindow(cert *x509.Certificate, now time.Time, renewBefore time.Duration) error {
	if err := certCurrentlyValid(cert, now); err != nil {
		return err
	}

	if cert.NotAfter.Sub(now) < renewBefore {
		return fmt.Errorf("within renew window (NotAfter=%s)", cert.NotAfter.UTC().Format(time.RFC3339))
	}

	return nil
}

func checkServerSANs(cert *x509.Certificate) error {
	hasLocalhost := slices.Contains(cert.DNSNames, "localhost")
	hasLoopback := false

	for _, ip := range cert.IPAddresses {
		if ip.Equal(net.ParseIP("127.0.0.1")) {
			hasLoopback = true
			break
		}
	}

	if !hasLocalhost || !hasLoopback {
		return fmt.Errorf("server cert missing required SANs localhost/127.0.0.1 (dns=%v ips=%v)",
			cert.DNSNames, cert.IPAddresses)
	}

	return nil
}

func hasEKU(cert *x509.Certificate, want x509.ExtKeyUsage) bool {
	return slices.Contains(cert.ExtKeyUsage, want)
}

func verifySignedBy(cert, ca *x509.Certificate) error {
	roots := x509.NewCertPool()
	roots.AddCert(ca)

	_, err := cert.Verify(x509.VerifyOptions{
		Roots:     roots,
		KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageAny},
	})

	return err //nolint:wrapcheck // passed to Assess reason string
}

func verifyKeyPair(certPEM, keyPEM []byte) error {
	_, err := parseCertPEM(certPEM)
	if err != nil {
		return err
	}

	block, _ := pem.Decode(keyPEM)
	if block == nil {
		return errors.New("failed to decode key PEM")
	}

	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return errors.Wrap(err, "parse EC private key")
	}

	cert, err := parseCertPEM(certPEM)
	if err != nil {
		return err
	}

	pub, ok := cert.PublicKey.(*ecdsa.PublicKey)
	if !ok {
		return errors.New("certificate public key is not ECDSA")
	}

	if !pub.Equal(&key.PublicKey) {
		return errors.New("public key does not match certificate")
	}

	return nil
}

// PeerConfigYAML is the Cilium ClusterMesh etcd config blob for remoteName.
func PeerConfigYAML(remoteName, endpointURL string) []byte {
	return fmt.Appendf(nil, `endpoints:
- %s
trusted-ca-file: /var/lib/cilium/clustermesh/%s.etcd-client-ca.crt
key-file: /var/lib/cilium/clustermesh/%s.etcd-client.key
cert-file: /var/lib/cilium/clustermesh/%s.etcd-client.crt
`, endpointURL, remoteName, remoteName, remoteName)
}

func parseCertPEM(pemBytes []byte) (*x509.Certificate, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode certificate PEM")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "parse certificate")
	}

	return cert, nil
}

func parseECPrivateKeyPEM(pemBytes []byte) (*ecdsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, errors.New("failed to decode key PEM")
	}

	key, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, errors.Wrap(err, "parse EC private key")
	}

	return key, nil
}

func randomSerial() (*big.Int, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return nil, errors.Wrap(err, "generate serial")
	}

	return serial, nil
}

func marshalECPrivateKey(key *ecdsa.PrivateKey) ([]byte, error) {
	der, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		return nil, errors.Wrap(err, "marshal EC private key")
	}

	return pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der}), nil
}
