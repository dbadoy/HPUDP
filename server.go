package hpudp

import (
	"encoding/json"
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

	//
	packetSizeIndex = 0
	packetTypeIndex = 1
	prefixSize      = 2
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

	beater Beater

	mu sync.Mutex
	wg sync.WaitGroup
}

func NewServer(addr *net.UDPAddr) *Server {
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println(err)
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

func (s *Server) Start() {
	fmt.Println(s.conn.LocalAddr())
	s.wg.Add(2)

	go s.detectLoop()
	go s.distributeLoop()

	s.wg.Wait()
}

func (s *Server) detectLoop() {
	defer s.wg.Done()

	for {
		b := make([]byte, MaxPacketSize)
		size, sender, err := s.conn.ReadFromUDP(b)
		if size == 0 {
			continue
		}
		if err != nil {
			fmt.Printf("server: ReadFromUDP error: %v", err)
			continue
		}
		s.req[s.NextSequnce()] = sender
		// It's work?
		go func() {
			packetSize := b[packetSizeIndex]
			packet, err := ParsePacket(s.Sequnce(), b[packetTypeIndex], b[prefixSize:prefixSize+packetSize])
			if err != nil {
				fmt.Printf("detected invalid protocol, reason : %v\n", err)
				return
			}
			s.queue <- packet
		}()
	}
}

func (s *Server) distributeLoop() {
	defer s.wg.Done()

	for packet := range s.queue {
		switch packet.Kind() {
		case Ping:
			go s.pong(packet)
		case Pong:
			r := BroadResponse{
				Sender: s.targetFromSequnce(packet.Sequnce()),
				P:      packet,
			}
			s.beater.Put(r)
		case Join:
			go s.join(packet)
		case Find:
			go s.find(packet)
		default:
			// should not enter this case
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

func (s *Server) pong(packet Packet) {
	target := s.targetFromSequnce(packet.Sequnce())
	p := new(PongPacket)
	p.SetSequnce(packet.Sequnce())
	p.SetKind(Pong)
	if err := Send(s.conn, target, p); err != nil {
		fmt.Println("server: send failed " + err.Error())
	}
}

func (s *Server) join(packet Packet) {
	target := s.targetFromSequnce(packet.Sequnce())
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

func (s *Server) find(packet Packet) {
	target := s.targetFromSequnce(packet.Sequnce())
	fp, ok := packet.(*FindPacket)
	if !ok {
		fmt.Println("server: invalid packet in s.find")
		return
	}
	fid := fp.FindID
	found := s.users[fid]
	if found != nil {
		// is pointer ok?
		fp.Founded = found
	}
	if err := Send(s.conn, target, fp); err != nil {
		fmt.Println(err)
		return
	}
}
