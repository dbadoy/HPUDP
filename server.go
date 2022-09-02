package hpudp

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
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

	ticker := time.NewTicker(5 * time.Second)
	cancel := s.beater.BroadcastPingWithTicker(*ticker, 3*time.Second)
	_ = cancel

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
		s.mu.Lock()
		s.req[s.NextSequnce()] = sender
		s.mu.Unlock()

		// Get sequence number when before start goroutine.
		// Not after started goroutine. It may not thread safety.
		go func(seq uint32) {
			packet, err := ParsePacket(seq, b)
			if err != nil {
				fmt.Printf("detected invalid protocol, reason : %v\n", err)
				return
			}
			s.queue <- packet
		}(s.Sequnce())
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
	s.mu.Lock()
	addr = s.req[seq]
	delete(s.req, seq)
	s.mu.Unlock()
	return
}

func (s *Server) NextSequnce() uint32 {
	return atomic.AddUint32(&s.seq, 1)
}

func (s *Server) Sequnce() uint32 {
	return atomic.LoadUint32(&s.seq)
}

func Send(conn *net.UDPConn, target *net.UDPAddr, packet Packet) error {
	result, err := SuitablePack(packet)
	if err != nil {
		return err
	}
	if _, err := conn.WriteToUDP(result, target); err != nil {
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
	s.users[id] = target
	s.mu.Unlock()

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
	s.mu.Lock()
	found := s.users[fid]
	s.mu.Unlock()
	if found != nil {
		// is pointer ok?
		fp.Founded = found.String()
	}
	if err := Send(s.conn, target, fp); err != nil {
		fmt.Println(err)
		return
	}
}
