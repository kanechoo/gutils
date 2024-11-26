package gnet

import (
	"errors"
	"net"
)

// IPToNet 转换IP和掩码为CIDR格式，mask为子网掩码长度
// mask 范围为0-32
func IPToNet[T string | net.IP](ip T, mask uint8) (net.IPNet, error) {
	if mask > 32 {
		return net.IPNet{}, errors.New("mask is out of range,should be 0-32")
	}
	switch ipType := any(ip).(type) {
	case string:
		parseIP := net.ParseIP(ipType)
		return doIPToCidr(parseIP, mask)
	case net.IP:
		return doIPToCidr(ipType, mask)
	default:
		panic("ip is not string or net.IP")
	}
}

func doIPToCidr(ip net.IP, mask uint8) (net.IPNet, error) {
	cidr := net.IPNet{
		IP:   ip,
		Mask: net.CIDRMask(int(mask), 32),
	}
	_, ipNet, err := net.ParseCIDR(cidr.String())
	if err != nil {
		return net.IPNet{}, err
	}
	return *ipNet, nil
}

func NetSplit[T string | net.IPNet](cidr T) *[]net.IP {
	switch value := any(cidr).(type) {
	case string:
		_, ipNet, err := net.ParseCIDR(value)
		if err != nil {
			return nil
		}
		return doCidrSplit(*ipNet)
	case net.IPNet:
		return doCidrSplit(value)
	default:
		panic("cidr type is not string or net.IPNet")
	}
}

func doCidrSplit(cidr net.IPNet) *[]net.IP {
	ips := make([]net.IP, 0)
	for ip := cidr.IP.Mask(cidr.Mask); cidr.Contains(ip); inc(ip) {
		ips = append(ips, net.IP{ip[0], ip[1], ip[2], ip[3]})
	}
	return &ips
}
func IPInNet[I string | net.IP, N string | net.IPNet](ip I, cidr N) bool {
	var i net.IP
	var n net.IPNet
	switch ipType := any(ip).(type) {
	case string:
		i = net.ParseIP(ipType)
	case net.IP:
		i = ipType
	default:
		panic("ip is not string or net.IP")
	}
	switch networkType := any(cidr).(type) {
	case string:
		_, c, err := net.ParseCIDR(networkType)
		if err != nil {
			return false
		}
		n = *c
	case net.IPNet:
		n = networkType
	}
	return n.Contains(i)
}
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
