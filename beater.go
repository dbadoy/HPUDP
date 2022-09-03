package hpudp

import (
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

// Register is register to broadcast list that input address.
func (b *Beater) Register(addr *net.UDPAddr) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.peers[addr.String()] {
		return
	}
	b.peers[addr.String()] = false
}

// Unregister is remove input address from broadcast list.
func (b *Beater) Unregister(addr *net.UDPAddr) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.peers, addr.String())
}

func (b *Beater) Ping(target *net.UDPAddr, timeout time.Duration) bool {
	b.ping([]*net.UDPAddr{target}, timeout)
	return b.IsAlive(target.String())
}

// Send broadcast only once.
func (b *Beater) BroadcastPing(timeout time.Duration) {
	b.broadcastPing(timeout)
}

// Send broadcast with ticker.
func (b *Beater) BroadcastPingWithTicker(ticker time.Ticker, per time.Duration) chan struct{} {
	var cancel chan struct{}
	go func() {
		for {
			select {
			case <-ticker.C:
				// If 'per' greater than ticker duration, ticker wait broadcasePing done.
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
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.peers[raw]
}

func (b *Beater) PingTable() map[string]bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.snapPingTable()
}

// snapPingTable returns deep copied snapshot about broadcast list.
//
// The caller must hold b.mu.
func (b *Beater) snapPingTable() (r map[string]bool) {
	r = make(map[string]bool, len(b.peers))
	for addr, health := range b.peers {
		r[addr] = health
	}
	return
}

func (b *Beater) broadcast(t byte, timeout time.Duration) {
	b.mu.Lock()
	addrs := make([]*net.UDPAddr, 0, len(b.peers))
	for target := range b.peers {
		addrs = append(addrs, rawAddrToUDPAddr(target))
	}
	b.mu.Unlock()
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
		byt, _ := SuitablePack(packet)

		if _, err := b.conn.WriteToUDP(byt, addr); err != nil {
			fmt.Println(err)
			break
		}
	}

	// The snapshot that marking the changed peer status. If get 'pong',
	// remove sender from snapshot. This means that peers that did not
	// send a response to the PING remain in the snapshot.
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
			// Logging for test.
			fmt.Println(b.snapPingTable())
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
