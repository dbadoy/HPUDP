package hpudp

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

const (
	// No basis. need to be considered
	MaxPacketSize = 256

	// No basis. need to be considered
	QueueLimit = 4096
)

type Server struct {
	// The unique number for request. It must increase by atomic.AddUint32.
	seq uint32

	conn *net.UDPConn

	// 'req' mapping the request sequnce number to reqeust UDP address.
	// It's use send response.
	req map[uint32]*net.UDPAddr

	// 'users' mapping random ID to UDP address.
	// this is core variable in hole-punching
	// if clients request the user's public address, find map 'users'.
	users map[ID]*net.UDPAddr

	// 'exists' mapping UDPAddress.String to ID
	// If clients request 'Join' request, check this variable and if already
	// exist, return the user ID without store.
	exists map[string]ID

	// 'queue' is the channel recv the packets. This channel limit size is 256.
	// This is for that split the dispatch packet, processing packet.
	queue chan Packet

	//
	beater Beater

	mu sync.Mutex
	wg sync.WaitGroup
}

func NewServer(addr *net.UDPAddr) *Server {
	conn, err := net.DialUDP("temp", addr, addr)
	if err != nil {
		return nil
	}
	return &Server{
		conn:   conn,
		users:  make(map[ID]*net.UDPAddr),
		exists: make(map[string]ID),
		req:    make(map[uint32]*net.UDPAddr),
		queue:  make(chan Packet, QueueLimit),
		beater: NewBeater(conn),
	}
}

func (s *Server) detectLoop() {
	defer s.wg.Done()

	for {
		var b []byte
		size, sender, err := s.conn.ReadFromUDP(b)
		if err != nil {
			fmt.Printf("server: ReadFromUDP error: %v", err)
			continue
		}
		if size > MaxPacketSize {
			fmt.Printf("server: exeed MaxPacketSize, got: %v, limit: %v", size, MaxPacketSize)
			continue
		}
		s.req[s.NextSequnce()] = sender
		// It's work?
		go func() {
			packet, err := ParsePacket(s.Sequnce(), b[0], b[1:])
			if err != nil {
				fmt.Printf("detected invalid protocol, reason : %v\n", err)
				return
			}
			s.queue <- packet
		}()
	}
}

func (s *Server) distributorLoop() {
	defer s.wg.Done()

	for {
		for packet := range s.queue {
			switch packet.Kind() {
			case Ping:
				go s.pong(s.targetFromSequnce(packet.Sequnce()), packet)
			case Pong:
				r := BroadRequest{
					Sender: s.targetFromSequnce(packet.Sequnce()),
					P:      packet,
				}
				s.beater.Put(r)
			case Join:
				go s.join(s.targetFromSequnce(packet.Sequnce()), packet)
			case Find:
				go s.find()
			default:
				// should not enter this case
			}
		}
	}
}

func (s *Server) targetFromSequnce(seq uint32) (addr *net.UDPAddr) {
	defer delete(s.req, seq)
	addr = s.req[seq]
	return
}

func (s *Server) NextSequnce() uint32 {
	return atomic.AddUint32(&s.seq, 1)
}

func (s *Server) Sequnce() uint32 {
	return atomic.LoadUint32(&s.seq)
}

// TODO:
func Send(conn *net.UDPConn, target *net.UDPAddr, packet Packet) error {
	b, err := json.Marshal(packet)
	if err != nil {
		return err
	}
	if _, err := conn.WriteToUDP(b, target); err != nil {
		return err
	}
	return nil
}

func (s *Sever) pong(target *net.UDPAddr, packet Packet) {
	p := new(PongPacket)
	p.SetSequnce(packet.Sequnce())
	p.SetKind(Pong)
	if err := Send(s.conn, target, p); err != nil {
		fmt.Println("server: send failed " + err.Error())
	}
}

func (s *Server) join(target *net.UDPAddr, packet Packet) {
	jp, ok := packet.(*JoinPacket)
	if !ok {
		fmt.Println("server: invalid packet in s.join")
		return
	}
	// If already exist, just filled user id in Server.
	if id := s.exists[target.String()]; id != [16]byte{} {
		// Default value, only for readabillity
		jp.Response = false
		jp.UserID = id
		if err := Send(s.conn, target, jp); err != nil {
			fmt.Println(err)
		}
		return
	}

	id := GenerateID()
	s.mu.Lock()
	s.exists[target.String()] = id
	s.mu.Unlock()

	s.users[id] = target
	s.beater.Register(target)

	jp.Response = true
	jp.UserID = id
	if err := Send(s.conn, target, jp); err != nil {
		fmt.Println(err)
		return
	}
}

func (s *Server) find() error {
	return errors.New("not implement")
}
