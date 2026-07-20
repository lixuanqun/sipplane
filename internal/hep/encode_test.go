package hep

import (
	"encoding/binary"
	"net"
	"testing"
	"time"
)

func TestEncodeHEP3HeaderAndPayload(t *testing.T) {
	payload := []byte("INVITE sip:bob@acme.example SIP/2.0\r\n")
	raw := Encode(Packet{
		Data:      payload,
		SrcIP:     net.ParseIP("1.2.3.4"),
		DstIP:     net.ParseIP("5.6.7.8"),
		SrcPort:   5060,
		DstPort:   5060,
		IPProto:   17,
		CaptureID: 42,
		Timestamp: time.Unix(1700000000, 123456000),
	})
	if string(raw[0:4]) != "HEP3" {
		t.Fatalf("magic=%q", raw[0:4])
	}
	total := binary.BigEndian.Uint16(raw[4:6])
	if int(total) != len(raw) {
		t.Fatalf("length field=%d actual=%d", total, len(raw))
	}
	// Payload chunk should contain INVITE bytes at the end.
	if !containsBytes(raw, payload) {
		t.Fatal("payload missing from HEP packet")
	}
}

func TestExporterDisabledNoop(t *testing.T) {
	e := NewExporter("", 1, nil)
	e.Send([]byte("x"), "1.1.1.1:5060", "2.2.2.2:5060", true)
}

func TestExporterSendUDP(t *testing.T) {
	pc, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer pc.Close()

	got := make(chan []byte, 1)
	go func() {
		buf := make([]byte, 65535)
		_ = pc.SetReadDeadline(time.Now().Add(3 * time.Second))
		n, _, err := pc.ReadFrom(buf)
		if err != nil {
			return
		}
		got <- append([]byte(nil), buf[:n]...)
	}()

	e := NewExporter(pc.LocalAddr().String(), 99, nil)
	defer e.Close()
	payload := []byte("INVITE sip:bob@acme.example SIP/2.0\r\n")
	e.Send(payload, "1.2.3.4:5060", "5.6.7.8:5060", true)

	select {
	case raw := <-got:
		if string(raw[0:4]) != "HEP3" {
			t.Fatalf("magic=%q", raw[0:4])
		}
		if !containsBytes(raw, payload) {
			t.Fatal("payload missing")
		}
	case <-time.After(3 * time.Second):
		t.Fatal("no HEP packet received")
	}
}

func containsBytes(haystack, needle []byte) bool {
	if len(needle) == 0 {
		return true
	}
	for i := 0; i+len(needle) <= len(haystack); i++ {
		ok := true
		for j := 0; j < len(needle); j++ {
			if haystack[i+j] != needle[j] {
				ok = false
				break
			}
		}
		if ok {
			return true
		}
	}
	return false
}
