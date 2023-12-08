package cni

import (
	"encoding/json"
	"fmt"
	"net"
	"runtime"
	"strings"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	current "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/vishvananda/netlink"

	kubeovnv1 "github.com/kubeovn/kube-ovn/pkg/apis/kubeovn/v1"
	"github.com/kubeovn/kube-ovn/pkg/request"
	"github.com/kubeovn/kube-ovn/pkg/util"
	"github.com/kubeovn/kube-ovn/versions"
)

func CmdMain() {
	// this ensures that main runs only on main thread (thread group leader).
	// since namespace ops (unshare, setns) are done for a single thread, we
	// must ensure that the goroutine does not jump from OS thread to thread
	runtime.LockOSThread()

	about := fmt.Sprintf("CNI kube-ovn plugin %s", versions.VERSION)
	skel.PluginMain(cmdAdd, nil, cmdDel, version.All, about)
}

func cmdAdd(args *skel.CmdArgs) error {
	netConf, cniVersion, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}
	podName, err := parseValueFromArgs("K8S_POD_NAME", args.Args)
	if err != nil {
		return err
	}
	podNamespace, err := parseValueFromArgs("K8S_POD_NAMESPACE", args.Args)
	if err != nil {
		return err
	}
	if netConf.Provider == "" && netConf.Type == util.CniTypeName && args.IfName == "eth0" {
		netConf.Provider = util.OvnProvider
	}

	client := request.NewCniServerClient(netConf.ServerSocket)
	response, err := client.Add(request.CniRequest{
		CniType:                   netConf.Type,
		PodName:                   podName,
		PodNamespace:              podNamespace,
		ContainerID:               args.ContainerID,
		NetNs:                     args.Netns,
		IfName:                    args.IfName,
		Provider:                  netConf.Provider,
		Routes:                    netConf.Routes,
		DNS:                       netConf.DNS,
		DeviceID:                  netConf.DeviceID,
		VfDriver:                  netConf.VfDriver,
		VhostUserSocketVolumeName: netConf.VhostUserSocketVolumeName,
		VhostUserSocketName:       netConf.VhostUserSocketName,
	})
	if err != nil {
		return types.NewError(types.ErrTryAgainLater, "RPC failed", err.Error())
	}

	if response.InterfaceType == "virtio" || response.InterfaceType == "macvtap" {
		err = SetDriver(netConf.DeviceID, Virtio)
		if err != nil {
			return err
		}

		ifaceName := getVirtioInterfaceName(netConf.DeviceID)
		if ifaceName == "" {
			return fmt.Errorf("get virtio interface name timeout, deviceID %s", netConf.DeviceID)
		}

		macAddr, err := net.ParseMAC(response.MacAddress)
		if err != nil {
			return fmt.Errorf("failed to parse mac %s %v", macAddr, err)
		}

		vfLink, err := netlink.LinkByName(ifaceName)
		if err != nil {
			return fmt.Errorf("failed to lookup link %s: %v", ifaceName, err)
		}
		if err = netlink.LinkSetHardwareAddr(vfLink, macAddr); err != nil {
			return fmt.Errorf("can not set mac address to vf nic:%s  %v", ifaceName, err)
		}
		if err := netlink.LinkSetAlias(vfLink, ifaceName); err != nil {
			return fmt.Errorf("failed to set link alias for container nic %s: %v", vfLink, err)
		}

		podNS, err := ns.GetNS(args.Netns)
		if err != nil {
			return fmt.Errorf("failed to open netns %q: %v", args.Netns, err)
		}

		ipAddr, err := netlink.ParseAddr(util.GetIpAddrWithMask(response.IpAddress, response.CIDR))
		if err != nil {
			return fmt.Errorf("failed to parse ip addr %q: %v", response.IpAddress, err)
		}

		if response.InterfaceType == "virtio" {
			if err = netlink.LinkSetNsFd(vfLink, int(podNS.Fd())); err != nil {
				return fmt.Errorf("failed to move link to netns: %v", err)
			}

			if err = ns.WithNetNSPath(podNS.Path(), func(_ ns.NetNS) error {
				err := netlink.AddrAdd(vfLink, ipAddr)
				if err != nil {
					return fmt.Errorf("failed to add ip addr: %v", err)
				}

				if err := netlink.LinkSetUp(vfLink); err != nil {
					return fmt.Errorf("failed to set link up ip: %v", err)
				}

				return nil
			}); err != nil {
				return fmt.Errorf("failed to config in ns: %v", err)
			}
		} else {
			macvtapLink := &netlink.Macvtap{
				Macvlan: netlink.Macvlan{
					LinkAttrs: netlink.LinkAttrs{Name: ifaceName + "-macvtap", ParentIndex: vfLink.Attrs().Index},
					Mode:      netlink.MACVLAN_MODE_PASSTHRU,
				},
			}
			if err := netlink.LinkAdd(macvtapLink); err != nil {
				return fmt.Errorf("failed to add macvtap link: %v", err)
			}
			if err = netlink.LinkSetNsFd(macvtapLink, int(podNS.Fd())); err != nil {
				return fmt.Errorf("failed to move link to netns: %v", err)
			}

			if err = ns.WithNetNSPath(podNS.Path(), func(_ ns.NetNS) error {
				err := netlink.AddrAdd(macvtapLink, ipAddr)
				if err != nil {
					return fmt.Errorf("failed to add ip addr: %v", err)
				}

				if err := netlink.LinkSetUp(macvtapLink); err != nil {
					return fmt.Errorf("failed to set link up ip: %v", err)
				}

				return nil
			}); err != nil {
				return fmt.Errorf("failed to config in ns: %v", err)
			}

		}
	}

	result := generateCNIResult(response, args.Netns)
	return types.PrintResult(&result, cniVersion)
}

func generateCNIResult(cniResponse *request.CniResponse, netns string) current.Result {
	result := current.Result{
		CNIVersion: current.ImplementedSpecVersion,
		DNS:        cniResponse.DNS,
	}
	_, mask, _ := net.ParseCIDR(cniResponse.CIDR)
	podIface := current.Interface{
		Name:    cniResponse.PodNicName,
		Mac:     cniResponse.MacAddress,
		Sandbox: netns,
	}
	switch cniResponse.Protocol {
	case kubeovnv1.ProtocolIPv4:
		ip, route := assignV4Address(cniResponse.IpAddress, cniResponse.Gateway, mask)
		result.IPs = []*current.IPConfig{ip}
		if route != nil {
			result.Routes = []*types.Route{route}
		}
		result.Interfaces = []*current.Interface{&podIface}
	case kubeovnv1.ProtocolIPv6:
		ip, route := assignV6Address(cniResponse.IpAddress, cniResponse.Gateway, mask)
		result.IPs = []*current.IPConfig{ip}
		if route != nil {
			result.Routes = []*types.Route{route}
		}
		result.Interfaces = []*current.Interface{&podIface}
	case kubeovnv1.ProtocolDual:
		var netMask *net.IPNet
		var gwStr string
		for _, cidrBlock := range strings.Split(cniResponse.CIDR, ",") {
			_, netMask, _ = net.ParseCIDR(cidrBlock)
			gwStr = ""
			if util.CheckProtocol(cidrBlock) == kubeovnv1.ProtocolIPv4 {
				ipStr := strings.Split(cniResponse.IpAddress, ",")[0]
				if cniResponse.Gateway != "" {
					gwStr = strings.Split(cniResponse.Gateway, ",")[0]
				}

				ip, route := assignV4Address(ipStr, gwStr, netMask)
				result.IPs = append(result.IPs, ip)
				if route != nil {
					result.Routes = append(result.Routes, route)
				}
			} else if util.CheckProtocol(cidrBlock) == kubeovnv1.ProtocolIPv6 {
				ipStr := strings.Split(cniResponse.IpAddress, ",")[1]
				if cniResponse.Gateway != "" {
					gwStr = strings.Split(cniResponse.Gateway, ",")[1]
				}

				ip, route := assignV6Address(ipStr, gwStr, netMask)
				result.IPs = append(result.IPs, ip)
				if route != nil {
					result.Routes = append(result.Routes, route)
				}
			}
		}
		result.Interfaces = []*current.Interface{&podIface}
	}

	return result
}

func cmdDel(args *skel.CmdArgs) error {
	netConf, _, err := loadNetConf(args.StdinData)
	if err != nil {
		return err
	}

	if IsVirtioDevice(netConf.DeviceID) {
		err = SetDriver(netConf.DeviceID, Sriov)
		if err != nil {
			return err
		}
	}

	client := request.NewCniServerClient(netConf.ServerSocket)
	podName, err := parseValueFromArgs("K8S_POD_NAME", args.Args)
	if err != nil {
		return err
	}
	podNamespace, err := parseValueFromArgs("K8S_POD_NAMESPACE", args.Args)
	if err != nil {
		return err
	}
	if netConf.Type == util.CniTypeName && args.IfName == "eth0" {
		netConf.Provider = util.OvnProvider
	}

	err = client.Del(request.CniRequest{
		CniType:                   netConf.Type,
		PodName:                   podName,
		PodNamespace:              podNamespace,
		ContainerID:               args.ContainerID,
		NetNs:                     args.Netns,
		IfName:                    args.IfName,
		Provider:                  netConf.Provider,
		DeviceID:                  netConf.DeviceID,
		VhostUserSocketVolumeName: netConf.VhostUserSocketVolumeName,
	})
	if err != nil {
		return types.NewError(types.ErrTryAgainLater, "RPC failed", err.Error())
	}
	return nil
}

type ipamConf struct {
	ServerSocket string `json:"server_socket"`
	Provider     string `json:"provider"`
}

func loadNetConf(bytes []byte) (*netConf, string, error) {
	n := &netConf{}
	if err := json.Unmarshal(bytes, n); err != nil {
		return nil, "", types.NewError(types.ErrDecodingFailure, "failed to load netconf", err.Error())
	}

	if n.Type != util.CniTypeName && n.IPAM != nil {
		n.Provider = n.IPAM.Provider
		n.ServerSocket = n.IPAM.ServerSocket
	}

	if n.ServerSocket == "" {
		return nil, "", types.NewError(types.ErrInvalidNetworkConfig, "Invalid Configuration", fmt.Sprintf("server_socket is required in cni.conf, %+v", n))
	}

	if n.Provider == "" {
		n.Provider = util.OvnProvider
	}

	n.postLoad()
	return n, n.CNIVersion, nil
}

func parseValueFromArgs(key, argString string) (string, error) {
	if argString == "" {
		return "", types.NewError(types.ErrInvalidNetworkConfig, "Invalid Configuration", "CNI_ARGS is required")
	}
	args := strings.Split(argString, ";")
	for _, arg := range args {
		if strings.HasPrefix(arg, fmt.Sprintf("%s=", key)) {
			value := strings.TrimPrefix(arg, fmt.Sprintf("%s=", key))
			if len(value) > 0 {
				return value, nil
			}
		}
	}
	return "", types.NewError(types.ErrInvalidNetworkConfig, "Invalid Configuration", fmt.Sprintf("%s is required in CNI_ARGS", key))
}

func assignV4Address(ipAddress, gateway string, mask *net.IPNet) (*current.IPConfig, *types.Route) {
	ip := &current.IPConfig{
		Address:   net.IPNet{IP: net.ParseIP(ipAddress).To4(), Mask: mask.Mask},
		Gateway:   net.ParseIP(gateway).To4(),
		Interface: current.Int(0),
	}

	var route *types.Route
	if gw := net.ParseIP(gateway); gw != nil {
		route = &types.Route{
			Dst: net.IPNet{IP: net.IPv4zero.To4(), Mask: net.CIDRMask(0, 32)},
			GW:  net.ParseIP(gateway).To4(),
		}
	}

	return ip, route
}

func assignV6Address(ipAddress, gateway string, mask *net.IPNet) (*current.IPConfig, *types.Route) {
	ip := &current.IPConfig{
		Address:   net.IPNet{IP: net.ParseIP(ipAddress).To16(), Mask: mask.Mask},
		Gateway:   net.ParseIP(gateway).To16(),
		Interface: current.Int(0),
	}

	var route *types.Route
	if gw := net.ParseIP(gateway); gw != nil {
		route = &types.Route{
			Dst: net.IPNet{IP: net.IPv6zero, Mask: net.CIDRMask(0, 128)},
			GW:  net.ParseIP(gateway).To16(),
		}
	}

	return ip, route
}
