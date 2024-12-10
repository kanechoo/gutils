package main

import (
	"fmt"
	"github.com/kanechoo/gutils/gasn"
)

func main() {
	//ip := "192.168.6.1"
	//cidr := "192.168.7.0/24"
	//
	////测试代码
	//fmt.Println("== Test IPToNet ==")
	//c, err := gnet.IPToNet("192.168.5.10", 24)
	//if err != nil {
	//	panic(err)
	//}
	//
	//fmt.Println("== Test IPToNet ==")
	//ips := gnet.NetSplit(cidr)
	//fmt.Printf("cidr : %s has %d ip\n", c.String(), len(*ips))
	//flag := gnet.IPInNet(ip, cidr)
	//if flag {
	//	fmt.Printf("ip %s in network %s\n", ip, cidr)
	//} else {
	//	fmt.Printf("ip %s not in network %s\n", ip, cidr)
	//}
	//
	//fmt.Println("== Test NetToNet ==")
	//testNetToNet := gnet.NetToNet("23.105.192.0/19", 24)
	//fmt.Println(*testNetToNet)
	//
	//fmt.Println("== Test MTR ==")
	//mtr := gnet.MTR("127.0.0.1", 15, 1, 200)
	//fmt.Println(*mtr)
	//
	//fmt.Println("== Test ByAsn ==")
	//client := gasn.New(gasn.WithRetryTimes(3))
	//asn, err := client.ByAsn(1299, gasn.OnlyAnnounced)
	//if err != nil {
	//	panic(err)
	//}
	//fmt.Printf("asn: %v\n", *asn)
	c2 := gasn.New(gasn.WithRetryTimes(3))
	country, err := c2.ByCountry("HK")
	if err != nil {
		panic(err)
	}
	fmt.Printf("country: %v\n", *country)
}
