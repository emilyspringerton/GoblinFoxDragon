package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math"
	"net"
	"time"

	"dragonsnshit/packages/common"
)

func main() {
	host := flag.String("host", "127.0.0.1", "server host")
	port := flag.Int("port", 6969, "server port")
	interval := flag.Duration("interval", 50*time.Millisecond, "send interval")
	flag.Parse()

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		panic(err)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		panic(err)
	}
	defer conn.Close()

	fmt.Printf("client-go connected to %s\n", addr.String())

	seq := uint32(0)
	ticker := time.NewTicker(*interval)
	defer ticker.Stop()

	for range ticker.C {
		seq++
		cmd := common.UserCmd{
			Sequence:  seq,
			Timestamp: uint32(time.Now().UnixMilli()),
			Msec:      uint16(interval.Milliseconds()),
			Fwd:       0,
			Str:       0,
			Yaw:       0,
			Pitch:     0,
			Buttons:   0,
			WeaponIdx: int32(common.WpnMagnum),
		}
		sendUserCmd(conn, cmd)
	}
}

func sendUserCmd(conn *net.UDPConn, cmd common.UserCmd) {
	buf := make([]byte, 0, 64)
	buf = append(buf, common.PacketUserCmd)
	buf = appendUint32(buf, cmd.Sequence)
	buf = appendUint32(buf, cmd.Timestamp)
	buf = appendUint16(buf, cmd.Msec)
	buf = appendFloat32(buf, cmd.Fwd)
	buf = appendFloat32(buf, cmd.Str)
	buf = appendFloat32(buf, cmd.Yaw)
	buf = appendFloat32(buf, cmd.Pitch)
	buf = appendUint32(buf, cmd.Buttons)
	buf = appendUint32(buf, uint32(cmd.WeaponIdx))

	_, _ = conn.Write(buf)
}

func appendUint16(buf []byte, v uint16) []byte {
	var tmp [2]byte
	binary.LittleEndian.PutUint16(tmp[:], v)
	return append(buf, tmp[:]...)
}

func appendUint32(buf []byte, v uint32) []byte {
	var tmp [4]byte
	binary.LittleEndian.PutUint32(tmp[:], v)
	return append(buf, tmp[:]...)
}

func appendFloat32(buf []byte, v float32) []byte {
	return appendUint32(buf, math.Float32bits(v))
}
