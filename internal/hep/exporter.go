package hep

import (
	"log/slog"
	"net"
	"strconv"
	"sync"
	"time"
)

// Exporter sends HEP3 packets to a Homer/captagent collector over UDP.
type Exporter struct {
	Addr      string
	CaptureID uint32
	Log       *slog.Logger

	mu   sync.Mutex
	conn net.Conn
}

// NewExporter creates a HEP exporter. Addr empty disables sending.
func NewExporter(addr string, captureID uint32, log *slog.Logger) *Exporter {
	if captureID == 0 {
		captureID = 2001
	}
	if log == nil {
		log = slog.Default()
	}
	return &Exporter{Addr: addr, CaptureID: captureID, Log: log}
}

// Send encodes and writes a SIP message payload.
func (e *Exporter) Send(payload []byte, src, dst string, udp bool) {
	if e == nil || e.Addr == "" || len(payload) == 0 {
		return
	}
	srcHost, srcPort := split(src)
	dstHost, dstPort := split(dst)
	ipProto := uint8(17)
	if !udp {
		ipProto = 6
	}
	pkt := Encode(Packet{
		Data:      payload,
		SrcIP:     net.ParseIP(srcHost),
		DstIP:     net.ParseIP(dstHost),
		SrcPort:   uint16(srcPort),
		DstPort:   uint16(dstPort),
		IPProto:   ipProto,
		CaptureID: e.CaptureID,
		Timestamp: time.Now(),
	})
	if err := e.write(pkt); err != nil {
		e.Log.Debug("hep send failed", "err", err)
	}
}

func (e *Exporter) write(b []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.conn == nil {
		c, err := net.DialTimeout("udp", e.Addr, 2*time.Second)
		if err != nil {
			return err
		}
		e.conn = c
	}
	_, err := e.conn.Write(b)
	if err != nil {
		_ = e.conn.Close()
		e.conn = nil
	}
	return err
}

// Close closes the UDP connection.
func (e *Exporter) Close() {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.conn != nil {
		_ = e.conn.Close()
		e.conn = nil
	}
}

func split(addr string) (string, int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 0
	}
	p, _ := strconv.Atoi(portStr)
	return host, p
}
