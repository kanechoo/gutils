package gnet

import (
	"errors"
	"github.com/ylwangs/go-mtr/mtr"
	"log"
	"net"
	"sort"
	"strconv"
	"strings"
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

// NetSplit 拆分CIDR为IP列表
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

// IPInNet 检测IP是否在网络中
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

// MTR 检测网络路径
// destAddr 目标地址
// maxHops 最大跳数 sntSize 发送数据包数量 timeoutMs 超时时间(毫秒)
func MTR[T string | net.IP](destAddr T, maxHops int, sntSize int, timeoutMs int) *[]MTRResult {
	var ip string
	switch value := any(destAddr).(type) {
	case string:
		ip = value
	case net.IP:
		ip = value.String()
	default:
		panic("destAddr type is not string or net.IP")
	}
	result, err := mtr.Mtr(ip, maxHops, sntSize, timeoutMs)
	if err != nil {
		log.Printf("mtr failed: %v", err)
		return nil
	}
	return parseMtrResult(result)
}

func parseMtrResult(result string) *[]MTRResult {
	if result == "" {
		return nil
	}
	lines := strings.Split(result, "\n")
	if len(lines) < 3 {
		return nil
	}
	var list []MTRResult
	for i := 2; i < len(lines); i++ {
		line := lines[i]
		if strings.TrimSpace(line) == "" {
			continue
		}
		line += " "
		word := ""
		var r MTRResult
		mark := 1
		for _, character := range strings.Split(line, "") {
			if character != " " {
				word += character
			} else {
				word = strings.TrimSpace(word)
				if word == "" {
					word = ""
					continue
				}
				switch mark {
				case 1:
					sq, err := strconv.Atoi(word)
					if err == nil {
						r.Sequence = sq
					}
				case 2:
					r.Host = word
				case 3:
					r.Loss = word
				case 4:
					snt, err := strconv.Atoi(word)
					if err == nil {
						r.Snt = snt
					}
				case 5:
					last, err := strconv.ParseFloat(word, 64)
					if err == nil {
						r.Last = last
					}
				case 6:
					avg, err := strconv.ParseFloat(word, 64)
					if err == nil {
						r.Avg = avg
					}
				case 7:
					best, err := strconv.ParseFloat(word, 64)
					if err == nil {
						r.Best = best
					}
				case 8:
					wrst, err := strconv.ParseFloat(word, 64)
					if err == nil {
						r.Wrst = wrst
					}
				default:
					break
				}
				mark++
				word = ""
			}
		}
		list = append(list, r)
	}
	return &list
}
func NetToNet[T net.IPNet | string](cidr T, mask int) *[]net.IPNet {
	if mask > 32 || mask < 0 {
		log.Printf("mask is out of range,should be 0-32")
		return nil
	}
	var c net.IPNet
	switch v := any(cidr).(type) {
	case string:
		_, c1, err := net.ParseCIDR(v)
		if err != nil {
			log.Printf("parse cidr error : %v", err)
			return nil
		}
		c = *c1
	case net.IPNet:
		c = v
	}
	m1, _ := c.Mask.Size()
	if m1 > mask {
		//当前网络范围小于目标网络范围，无法转换
		return nil
	} else if m1 == mask {
		return &[]net.IPNet{c}
	}
	var list []net.IPNet
	cidrMap := make(map[string]interface{})
	for ip := c.IP.Mask(c.Mask); c.Contains(ip); inc(ip) {
		_, n, err := net.ParseCIDR(ip.String() + "/" + strconv.Itoa(mask))
		if err != nil {
			continue
		}
		cidrMap[n.String()] = nil
	}
	for c1 := range cidrMap {
		_, n, err := net.ParseCIDR(c1)
		if err != nil {
			continue
		}
		list = append(list, *n)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].String() < list[j].String()
	})
	return &list
}
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
