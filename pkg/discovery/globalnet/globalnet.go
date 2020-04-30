/*
© 2020 Red Hat, Inc. and others.

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

package globalnet

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math/bits"
	"net"

	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
	"github.com/submariner-io/submariner-operator/pkg/subctl/datafile"
	submarinerClientset "github.com/submariner-io/submariner/pkg/client/clientset/versioned"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type GlobalNetwork struct {
	GlobalCIDRs  []string
	ServiceCIDRs []string
	ClusterCIDRs []string
	ClusterId    string
}

type GlobalCIDR struct {
	cidr              string
	net               *net.IPNet
	allocatedClusters []*CIDR
	allocatedCount    int
}

type CIDR struct {
	network *net.IPNet
	size    int
	lastIp  uint
}

type Config struct {
	GlobalnetClusterSize uint
	GlobalnetCIDR        string
	ServiceCIDR          string
	ClusterCIDR          string
	ClusterID            string
}

type overlapType int

const (
	cluster overlapType = iota
	service
	global
)

var globalCidr = GlobalCIDR{allocatedCount: 0}
var status = cli.NewStatus()

func (gn *GlobalNetwork) Show() {
	if gn == nil {
		fmt.Println("    No global network details discovered")
	} else {
		fmt.Printf("    Discovered global network details for Cluster %s:\n", gn.ClusterId)
		fmt.Printf("        ServiceCidrs: %v\n", gn.ServiceCIDRs)
		fmt.Printf("        ClusterCidrs: %v\n", gn.ClusterCIDRs)
		fmt.Printf("        Global CIDRs: %v\n", gn.GlobalCIDRs)
	}
}

func ShowNetworks(networks map[string]*GlobalNetwork) {
	for _, network := range networks {
		network.Show()
	}
}

func Discover(client *submarinerClientset.Clientset, namespace string) (map[string]*GlobalNetwork, error) {
	clusters, err := client.SubmarinerV1().Clusters(namespace).List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	var globalNetworks = make(map[string]*GlobalNetwork)
	for _, cluster := range clusters.Items {
		globalNetwork := GlobalNetwork{
			GlobalCIDRs:  cluster.Spec.GlobalCIDR,
			ServiceCIDRs: cluster.Spec.ServiceCIDR,
			ClusterCIDRs: cluster.Spec.ClusterCIDR,
			ClusterId:    cluster.Spec.ClusterID,
		}
		globalNetworks[cluster.Spec.ClusterID] = &globalNetwork
	}
	return globalNetworks, nil
}

func isOverlappingCIDR(cidrList []string, cidr string) (bool, error) {
	_, newNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err
	}
	for _, v := range cidrList {
		_, baseNet, err := net.ParseCIDR(v)
		if err != nil {
			return false, err
		}
		if baseNet.Contains(newNet.IP) || newNet.Contains(baseNet.IP) {
			return true, nil
		}
	}
	return false, nil
}

func NewCIDR(cidr string) CIDR {
	_, network, _ := net.ParseCIDR(cidr)
	ones, total := network.Mask.Size()
	size := total - ones
	lastIp := LastIp(network)
	clusterCidr := CIDR{network: network, size: size, lastIp: lastIp}
	return clusterCidr
}

func LastIp(network *net.IPNet) uint {
	ones, total := network.Mask.Size()
	clusterSize := uint(total - ones)
	firstIpInt := ipToUint(network.IP)
	lastIpUint := (firstIpInt + 1<<clusterSize) - 1
	return lastIpUint
}

func allocateByCidr(cidr string) (uint, error) {
	requestedIp, requestedNetwork, err := net.ParseCIDR(cidr)
	if err != nil || !globalCidr.net.Contains(requestedIp) {
		return 0, fmt.Errorf("%s not a valid subnet of %v\n", cidr, globalCidr.net)
	}
	clusterCidr := NewCIDR(cidr)
	if !globalCidr.net.Contains(uintToIP(clusterCidr.lastIp)) {
		return 0, fmt.Errorf("%s not a valid subnet of %v\n", cidr, globalCidr.net)
	}
	for i := 0; i < globalCidr.allocatedCount; i++ {
		allocated := globalCidr.allocatedClusters[i]
		if allocated.network.Contains(requestedIp) {
			//subset of already allocated, try next
			return allocated.lastIp, fmt.Errorf("%s subset of already allocated globalCidr %v\n", cidr, allocated.network)
		}
		if requestedNetwork.Contains(allocated.network.IP) {
			//already allocated is subset of requested, no valid lastIp
			return clusterCidr.lastIp, fmt.Errorf("%s overlaps with already allocated globalCidr %s\n", cidr, allocated.network)
		}
	}
	globalCidr.allocatedClusters = append(globalCidr.allocatedClusters, &clusterCidr)
	globalCidr.allocatedCount++
	return 0, nil
}

func allocateByClusterSize(numSize uint) (string, error) {
	bitSize := bits.LeadingZeros(0) - bits.LeadingZeros(numSize-1)
	_, totalbits := globalCidr.net.Mask.Size()
	clusterPrefix := totalbits - bitSize
	mask := net.CIDRMask(clusterPrefix, totalbits)

	cidr := fmt.Sprintf("%s/%d", globalCidr.net.IP, clusterPrefix)

	last, err := allocateByCidr(cidr)
	if err != nil && last == 0 {
		return "", err
	}
	for err != nil {
		nextNet := net.IPNet{
			IP:   uintToIP(last + 1),
			Mask: mask,
		}
		cidr = nextNet.String()
		last, err = allocateByCidr(cidr)
		if err != nil && last == 0 {
			return "", fmt.Errorf("allocation not available")
		}
	}
	return cidr, nil
}

func AllocateGlobalCIDR(globalNetworks map[string]*GlobalNetwork, subctlData *datafile.SubctlData) (string, error) {
	globalCidr = GlobalCIDR{allocatedCount: 0, cidr: subctlData.GlobalnetCidrRange}
	_, network, err := net.ParseCIDR(globalCidr.cidr)
	if err != nil {
		return "", fmt.Errorf("invalid GlobalCIDR %s configured", globalCidr.cidr)
	}
	globalCidr.net = network
	for _, globalNetwork := range globalNetworks {
		for _, otherCluster := range globalNetwork.GlobalCIDRs {
			otherClusterCIDR := NewCIDR(otherCluster)
			globalCidr.allocatedClusters = append(globalCidr.allocatedClusters, &otherClusterCIDR)
			globalCidr.allocatedCount++
		}
	}
	return allocateByClusterSize(subctlData.GlobalnetClusterSize)
}

func ipToUint(ip net.IP) uint {
	intIp := ip
	if len(ip) == 16 {
		intIp = ip[12:16]
	}
	return uint(binary.BigEndian.Uint32(intIp))
}

func uintToIP(ip uint) net.IP {
	netIp := make(net.IP, 4)
	binary.BigEndian.PutUint32(netIp, uint32(ip))
	return netIp
}

func GetValidClusterSize(cidrRange string, clusterSize uint) (uint, error) {

	_, network, err := net.ParseCIDR(cidrRange)
	if err != nil {
		return 0, err
	}
	ones, totalbits := network.Mask.Size()
	availableSize := 1 << uint(totalbits-ones)
	userClusterSize := clusterSize
	clusterSize = nextPowerOf2(uint32(clusterSize))
	if clusterSize > uint(availableSize/2) {
		return 0, fmt.Errorf("Cluster size %d, should be <= %d", userClusterSize, availableSize/2)
	}
	return clusterSize, nil
}

//Refer: https://graphics.stanford.edu/~seander/bithacks.html#RoundUpPowerOf2
func nextPowerOf2(n uint32) uint {
	n--
	n |= n >> 1
	n |= n >> 2
	n |= n >> 4
	n |= n >> 8
	n |= n >> 16
	n++
	return uint(n)
}

func CheckOverlappingCidr(networks map[string]*GlobalNetwork, overlappingFor overlapType, netconfig Config) error {
	var cidrlist []string
	var cidr string
	for k, v := range networks {
		switch overlappingFor {
		case cluster:
			cidrlist = v.ClusterCIDRs
			cidr = netconfig.ClusterCIDR
		case service:
			cidrlist = v.ServiceCIDRs
			cidr = netconfig.ServiceCIDR
		case global:
			cidrlist = v.GlobalCIDRs
			cidr = netconfig.GlobalnetCIDR
		}
		overlap, err := isOverlappingCIDR(cidrlist, cidr)
		if err != nil {
			return fmt.Errorf("unable to validate overlapping CIDR: %s", err)
		}
		if overlap && k != netconfig.ClusterID {
			return fmt.Errorf("invalid CIDR: %s overlaps with cluster %s", cidr, k)
		}
	}
	return nil
}

func isCIDRPreConfigured(clusterID string, globalNetworks map[string]*GlobalNetwork) bool {
	// GlobalCIDR is not pre-configured
	if globalNetworks[clusterID] == nil || globalNetworks[clusterID].GlobalCIDRs == nil || len(globalNetworks[clusterID].GlobalCIDRs) <= 0 {
		return false
	}
	// GlobalCIDR is pre-configured
	return true
}

func ValidateGlobalnetConfiguration(subctlData *datafile.SubctlData, netconfig Config) (error, string, uint) {
	status.Start("Validating Globalnet configurations")
	globalnetClusterSize := netconfig.GlobalnetClusterSize
	globalnetCIDR := netconfig.GlobalnetCIDR
	if subctlData.GlobalnetCidrRange != "" && globalnetClusterSize != 0 && globalnetClusterSize != subctlData.GlobalnetClusterSize {
		clusterSize, err := GetValidClusterSize(subctlData.GlobalnetCidrRange, globalnetClusterSize)
		if err != nil || clusterSize == 0 {
			return fmt.Errorf("Invalid globalnet-cluster-size %s", err), "", 0
		}
		subctlData.GlobalnetClusterSize = clusterSize
	}

	if globalnetCIDR != "" && globalnetClusterSize != 0 {
		err := errors.New("Both globalnet-cluster-size and globalnet-cidr can't be specified. Specify either one.\n")
		return fmt.Errorf("%s", err), "", 0
	}

	if globalnetCIDR != "" {
		_, _, err := net.ParseCIDR(globalnetCIDR)
		if err != nil {
			return fmt.Errorf("Specified globalnet-cidr is invalid: %s", err), globalnetCIDR, globalnetClusterSize
		}
	}

	if subctlData.GlobalnetCidrRange == "" {
		if globalnetCIDR != "" {
			status.QueueSuccessMessage("globalnet is not enabled on Broker. Ignoring specified globalnet-cidr")
			globalnetCIDR = ""
		} else if globalnetClusterSize != 0 {
			status.QueueSuccessMessage("globalnet is not enabled on Broker. Ignoring specified globalnet-cluster-size")
			globalnetClusterSize = 0
		}
	}
	status.End(true)
	return nil, globalnetCIDR, globalnetClusterSize
}

func GetGlobalNetworks(subctlData *datafile.SubctlData) (map[string]*GlobalNetwork, error) {

	brokerConfig := subctlData.GetBrokerAdministratorConfig()
	brokerSubmClient, err := submarinerClientset.NewForConfig(brokerConfig)
	if err != nil {
		return nil, err
	}
	// exitOnError("Unable to create submariner rest client for broker cluster", err)
	brokerNamespace := string(subctlData.ClientToken.Data["namespace"])
	globalNetworks, err := Discover(brokerSubmClient, brokerNamespace)
	if err != nil {
		return nil, err
	}
	// exitOnError("Error trying to discover multi-cluster network details", err)
	if globalNetworks != nil {
		ShowNetworks(globalNetworks)
	}
	return globalNetworks, nil
}

func AssignGlobalnetIPs(subctlData *datafile.SubctlData, globalNetworks map[string]*GlobalNetwork, netconfig Config) (string, error) {
	status.Start("Assigning Globalnet IPs")
	globalnetCIDR := netconfig.GlobalnetCIDR
	clusterID := netconfig.ClusterID
	if globalnetCIDR == "" {
		// Globalnet enabled, GlobalCIDR not specified by the user
		if isCIDRPreConfigured(clusterID, globalNetworks) {
			// globalCidr already configured on this cluster
			globalnetCIDR = globalNetworks[clusterID].GlobalCIDRs[0]
			status.QueueSuccessMessage(fmt.Sprintf("Cluster already has GlobalCIDR allocated: %s", globalNetworks[clusterID].GlobalCIDRs[0]))
		} else {
			// no globalCidr configured on this cluster
			globalnetCIDR, err := AllocateGlobalCIDR(globalNetworks, subctlData)
			if err != nil {
				return "", fmt.Errorf("Globalnet failed %s", err)
			}
			status.QueueSuccessMessage(fmt.Sprintf("Allocated GlobalCIDR: %s", globalnetCIDR))
		}
	} else {
		// Globalnet enabled, globalnetCIDR specified by user
		if isCIDRPreConfigured(clusterID, globalNetworks) {
			// globalCidr pre-configured on this cluster
			globalnetCIDR = globalNetworks[clusterID].GlobalCIDRs[0]
			status.QueueSuccessMessage(fmt.Sprintf("Pre-configured GlobalCIDR %s detected. Not changing it.", globalnetCIDR))
		} else {
			// globalCidr as specified by the user
			err := CheckOverlappingCidr(globalNetworks, global, netconfig)
			if err != nil {
				return "", fmt.Errorf("Error validating overlapping GlobalCIDRs %s %s", globalnetCIDR, err)
			}
			status.QueueSuccessMessage(fmt.Sprintf("GlobalCIDR is: %s", globalnetCIDR))
		}
	}
	status.End(true)
	return globalnetCIDR, nil
}

func CheckForOverlappingCIDRs(globalNetworks map[string]*GlobalNetwork, netconfig Config) error {
	err := CheckOverlappingCidr(globalNetworks, service, netconfig)
	if err != nil {
		return err
	}
	err = CheckOverlappingCidr(globalNetworks, cluster, netconfig)
	if err != nil {
		return err
	}
	return nil
}
