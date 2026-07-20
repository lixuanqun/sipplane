package hep

import (
	"encoding/binary"
	"net"
	"time"
)

// Vendor / chunk type IDs (generic vendor 0x0000).
const (
	vendorGeneric uint16 = 0x0000

	chunkIPProto     uint16 = 0x0001 // uint8 IP protocol
	chunkIP4Src      uint16 = 0x0003
	chunkIP4Dst      uint16 = 0x0004
	chunkSrcPort     uint16 = 0x0007
	chunkDstPort     uint16 = 0x0008
	chunkTimestamp   uint16 = 0x0009 // uint32 seconds
	chunkTimestampUS uint16 = 0x000a // uint32 microseconds
	chunkProtoType   uint16 = 0x000b // uint8: 1=SIP
	chunkCaptureID   uint16 = 0x000c // uint32
	chunkPayload     uint16 = 0x000f

	protoSIP uint8 = 1
)

// Packet is a SIP capture to encapsulate.
type Packet struct {
	Data       []byte
	SrcIP      net.IP
	DstIP      net.IP
	SrcPort    uint16
	DstPort    uint16
	IPProto    uint8 // 17=UDP, 6=TCP
	CaptureID  uint32
	Timestamp  time.Time
}

// Encode builds a HEP3 datagram.
func Encode(p Packet) []byte {
	if p.Timestamp.IsZero() {
		p.Timestamp = time.Now()
	}
	if p.IPProto == 0 {
		p.IPProto = 17
	}
	src4 := p.SrcIP.To4()
	dst4 := p.DstIP.To4()
	if src4 == nil {
		src4 = net.IPv4zero.To4()
	}
	if dst4 == nil {
		dst4 = net.IPv4zero.To4()
	}

	var chunks []byte
	chunks = append(chunks, chunkUint8(chunkIPProto, p.IPProto)...)
	chunks = append(chunks, chunkBytes(chunkIP4Src, src4)...)
	chunks = append(chunks, chunkBytes(chunkIP4Dst, dst4)...)
	chunks = append(chunks, chunkUint16(chunkSrcPort, p.SrcPort)...)
	chunks = append(chunks, chunkUint16(chunkDstPort, p.DstPort)...)
	sec := uint32(p.Timestamp.Unix())
	usec := uint32(p.Timestamp.Nanosecond() / 1000)
	chunks = append(chunks, chunkUint32(chunkTimestamp, sec)...)
	chunks = append(chunks, chunkUint32(chunkTimestampUS, usec)...)
	chunks = append(chunks, chunkUint8(chunkProtoType, protoSIP)...)
	chunks = append(chunks, chunkUint32(chunkCaptureID, p.CaptureID)...)
	chunks = append(chunks, chunkBytes(chunkPayload, p.Data)...)

	total := 6 + len(chunks)
	out := make([]byte, total)
	copy(out[0:4], []byte("HEP3"))
	binary.BigEndian.PutUint16(out[4:6], uint16(total))
	copy(out[6:], chunks)
	return out
}

func chunkHeader(typ uint16, payloadLen int) []byte {
	// vendor(2) + type(2) + length(2) + payload
	total := 6 + payloadLen
	h := make([]byte, 6)
	binary.BigEndian.PutUint16(h[0:2], vendorGeneric)
	binary.BigEndian.PutUint16(h[2:4], typ)
	binary.BigEndian.PutUint16(h[4:6], uint16(total))
	return h
}

func chunkUint8(typ uint16, v uint8) []byte {
	return append(chunkHeader(typ, 1), v)
}

func chunkUint16(typ uint16, v uint16) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, v)
	return append(chunkHeader(typ, 2), b...)
}

func chunkUint32(typ uint16, v uint32) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, v)
	return append(chunkHeader(typ, 4), b...)
}

func chunkBytes(typ uint16, payload []byte) []byte {
	return append(chunkHeader(typ, len(payload)), payload...)
}
