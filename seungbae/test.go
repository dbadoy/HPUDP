package main

import (
	"encoding/json"
	"fmt"
	"hpudp"
	"net"
)

func main() {
	client, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 8753})
	if err != nil {
		fmt.Println(err)
		return
	}

	// packet := new(hpudp.PingPacket)
	// packet.SetKind(hpudp.Ping)

	// b, err := json.Marshal(&packet)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// pb := make([]byte, len(b)+2)
	// if len(b) > 256 {
	// 	return
	// }
	// pb[0] = byte(len(b))
	// pb[1] = hpudp.Ping
	// copy(pb[2:], b[:])

	// fmt.Println(b, pb)

	// if _, err = client.WriteToUDP(pb, &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 8751}); err != nil {
	// 	fmt.Println(err)
	// }

	// packet2 := new(hpudp.JoinPacket)
	// packet2.SetKind(hpudp.Join)
	// packet2.NickName = "seungbae"

	// b2, err := json.Marshal(&packet2)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }

	// pb2 := make([]byte, len(b2)+2)
	// if len(b2) > 256 {
	// 	fmt.Println("oops")
	// 	return
	// }
	// pb2[0] = byte(len(b2))
	// pb2[1] = hpudp.Join
	// copy(pb2[2:], b2[:])
	// if _, err = client.WriteToUDP(pb2, &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 8751}); err != nil {
	// 	fmt.Println(err)
	// }

	packet3 := new(hpudp.FindPacket)
	packet3.SetKind(hpudp.Find)
	packet3.FindID = hpudp.ID([16]byte{82, 253, 252, 7, 33, 130, 101, 79, 22, 63, 95, 15, 154, 98, 29, 114})

	b3, err := json.Marshal(&packet3)
	if err != nil {
		fmt.Println(err)
		return
	}

	pb3 := make([]byte, len(b3)+2)
	if len(b3) > 256 {
		fmt.Println("oops")
		return
	}
	pb3[0] = byte(len(b3))
	pb3[1] = hpudp.Find
	copy(pb3[2:], b3[:])
	if _, err = client.WriteToUDP(pb3, &net.UDPAddr{IP: net.IP{127, 0, 0, 1}, Port: 8751}); err != nil {
		fmt.Println(err)
	}
}
