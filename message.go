package hpudp

import (
	"encoding/json"
	"errors"
	"net"
)

const (
	Ping = iota
	Pong
	Join
	Find

	PrefixSize      = 2
	packetSizeIndex = 0
	packetTypeIndex = 1
)

type Packet interface {
	SetSequnce(seq uint32)
	Sequnce() uint32
	SetKind(k byte)
	Kind() byte
}

type BroadResponse struct {
	Sender *net.UDPAddr
	P      Packet
}

type (
	PingPacket struct {
		seq  uint32
		kind byte
	}
	PongPacket struct {
		seq  uint32
		kind byte
	}
	JoinPacket struct {
		seq      uint32
		kind     byte
		UserID   ID
		NickName string
		Response bool
	}
	FindPacket struct {
		seq          uint32
		kind         byte
		FindID       ID
		FindNickName string
		Founded      string
	}
)

func ParsePacket(seq uint32, d []byte) (Packet, error) {
	var r Packet
	switch d[packetTypeIndex] {
	case Ping:
		r = new(PingPacket)
	case Pong:
		r = new(PongPacket)
	case Join:
		r = new(JoinPacket)
	case Find:
		r = new(FindPacket)
	default:
		return nil, errors.New("invalid packet type")
	}

	if err := SuitableUnpack(d, r); err != nil {
		return nil, err
	}
	r.SetSequnce(seq)
	r.SetKind(d[packetTypeIndex])
	return r, nil
}

// SuitablePack is change Packet to suitable protocol message.
// If send message, you must use this method.
// [TODO: add InternalIP]
func SuitablePack(packet Packet) ([]byte, error) {
	b, err := json.Marshal(packet)
	if err != nil {
		return nil, err
	}
	result := make([]byte, len(b)+PrefixSize)
	result[packetSizeIndex] = byte(len(b))
	result[packetTypeIndex] = packet.Kind()
	copy(result[2:], b[:])
	return result, nil
}

// The recv message format like this: PacketSize + PacketType + Payload
// [TODO: InternalIP + PacketSize + PacketType + Payload]
// This is the logic for parse the Payload
func SuitableUnpack(b []byte, packet Packet) error {
	len := b[packetSizeIndex]
	byt := make([]byte, len)
	copy(byt[:], b[PrefixSize:PrefixSize+len])
	if err := json.Unmarshal(byt, packet); err != nil {
		return errors.New("invalid packet data")
	}
	return nil
}

func (p *PingPacket) Sequnce() uint32       { return p.seq }
func (p *PingPacket) SetSequnce(seq uint32) { p.seq = seq }
func (p *PingPacket) SetKind(k byte)        { p.kind = k }
func (p *PingPacket) Kind() byte            { return p.kind }

func (p *PongPacket) Sequnce() uint32       { return p.seq }
func (p *PongPacket) SetSequnce(seq uint32) { p.seq = seq }
func (p *PongPacket) SetKind(k byte)        { p.kind = k }
func (p *PongPacket) Kind() byte            { return p.kind }

func (j *JoinPacket) Sequnce() uint32       { return j.seq }
func (j *JoinPacket) SetSequnce(seq uint32) { j.seq = seq }
func (j *JoinPacket) SetKind(k byte)        { j.kind = k }
func (j *JoinPacket) Kind() byte            { return j.kind }

func (f *FindPacket) Sequnce() uint32       { return f.seq }
func (f *FindPacket) SetSequnce(seq uint32) { f.seq = seq }
func (f *FindPacket) SetKind(k byte)        { f.kind = k }
func (f *FindPacket) Kind() byte            { return f.kind }
