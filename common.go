package hpudp

import "math/rand"

const IdLength = 16

type ID [IdLength]byte

func GenerateID() ID {
	b := make([]byte, IdLength)
	rand.Read(b)
	var t [IdLength]byte
	copy(t[:], b)
	return ID(t)
}
