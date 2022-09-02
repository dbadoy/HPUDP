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
	conn *net.UDPConn

	// 'peers' mapping rawAddress to health status.
	// The health status set when the ping-pong request completes
	peers map[string]bool

	d  chan BroadResponse
	mu sync.Mutex
}

func NewBeater(conn *net.UDPConn) Beater {
	return Beater{
		conn:  conn,
		peers: make(map[string]bool),
		d:     make(chan BroadResponse, 512),
	}
}

// TODO: Is it best?
func (b *Beater) Put(r BroadResponse) {
	b.d <- r
}

func (b *Beater) Register(addr *net.UDPAddr) {
	if b.peers[addr.String()] {
		return
	}
	b.peers[addr.String()] = false
}

func (b *Beater) Unregister(addr *net.UDPAddr) {
	delete(b.peers, addr.String())
}

func (b *Beater) BroadcastPing(timeout time.Duration) {
	b.broadcastPing(timeout)
}

func (b *Beater) BroadcastPingWithTicker(ticker time.Ticker, per time.Duration) chan struct{} {
	var cancel chan struct{}
	go func() {
		for {
			select {
			case <-ticker.C:
				// If 'per' greater than ticket duration, ticker wait broadcasePing done.
				// Do not call broadcastPing by goroutine. If you use goroutine, will accumulate
				// meaningless running goroutines.
				b.broadcastPing(per)
			case <-cancel:
				return
			}
		}
	}()
	return cancel
}

// Do not call by goroutine. It's running it once is enough.
func (b *Beater) broadcastPing(timeout time.Duration) {
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	go b.broadcast(PingType, timeout)
}

func (b *Beater) IsAlive(raw string) bool {
	return b.snapPingTable()[raw]
}

func (b *Beater) PingTable() map[string]bool {
	return b.snapPingTable()
}

func (b *Beater) snapTargets() (r []string) {
	b.mu.Lock()
	r = make([]string, 0, len(b.peers)/2)
	for raw, _ := range b.peers {
		r = append(r, raw)
	}
	b.mu.Unlock()
	return
}

func (b *Beater) snapPingTable() (r map[string]bool) {
	b.mu.Lock()
	r = make(map[string]bool, len(b.peers))
	// It really need deep copy ?
	for addr, health := range b.peers {
		r[addr] = health
	}
	b.mu.Unlock()
	return
}

func (b *Beater) broadcast(t byte, timeout time.Duration) {
	targets := b.snapTargets()

	addrs := make([]*net.UDPAddr, 0, len(targets))
	for _, target := range targets {
		addrs = append(addrs, rawAddrToUDPAddr(target))
	}
	if len(addrs) == 0 {
		fmt.Println("beater: there is no target")
		return
	}
	switch t {
	case PingType:
		b.ping(addrs, timeout)
	case NotificationType:
		b.notification(addrs)
	}
}

func (b *Beater) ping(addrs []*net.UDPAddr, timeout time.Duration) {
	for _, addr := range addrs {
		// This can be use goroutine.
		// I don't know better way about point of performance. Need basis.
		packet := new(PingPacket)
		packet.SetKind(Ping)
		byt, _ := json.Marshal(&packet)

		if _, err := b.conn.WriteToUDP(byt, addr); err != nil {
			fmt.Println(err)
			break
		}
	}

	tempSnapTable := b.snapPingTable()
	timer := time.NewTimer(timeout)
	for {
		select {
		case <-timer.C:
			for rawAdrr := range tempSnapTable {
				b.mu.Lock()
				b.peers[rawAdrr] = false
				b.mu.Unlock()
			}
			fmt.Println(b.PingTable())
			return
		case r := <-b.d:
			if r.P.Kind() == Pong {
				rawAddr := (*r.Sender).String()
				b.peers[rawAddr] = true
				delete(tempSnapTable, rawAddr)
			}
		}
	}
}

func (b *Beater) notification(addrs []*net.UDPAddr) {
	fmt.Println("not implements")
}

// [Benchmark]
//		net.ResolveUDPAddr									10000000                151.0 ns/op
//		netip.MustParseAddrPort, net.UDPAddrFromAddrPort	10000000                62.55 ns/op
func rawAddrToUDPAddr(s string) *net.UDPAddr {
	rawAddr := netip.MustParseAddrPort(s)
	return net.UDPAddrFromAddrPort(rawAddr)
}
