package udphp

import "net"

type Peer struct {
	conn net.Conn
}

func NewPeer() *Peer {
	return nil
}
