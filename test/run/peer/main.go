package main

import (
	"fmt"
	"hpudp"
	"net"
	"net/netip"
)

func main() {
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 9751})
	if err != nil {
		fmt.Println(err)
		return
	}
	ip, err := netip.ParseAddr("127.0.0.1")
	if err != nil {
		fmt.Println(err)
		return
	}
	i := netip.AddrPortFrom(ip, uint16(8751))
	tracker := net.UDPAddrFromAddrPort(i)

	peer := hpudp.NewPeer(conn, tracker)
	fmt.Println(peer)
}
