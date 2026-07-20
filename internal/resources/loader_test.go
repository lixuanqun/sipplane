package resources

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDirLab(t *testing.T) {
	dir := filepath.Join("..", "..", "examples", "config")
	if _, err := os.Stat(filepath.Join(dir, "lab.yaml")); err != nil {
		t.Skip("examples/config/lab.yaml not found")
	}
	snap, err := LoadDir(filepath.Join(dir, "lab.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Endpoints) < 2 {
		t.Fatalf("endpoints=%d", len(snap.Endpoints))
	}
	if len(snap.Routes) < 1 {
		t.Fatalf("routes=%d", len(snap.Routes))
	}
	ep := snap.FindEndpointByAOR("sip:alice@acme.example")
	if ep == nil {
		t.Fatal("alice not found")
	}
	pass, ok := snap.ResolvePassword(ep.Spec.Auth.PasswordSecretRef)
	if !ok || pass != "alice-secret" {
		t.Fatalf("password resolve failed: ok=%v pass=%q ref=%q", ok, pass, ep.Spec.Auth.PasswordSecretRef)
	}
	// Priority order: reject-blocked (200) before ua-to-ua (100)
	if snap.Routes[0].Metadata.Name != "reject-blocked" {
		t.Fatalf("first route=%s", snap.Routes[0].Metadata.Name)
	}
}

func TestLoadDirSkipsBootstrapFiles(t *testing.T) {
	dir := filepath.Join("..", "..", "examples", "config")
	if _, err := os.Stat(dir); err != nil {
		t.Skip("examples/config missing")
	}
	snap, err := LoadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Endpoints) < 2 || len(snap.Routes) < 1 {
		t.Fatalf("snap endpoints=%d routes=%d", len(snap.Endpoints), len(snap.Routes))
	}
}

func TestIsBootstrapFile(t *testing.T) {
	cases := map[string]bool{
		"bootstrap.yaml":      true,
		"bootstrap.yml":       true,
		"bootstrap-edge.yaml": true,
		"Bootstrap-Lab.YAML":  true,
		"lab.yaml":            false,
		"my-bootstrap.yaml":   false,
	}
	for name, want := range cases {
		if got := isBootstrapFile(name); got != want {
			t.Fatalf("%s: got %v want %v", name, got, want)
		}
	}
}
