package main

import (
	"encoding/json"
	"fmt"
	"hpudp"
	"net"
)

func main() {
	client, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 8754})
	if err != nil {
		fmt.Println(err)
		return
	}

	for {
		b := make([]byte, 32)
		size, sender, err := client.ReadFromUDP(b)
		if size == 0 {
			continue
		}
		if err != nil {
			fmt.Printf("server: ReadFromUDP error: %v", err)
			continue
		}
		r := new(hpudp.PongPacket)
		bb, _ := json.Marshal(&r)
		bb2 := make([]byte, len(bb)+2)
		bb2[0] = byte(len(bb))
		bb2[1] = hpudp.Pong
		copy(bb2[2:], bb[:])
		_, err = client.WriteToUDP(bb2, sender)
		fmt.Println(err)
	}
}
