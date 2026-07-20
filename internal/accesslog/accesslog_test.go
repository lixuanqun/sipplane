package accesslog

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestLogFields(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	l := New(log)
	l.Log(Event{
		CallID:     "c1",
		Method:     "INVITE",
		Code:       200,
		DurationMS: 12,
		Tenant:     "acme",
		Route:      "ua-to-ua",
		Trunk:      "",
		Src:        "1.2.3.4:5060",
		Dst:        "10.0.0.1:5060",
		Revision:   7,
	})
	var rec map[string]any
	if err := json.Unmarshal(buf.Bytes(), &rec); err != nil {
		t.Fatal(err)
	}
	if rec["msg"] != "access" {
		t.Fatalf("msg=%v", rec["msg"])
	}
	if rec["call_id"] != "c1" || rec["method"] != "INVITE" {
		t.Fatalf("%v", rec)
	}
	if int(rec["code"].(float64)) != 200 {
		t.Fatalf("code=%v", rec["code"])
	}
	if int(rec["revision"].(float64)) != 7 {
		t.Fatalf("revision=%v", rec["revision"])
	}
}

func TestStartElapsed(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))
	done := New(log).Start("cid", "OPTIONS", "127.0.0.1:9")
	time.Sleep(5 * time.Millisecond)
	done(200, "", "", "", "", 1)
	s := buf.String()
	if !strings.Contains(s, `"method":"OPTIONS"`) {
		t.Fatalf("%s", s)
	}
	if !strings.Contains(s, `"duration_ms"`) {
		t.Fatalf("missing duration: %s", s)
	}
}
