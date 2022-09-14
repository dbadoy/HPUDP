package main

import (
	"flag"
	"fmt"
	"net"
	"net/netip"

	"hpudp"
)

type Config struct {
	ip   string
	port int
	role string
	nick string
}

func main() {
	var cfg Config

	flag.StringVar(&cfg.ip, "ip", "127.0.0.1", "this is the something blah blah")
	flag.IntVar(&cfg.port, "port", 8751, "")
	flag.StringVar(&cfg.role, "role", "peer", "")
	flag.StringVar(&cfg.nick, "name", "peer", "")

	flag.Parse()

	ip, err := netip.ParseAddr(cfg.ip)
	if err != nil {
		fmt.Println(err)
		return
	}
	i := netip.AddrPortFrom(ip, uint16(cfg.port))
	conn := net.UDPAddrFromAddrPort(i)
	server := hpudp.NewServer(conn)
	if server != nil {
		server.Start()
	}
}
