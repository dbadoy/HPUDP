package hpudp

import (
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"sync"
	"time"
)

const (
	PingType = 1 + iota
	NotificationType
)

type Beater struct {
	conn    *net.UDPConn
	pingRes map[string]bool

	d  chan BroadRequest
	mu sync.Mutex
}

func NewBeater(conn *net.UDPConn) *Beater {
	return &Beater{
		conn:    conn,
		pingRes: make(map[string]bool),
		d:       make(chan BroadRequest),
	}
}

func (b *Beater) Put(r BroadRequest) {
	b.d <- r
}

func (b *Beater) Register(addr *net.UDPAddr) {
	if b.pingRes[addr.String()] {
		return
	}
	b.pingRes[addr.String()] = false
}

func (b *Beater) snap() (r []string) {
	b.mu.Lock()
	r = make([]string, len(b.pingRes))
	for raw, _ := range b.pingRes {
		r = append(r, raw)
	}
	b.mu.Unlock()
	return
}

func (b *Beater) Broadcast(t byte) {
	targets := b.snap()

	addrs := make([]*net.UDPAddr, len(targets))
	for _, target := range targets {
		addrs = append(addrs, rawAddrToUDPAddr(target))
	}
	if len(addrs) == 0 {
		return
	}
	switch t {
	case PingType:
		b.ping(addrs)
	case NotificationType:
		b.notification(addrs)
	}
}

func (b *Beater) ping(addrs []*net.UDPAddr) {
	for _, addr := range addrs {
		packet := new(PingPacket)
		packet.SetKind(Ping)
		byt, _ := json.Marshal(packet)

		if _, err := b.conn.WriteToUDP(byt, addr); err != nil {
			// TODO
			break
		}
	}

	timer := time.NewTimer(5 * time.Second)
	for {
		select {
		case <-timer.C:
			break
		case r := <-b.d:
			if r.P.Kind() == Pong {
				rawAddr := (*r.Sender).String()
				b.pingRes[rawAddr] = true
			}
		}
	}
}

func (b *Beater) notification(addrs []*net.UDPAddr) {
	fmt.Println("not implements")
}

func rawAddrToUDPAddr(s string) *net.UDPAddr {
	rawAddr := netip.MustParseAddrPort(s)
	return net.UDPAddrFromAddrPort(rawAddr)
}
