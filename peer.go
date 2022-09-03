package hpudp

import (
	"fmt"
	"net"
	"sync"
	"time"
)

type Peer struct {
	userName string

	table map[string]*net.UDPAddr

	conn *net.UDPConn

	isRun   uint32
	tracker *net.UDPAddr

	beater Beater

	done chan struct{}

	wg sync.WaitGroup
}

func NewPeer(conn *net.UDPConn, tracker *net.UDPAddr) *Peer {
	p := &Peer{
		conn:    conn,
		tracker: tracker,
		beater:  NewBeater(conn),
		done:    make(chan struct{}),
	}
	go p.tempLoop()
	go p.trackerCheck(3 * time.Second)
	<-p.done
	return p
}

func (p *Peer) trackerCheck(timeout time.Duration) {
	ticker := time.NewTicker(timeout)
	for _ = range ticker.C {
		if p.beater.Ping(p.tracker, timeout) {
			close(p.done)
			return
		}
	}
}

func (p *Peer) tempLoop() {
	p.wg.Add(1)
	defer p.wg.Done()
	for {
		b := make([]byte, MaxPacketSize)
		size, sender, err := p.conn.ReadFromUDP(b)
		if size == 0 {
			continue
		}
		if err != nil {
			fmt.Printf("server: ReadFromUDP error: %v", err)
			continue
		}
		packet, err := ParsePacket(0, b)
		if err != nil {
			fmt.Println(err)
			continue
		}
		r := BroadResponse{
			Sender: sender,
			P:      packet,
		}
		p.beater.Put(r)
	}
}
