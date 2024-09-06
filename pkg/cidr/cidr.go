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

package cidr

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math/bits"
	"net"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

const ClusterInfoKey = "clusterinfo"

type ClusterInfo struct {
	ClusterID string   `json:"cluster_id"`
	CIDRs     []string `json:"global_cidr"`
}

type Info struct {
	CIDR           string
	AllocationSize uint
	Clusters       map[string]*ClusterInfo
}

type allocationInfo struct {
	network *net.IPNet
	size    int
	lastIP  uint32
}

func unmarshalClusterInfo(fromConfigMap *corev1.ConfigMap) ([]ClusterInfo, error) {
	existingData := fromConfigMap.Data[ClusterInfoKey]
	if existingData == "" {
		existingData = "[]"
	}

	var clusterInfo []ClusterInfo

	err := json.Unmarshal([]byte(existingData), &clusterInfo)
	if err != nil {
		return nil, errors.Wrapf(err, "error unmarshalling %q data from ConfigMap %q", ClusterInfoKey, fromConfigMap.Name)
	}

	return clusterInfo, nil
}

func ExtractClusterInfo(fromConfigMap *corev1.ConfigMap) (map[string]*ClusterInfo, error) {
	clusterInfo, err := unmarshalClusterInfo(fromConfigMap)

	clusterInfoMap := make(map[string]*ClusterInfo)

	for _, info := range clusterInfo {
		clusterInfoMap[info.ClusterID] = &info
	}

	return clusterInfoMap, err
}

func AddClusterInfoData(toConfigMap *corev1.ConfigMap, newCluster ClusterInfo) error {
	var existingInfo []ClusterInfo

	if toConfigMap.Data == nil {
		toConfigMap.Data = map[string]string{}
	}

	existingData := toConfigMap.Data[ClusterInfoKey]
	if existingData == "" {
		existingData = "[]"
	}

	err := json.Unmarshal([]byte(existingData), &existingInfo)
	if err != nil {
		return errors.Wrapf(err, "error unmarshalling ClusterInfo")
	}

	exists := false

	for k, value := range existingInfo {
		if value.ClusterID == newCluster.ClusterID {
			existingInfo[k].CIDRs = newCluster.CIDRs
			exists = true
		}
	}

	if !exists {
		existingInfo = append(existingInfo, newCluster)
	}

	data, err := json.MarshalIndent(existingInfo, "", "\t")
	if err != nil {
		return errors.Wrapf(err, "error marshalling ClusterInfo")
	}

	toConfigMap.Data[ClusterInfoKey] = string(data)

	return nil
}

func IsValid(cidr string) error {
	ip, _, err := net.ParseCIDR(cidr)
	if err != nil {
		return err //nolint:wrapcheck // No need to wrap here
	}

	if ip.IsUnspecified() {
		return fmt.Errorf("%s can't be unspecified", cidr)
	}

	if ip.IsLoopback() {
		return fmt.Errorf("%s can't be in loopback range", cidr)
	}

	if ip.IsLinkLocalUnicast() {
		return fmt.Errorf("%s can't be in link-local range", cidr)
	}

	if ip.IsLinkLocalMulticast() {
		return fmt.Errorf("%s can't be in link-local multicast range", cidr)
	}

	return nil
}

func CheckForOverlappingCIDRs(infoMap map[string]*ClusterInfo, cidr, clusterID string) error {
	for _, ci := range infoMap {
		overlap, err := isOverlappingCIDR(ci.CIDRs, cidr)
		if err != nil {
			return errors.Wrap(err, "unable to validate overlapping CIDRs")
		}

		if overlap && ci.ClusterID != clusterID {
			return fmt.Errorf("invalid CIDR %q overlaps with cluster %q", cidr, ci.ClusterID)
		}
	}

	return nil
}

func Allocate(info *Info) (string, error) {
	_, network, err := net.ParseCIDR(info.CIDR)
	if err != nil {
		return "", fmt.Errorf("unable to parse CIDR %q", info.CIDR)
	}

	var allocated []allocationInfo

	for _, cluster := range info.Clusters {
		for _, cidr := range cluster.CIDRs {
			_, n, err := net.ParseCIDR(cidr)
			if err != nil {
				return "", fmt.Errorf("unable to parse CIDR %q", cidr)
			}

			allocated = append(allocated, newAllocationInfo(n))
		}
	}

	return allocateBySize(info.AllocationSize, network, allocated)
}

func allocateBySize(size uint, network *net.IPNet, allocated []allocationInfo) (string, error) {
	bitSize := bits.LeadingZeros(0) - bits.LeadingZeros(size-1)
	_, totalbits := network.Mask.Size()
	clusterPrefix := totalbits - bitSize
	mask := net.CIDRMask(clusterPrefix, totalbits)

	cidr := fmt.Sprintf("%s/%d", network.IP, clusterPrefix)

	for {
		last, err := allocateByCIDR(cidr, network, allocated)
		if err == nil {
			break
		}

		if last == 0 {
			return "", err
		}

		nextNet := net.IPNet{
			IP:   uintToIP(last + 1),
			Mask: mask,
		}

		cidr = nextNet.String()
	}

	return cidr, nil
}

func allocateByCIDR(cidr string, network *net.IPNet, allocatedCIDRs []allocationInfo) (uint32, error) {
	requestedIP, requestedNetwork, _ := net.ParseCIDR(cidr)
	if !network.Contains(requestedIP) {
		return 0, fmt.Errorf("no more allocations available in %q", network)
	}

	newAllocation := newAllocationInfo(requestedNetwork)
	if !network.Contains(uintToIP(newAllocation.lastIP)) {
		return 0, fmt.Errorf("%s not a valid subnet of %v", uintToIP(newAllocation.lastIP), network)
	}

	for _, allocated := range allocatedCIDRs {
		if allocated.network.Contains(requestedIP) {
			// subset of already allocated, try next
			return allocated.lastIP, fmt.Errorf("%s is a subset of already allocated CIDR %v", cidr, allocated.network)
		}

		if requestedNetwork.Contains(allocated.network.IP) {
			// already allocated is subset of requested, no valid lastIP
			return newAllocation.lastIP, fmt.Errorf("%s overlaps with already allocated globalCidr %s", cidr, allocated.network)
		}
	}

	return 0, nil
}

func newAllocationInfo(network *net.IPNet) allocationInfo {
	ones, total := network.Mask.Size()
	size := total - ones

	return allocationInfo{
		network: network,
		size:    size,
		lastIP:  ipToUint(network.IP) + 1<<size - 1,
	}
}

func isOverlappingCIDR(cidrList []string, cidr string) (bool, error) {
	_, newNet, err := net.ParseCIDR(cidr)
	if err != nil {
		return false, err //nolint:wrapcheck // No need to wrap here
	}

	for _, v := range cidrList {
		_, baseNet, err := net.ParseCIDR(v)
		if err != nil {
			return false, err //nolint:wrapcheck // No need to wrap here
		}

		if baseNet.Contains(newNet.IP) || newNet.Contains(baseNet.IP) {
			return true, nil
		}
	}

	return false, nil
}

func ipToUint(ip net.IP) uint32 {
	intIP := ip
	if len(ip) == 16 {
		intIP = ip[12:16]
	}

	return binary.BigEndian.Uint32(intIP)
}

func uintToIP(ip uint32) net.IP {
	netIP := make(net.IP, 4)
	binary.BigEndian.PutUint32(netIP, ip)

	return netIP
}
