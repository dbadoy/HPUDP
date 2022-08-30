package udphp

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
)

type Packet interface {
	SetSequnce(seq uint32)
	Sequnce() uint32
	SetKind(k byte)
	Kind() byte
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
		Founded      net.Addr
	}
)

func ParsePacket(seq uint32, p byte, d []byte) (Packet, error) {
	var r Packet
	switch p {
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
	if err := json.Unmarshal(d, r); err != nil {
		return nil, errors.New("invalid packet data")
	}
	r.SetSequnce(seq)
	r.SetKind(p)
	return r, nil
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
