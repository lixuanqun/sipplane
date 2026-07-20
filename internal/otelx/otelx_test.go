package otelx

import (
	"context"
	"testing"
)

func TestSetupDisabled(t *testing.T) {
	shutdown, err := Setup(context.Background(), "sipplane", "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}

func TestSIPAttrs(t *testing.T) {
	attrs := SIPAttrs("INVITE", "cid-1", "acme", "lookup")
	if len(attrs) != 4 {
		t.Fatalf("attrs=%d", len(attrs))
	}
}

func TestTracerNoopWhenDisabled(t *testing.T) {
	tr := Tracer("sipplane/test")
	_, span := tr.Start(context.Background(), "invite")
	span.End()
}
