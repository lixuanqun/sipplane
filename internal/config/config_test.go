package config

import "testing"

func TestValidateRequiresAdvertisedHost(t *testing.T) {
	cfg := Config{Listen: "0.0.0.0:5060"}
	if err := cfg.Validate(); err == nil {
		t.Fatal("expected error for missing advertised_host")
	}
	cfg.AdvertisedHost = "sip.example.com"
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateAllowsLoopbackWithoutAdvertised(t *testing.T) {
	cfg := Config{Listen: "127.0.0.1:5060"}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestAdvertisedSIPURI(t *testing.T) {
	cfg := Config{AdvertisedHost: "sip.example.com", AdvertisedPort: 5060}
	got := cfg.AdvertisedSIPURI()
	want := "sip:sip.example.com:5060;lr"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}
