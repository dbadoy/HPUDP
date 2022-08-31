package hpudp

import "net"

type Peer struct {
	conn   *net.Conn
	beater Beater
}

func NewPeer() *Peer {
	return nil
}
