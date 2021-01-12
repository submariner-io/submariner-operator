/*
© 2021 Red Hat, Inc. and others.

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
	"encoding/json"
	"errors"
	"fmt"
	"math/bits"
	"net"

	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/submariner-io/submariner-operator/pkg/broker"
	"github.com/submariner-io/submariner-operator/pkg/internal/cli"
)

type GlobalnetInfo struct {
	GlobalnetEnabled     bool
	GlobalnetCidrRange   string
	GlobalnetClusterSize uint
	GlobalCidrInfo       map[string]*GlobalNetwork
}

type GlobalNetwork struct {
	GlobalCIDRs []string
	ClusterId   string
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
	ClusterCIDR             string
	ClusterID               string
	GlobalnetCIDR           string
	ServiceCIDR             string
	GlobalnetClusterSize    uint
	ClusterCIDRAutoDetected bool
	ServiceCIDRAutoDetected bool
}

var globalCidr = GlobalCIDR{allocatedCount: 0}
var status = cli.NewStatus()

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

func NewCIDR(cidr string) (CIDR, error) {
	_, network, err := net.ParseCIDR(cidr)
	if err != nil {
		return CIDR{}, fmt.Errorf("invalid cidr %q passed as input", cidr)
	}
	ones, total := network.Mask.Size()
	size := total - ones
	lastIp := LastIp(network)
	clusterCidr := CIDR{network: network, size: size, lastIp: lastIp}
	return clusterCidr, nil
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

	var clusterCidr CIDR
	if clusterCidr, err = NewCIDR(cidr); err != nil {
		return 0, err
	}

	if !globalCidr.net.Contains(uintToIP(clusterCidr.lastIp)) {
		return 0, fmt.Errorf("%s not a valid subnet of %v\n", cidr, globalCidr.net)
	}
	for i := 0; i < globalCidr.allocatedCount; i++ {
		allocated := globalCidr.allocatedClusters[i]
		if allocated.network.Contains(requestedIp) {
			// subset of already allocated, try next
			return allocated.lastIp, fmt.Errorf("%s subset of already allocated globalCidr %v\n", cidr, allocated.network)
		}
		if requestedNetwork.Contains(allocated.network.IP) {
			// already allocated is subset of requested, no valid lastIp
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

func AllocateGlobalCIDR(globalnetInfo *GlobalnetInfo) (string, error) {
	globalCidr = GlobalCIDR{allocatedCount: 0, cidr: globalnetInfo.GlobalnetCidrRange}
	_, network, err := net.ParseCIDR(globalCidr.cidr)
	if err != nil {
		return "", fmt.Errorf("invalid GlobalCIDR %s configured", globalCidr.cidr)
	}
	globalCidr.net = network
	for _, globalNetwork := range globalnetInfo.GlobalCidrInfo {
		for _, otherCluster := range globalNetwork.GlobalCIDRs {
			otherClusterCIDR, err := NewCIDR(otherCluster)
			if err != nil {
				return "", err
			}
			globalCidr.allocatedClusters = append(globalCidr.allocatedClusters, &otherClusterCIDR)
			globalCidr.allocatedCount++
		}
	}
	return allocateByClusterSize(globalnetInfo.GlobalnetClusterSize)
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

func CheckOverlappingCidrs(globalnetInfo *GlobalnetInfo, netconfig Config) error {
	var cidrlist []string
	var cidr string
	for k, v := range globalnetInfo.GlobalCidrInfo {
		cidrlist = v.GlobalCIDRs
		cidr = netconfig.GlobalnetCIDR
		overlap, err := isOverlappingCIDR(cidrlist, cidr)
		if err != nil {
			return fmt.Errorf("unable to validate overlapping CIDR: %s", err)
		}
		if overlap && k != netconfig.ClusterID {
			return fmt.Errorf("invalid CIDR %s overlaps with cluster %q", cidr, k)
		}
	}
	return nil
}

func isCIDRPreConfigured(clusterID string, globalNetworks map[string]*GlobalNetwork) bool {
	// GlobalCIDR is not pre-configured
	if globalNetworks[clusterID] == nil || globalNetworks[clusterID].GlobalCIDRs == nil || len(globalNetworks[clusterID].GlobalCIDRs) == 0 {
		return false
	}
	// GlobalCIDR is pre-configured
	return true
}

func ValidateGlobalnetConfiguration(globalnetInfo *GlobalnetInfo, netconfig Config) (string, error) {
	status.Start("Validating Globalnet configurations")
	globalnetClusterSize := netconfig.GlobalnetClusterSize
	globalnetCIDR := netconfig.GlobalnetCIDR
	if globalnetInfo.GlobalnetEnabled && globalnetClusterSize != 0 && globalnetClusterSize != globalnetInfo.GlobalnetClusterSize {
		clusterSize, err := GetValidClusterSize(globalnetInfo.GlobalnetCidrRange, globalnetClusterSize)
		if err != nil || clusterSize == 0 {
			return "", fmt.Errorf("invalid globalnet-cluster-size %s", err)
		}
		globalnetInfo.GlobalnetClusterSize = clusterSize
	}

	if globalnetCIDR != "" && globalnetClusterSize != 0 {
		err := errors.New("Both globalnet-cluster-size and globalnet-cidr can't be specified. Specify either one.\n")
		return "", fmt.Errorf("%s", err)
	}

	if globalnetCIDR != "" {
		_, _, err := net.ParseCIDR(globalnetCIDR)
		if err != nil {
			return "", fmt.Errorf("specified globalnet-cidr is invalid: %s", err)
		}
	}

	if !globalnetInfo.GlobalnetEnabled {
		if globalnetCIDR != "" {
			status.QueueWarningMessage("Globalnet is not enabled on Broker. Ignoring specified globalnet-cidr")
			globalnetCIDR = ""
		} else if globalnetClusterSize != 0 {
			status.QueueWarningMessage("Globalnet is not enabled on Broker. Ignoring specified globalnet-cluster-size")
			globalnetInfo.GlobalnetClusterSize = 0
		}
	}
	status.End(cli.Success)
	return globalnetCIDR, nil
}

func GetGlobalNetworks(k8sClientset *kubernetes.Clientset, brokerNamespace string) (*GlobalnetInfo, *v1.ConfigMap, error) {
	configMap, err := broker.GetGlobalnetConfigMap(k8sClientset, brokerNamespace)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading configMap: %s", err)
	}

	globalnetInfo := GlobalnetInfo{}
	err = json.Unmarshal([]byte(configMap.Data[broker.GlobalnetStatusKey]), &globalnetInfo.GlobalnetEnabled)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading globalnetEnabled status: %s", err)
	}

	if globalnetInfo.GlobalnetEnabled {
		err = json.Unmarshal([]byte(configMap.Data[broker.GlobalnetClusterSize]), &globalnetInfo.GlobalnetClusterSize)
		if err != nil {
			return nil, nil, fmt.Errorf("error reading GlobalnetClusterSize: %s", err)
		}

		err = json.Unmarshal([]byte(configMap.Data[broker.GlobalnetCidrRange]), &globalnetInfo.GlobalnetCidrRange)
		if err != nil {
			return nil, nil, fmt.Errorf("error reading GlobalnetCidrRange: --> %s", err)
		}
	}

	var clusterInfo []broker.ClusterInfo
	err = json.Unmarshal([]byte(configMap.Data[broker.ClusterInfoKey]), &clusterInfo)
	if err != nil {
		return nil, nil, fmt.Errorf("error reading globalnet clusterInfo: %s", err)
	}

	var globalNetworks = make(map[string]*GlobalNetwork)
	for _, cluster := range clusterInfo {
		globalNetwork := GlobalNetwork{
			GlobalCIDRs: cluster.GlobalCidr,
			ClusterId:   cluster.ClusterId,
		}
		globalNetworks[cluster.ClusterId] = &globalNetwork
	}

	globalnetInfo.GlobalCidrInfo = globalNetworks
	return &globalnetInfo, configMap, nil
}

func AssignGlobalnetIPs(globalnetInfo *GlobalnetInfo, netconfig Config) (string, error) {
	status.Start("Assigning Globalnet IPs")
	globalnetCIDR := netconfig.GlobalnetCIDR
	clusterID := netconfig.ClusterID
	var err error
	if globalnetCIDR == "" {
		// Globalnet enabled, GlobalCIDR not specified by the user
		if isCIDRPreConfigured(clusterID, globalnetInfo.GlobalCidrInfo) {
			// globalCidr already configured on this cluster
			globalnetCIDR = globalnetInfo.GlobalCidrInfo[clusterID].GlobalCIDRs[0]
			status.QueueWarningMessage(fmt.Sprintf("Cluster already has GlobalCIDR allocated: %s", globalnetCIDR))
		} else {
			// no globalCidr configured on this cluster
			globalnetCIDR, err = AllocateGlobalCIDR(globalnetInfo)
			if err != nil {
				return "", fmt.Errorf("Globalnet failed %s", err)
			}
			status.QueueSuccessMessage(fmt.Sprintf("Allocated GlobalCIDR: %s", globalnetCIDR))
		}
	} else {
		// Globalnet enabled, globalnetCIDR specified by user
		if isCIDRPreConfigured(clusterID, globalnetInfo.GlobalCidrInfo) {
			// globalCidr pre-configured on this cluster
			globalnetCIDR = globalnetInfo.GlobalCidrInfo[clusterID].GlobalCIDRs[0]
			status.QueueWarningMessage(fmt.Sprintf("Pre-configured GlobalCIDR %s detected. Not changing it.", globalnetCIDR))
		} else {
			// globalCidr as specified by the user
			err := CheckOverlappingCidrs(globalnetInfo, netconfig)
			if err != nil {
				return "", fmt.Errorf("error validating overlapping GlobalCIDRs %s: %s", globalnetCIDR, err)
			}
			status.QueueSuccessMessage(fmt.Sprintf("GlobalCIDR is: %s", globalnetCIDR))
		}
	}
	status.End(cli.Success)
	return globalnetCIDR, nil
}
