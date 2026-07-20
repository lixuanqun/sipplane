package transform

import "testing"

func TestApplyUserStripPlus86(t *testing.T) {
	rules := []Rule{{
		Name:    "cn-mobile",
		Match:   `^\+86(\d+)$`,
		Replace: "0$1",
	}}
	got := ApplyUser("sip:+8613800138000@acme.example", rules)
	want := "sip:013800138000@acme.example"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestApplyUserNoMatch(t *testing.T) {
	rules := []Rule{{Match: `^\+1`, Replace: "1"}}
	in := "sip:1001@acme.example"
	if ApplyUser(in, rules) != in {
		t.Fatal("should leave unmatched")
	}
}
