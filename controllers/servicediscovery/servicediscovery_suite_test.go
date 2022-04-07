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

package servicediscovery_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/submariner-io/admiral/pkg/log/kzerolog"
	submariner_v1 "github.com/submariner-io/submariner-operator/api/submariner/v1alpha1"
	"github.com/submariner-io/submariner-operator/controllers/constants"
	"github.com/submariner-io/submariner-operator/controllers/servicediscovery"
	"github.com/submariner-io/submariner-operator/controllers/test"
	"github.com/submariner-io/submariner-operator/pkg/names"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	fakeKubeClient "k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	controllerClient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	serviceDiscoveryName     = "test-service-discovery"
	submarinerNamespace      = "test-ns"
	openShiftDNSConfigName   = "default"
	clusterIP                = "10.10.10.10"
	lighthouseDNSServiceName = "submariner-lighthouse-coredns"

	lighthouseDNSConfigFormat = `clusterset.local:53 {
    forward . $IP
}
supercluster.local:53 {
    forward . $IP
}`
	coreDNSConfigFormat = "#lighthouse-start AUTO-GENERATED SECTION. DO NOT EDIT\n" + lighthouseDNSConfigFormat +
		"\n#lighthouse-end\n"
)

var _ = BeforeSuite(func() {
	Expect(submariner_v1.AddToScheme(scheme.Scheme)).To(Succeed())
	Expect(operatorv1.Install(scheme.Scheme)).To(Succeed())
})

var _ = Describe("", func() {
	kzerolog.InitK8sLogging()
})

func TestSubmariner(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ServiceDiscovery Test Suite")
}

type testDriver struct {
	test.Driver
	kubeClient       *fakeKubeClient.Clientset
	serviceDiscovery *submariner_v1.ServiceDiscovery
}

func newTestDriver() *testDriver {
	t := &testDriver{
		Driver: test.Driver{
			Namespace:    submarinerNamespace,
			ResourceName: serviceDiscoveryName,
		},
	}

	BeforeEach(func() {
		t.BeforeEach()
		t.serviceDiscovery = newServiceDiscovery()
		t.kubeClient = fakeKubeClient.NewSimpleClientset()
		t.InitClientObjs = []controllerClient.Object{t.serviceDiscovery}
	})

	JustBeforeEach(func() {
		t.JustBeforeEach()

		t.Controller = servicediscovery.NewReconciler(&servicediscovery.Config{
			Client:         t.Client,
			Scheme:         scheme.Scheme,
			KubeClient:     t.kubeClient,
			OperatorClient: t.Client,
		})
	})

	return t
}

func (t *testDriver) awaitFinalizer() {
	t.AwaitFinalizer(t.serviceDiscovery, constants.CleanupFinalizer)
}

func (t *testDriver) awaitServiceDiscoveryDeleted() {
	t.AwaitNoResource(t.serviceDiscovery)
}

func (t *testDriver) assertUninstallServiceDiscoveryDeployment() *appsv1.Deployment {
	deployment := t.AssertDeployment(names.AppendUninstall(names.ServiceDiscoveryComponent))

	t.AssertUninstallInitContainer(&deployment.Spec.Template,
		fmt.Sprintf("%s/%s:%s", t.serviceDiscovery.Spec.Repository, names.ServiceDiscoveryImage, t.serviceDiscovery.Spec.Version))

	return deployment
}

func (t *testDriver) getDNSConfig() (*operatorv1.DNS, error) {
	foundDNSConfig := &operatorv1.DNS{}
	err := t.Client.Get(context.TODO(), types.NamespacedName{Name: openShiftDNSConfigName}, foundDNSConfig)

	return foundDNSConfig, err
}

func (t *testDriver) assertDNSConfig() *operatorv1.DNS {
	foundDNSConfig, err := t.getDNSConfig()
	Expect(err).To(Succeed())

	return foundDNSConfig
}

func (t *testDriver) assertCoreDNSConfigMap() *corev1.ConfigMap {
	return t.assertConfigMap("coredns", "kube-system")
}

func (t *testDriver) assertConfigMap(name, namespace string) *corev1.ConfigMap {
	foundCoreMap, err := t.kubeClient.CoreV1().ConfigMaps(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	Expect(err).To(Succeed())

	return foundCoreMap
}

func (t *testDriver) createConfigMap(cm *corev1.ConfigMap) {
	_, err := t.kubeClient.CoreV1().ConfigMaps(cm.Namespace).Create(context.TODO(), cm, metav1.CreateOptions{})
	Expect(err).To(Succeed())
}

func newDNSConfig(clusterIP string) *operatorv1.DNS {
	dns := &operatorv1.DNS{
		ObjectMeta: metav1.ObjectMeta{
			Name: openShiftDNSConfigName,
		},
		Spec: operatorv1.DNSSpec{
			Servers: []operatorv1.Server{
				{
					Name:  "other",
					Zones: []string{"other.local"},
					ForwardPlugin: operatorv1.ForwardPlugin{
						Upstreams: []string{"1.2.3.4"},
					},
				},
			},
		},
	}

	if clusterIP != "" {
		dns.Spec.Servers = append(dns.Spec.Servers,
			operatorv1.Server{
				Name:  "lighthouse",
				Zones: []string{"clusterset.local"},
				ForwardPlugin: operatorv1.ForwardPlugin{
					Upstreams: []string{clusterIP},
				},
			},
			operatorv1.Server{
				Name:  "lighthouse",
				Zones: []string{"supercluster.local"},
				ForwardPlugin: operatorv1.ForwardPlugin{
					Upstreams: []string{clusterIP},
				},
			})
	}

	dns.Spec.Servers = append(dns.Spec.Servers, operatorv1.Server{
		Name:  "another",
		Zones: []string{"another.local"},
		ForwardPlugin: operatorv1.ForwardPlugin{
			Upstreams: []string{"5.6.7.8"},
		},
	})

	return dns
}

func newServiceDiscovery() *submariner_v1.ServiceDiscovery {
	return &submariner_v1.ServiceDiscovery{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceDiscoveryName,
			Namespace: submarinerNamespace,
		},
		Spec: submariner_v1.ServiceDiscoverySpec{
			GlobalnetEnabled:         false,
			Repository:               "quay.io/submariner",
			Version:                  "1.0.0",
			BrokerK8sRemoteNamespace: "submariner-broker",
			BrokerK8sApiServer:       "https://192.168.99.110:8443",
			BrokerK8sApiServerToken:  "MIIDADCCAeigAw",
			BrokerK8sCA:              "client.crt",
			ClusterID:                "east",
			Namespace:                "submariner_ns",
			Debug:                    true,
			CustomDomains:            []string{"supercluster.local"},
		},
	}
}

func newDNSService(clusterIP string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      lighthouseDNSServiceName,
			Namespace: submarinerNamespace,
		},
		Spec: corev1.ServiceSpec{
			ClusterIP: clusterIP,
		},
	}
}

func (t *testDriver) assertLighthouseCoreDNSService() *corev1.Service {
	service := &corev1.Service{}
	Expect(t.Client.Get(context.TODO(), types.NamespacedName{Name: lighthouseDNSServiceName, Namespace: submarinerNamespace},
		service)).To(Succeed())

	Expect(service.Labels).To(HaveKeyWithValue("app", lighthouseDNSServiceName))
	Expect(service.Spec.Ports).To(HaveLen(1))
	Expect(service.Spec.Ports[0].Protocol).To(Equal(corev1.Protocol("UDP")))
	Expect(service.Spec.Ports[0].Port).To(Equal(int32(53)))
	Expect(service.Spec.Ports[0].TargetPort.IntVal).To(Equal(int32(53)))

	return service
}

func (t *testDriver) setLighthouseCoreDNSServiceIP() {
	service := t.assertLighthouseCoreDNSService()
	service.Spec.ClusterIP = clusterIP
	Expect(t.Client.Update(context.TODO(), service)).To(Succeed())
}

func (t *testDriver) testServiceDiscoveryDeleted() {
	It("eventually delete the ServiceDiscovery resource", func() {
		t.awaitServiceDiscoveryDeleted()
	})
}

func assertDNSConfigServers(actual, expected *operatorv1.DNS) {
	serverKey := func(s *operatorv1.Server) string {
		return fmt.Sprintf("Name: %s, Zones: %v", s.Name, s.Zones)
	}

	actualServers := map[string]*operatorv1.Server{}
	for i := range actual.Spec.Servers {
		actualServers[serverKey(&actual.Spec.Servers[i])] = &actual.Spec.Servers[i]
	}

	for i := range expected.Spec.Servers {
		key := serverKey(&expected.Spec.Servers[i])
		actualServer := actualServers[key]
		Expect(actualServer).ToNot(BeNil(), fmt.Sprintf("Missing expected Server %q", key))
		Expect(actualServer).To(Equal(&expected.Spec.Servers[i]))
		delete(actualServers, key)
	}

	for _, s := range actualServers {
		Fail(fmt.Sprintf("Unexpected Server %#v", s))
	}
}

func coreDNSCorefileData(clusterIP string) string {
	lighthouseConfig := ""
	if clusterIP != "" {
		lighthouseConfig = strings.ReplaceAll(coreDNSConfigFormat, "$IP", clusterIP)
	}

	return lighthouseConfig + `.:53 {
		errors
		health {
		lameduck 5s
	}
		ready
		kubernetes cluster1.local in-addr.arpa ip6.arpa {
		pods insecure
		fallthrough in-addr.arpa ip6.arpa
		ttl 30
	}
		prometheus :9153
		forward . /etc/resolv.conf
		cache 30
		loop
		reload
		loadbalance
	}`
}

func newCoreDNSConfigMap(corefile string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "coredns",
			Namespace: "kube-system",
		},
		Data: map[string]string{
			"Corefile": corefile,
		},
	}
}
