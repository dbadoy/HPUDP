package main

import (
	"encoding/json"
	"fmt"
	"hpudp"
	"net"
)

const peers = 50

func main() {
	for i := 0; i < peers; i++ {
		go func(i int) {
			client, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 8753 + i})
			if err != nil {
				fmt.Println(err)
				return
			}
			packet2 := new(hpudp.JoinPacket)
			packet2.SetKind(hpudp.Join)
			packet2.NickName = "seungbae"

			b2, err := json.Marshal(&packet2)
			if err != nil {
				fmt.Println(err)
				return
			}

			pb2 := make([]byte, len(b2)+2)
			if len(b2) > 256 {
				fmt.Println("oops")
				return
			}
			pb2[0] = byte(len(b2))
			pb2[1] = hpudp.Join
			copy(pb2[2:], b2[:])
			if _, err = client.WriteToUDP(pb2, &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 8751}); err != nil {
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
				fmt.Printf("%v : %v\n", client.LocalAddr(), err)
			}
		}(i)
	}
	for {
	}
}
