package discovery

import (
	"testing"

	"github.com/sipplane/sipplane/internal/resources"
)

func TestDispatchConsistentHashStable(t *testing.T) {
	trunks := map[string]*resources.Trunk{
		"a": {Metadata: resources.ObjectMeta{Name: "a"}, Spec: resources.TrunkSpec{Destination: resources.TrunkDestination{Host: "1.1.1.1", Port: 5060}}},
		"b": {Metadata: resources.ObjectMeta{Name: "b"}, Spec: resources.TrunkSpec{Destination: resources.TrunkDestination{Host: "2.2.2.2", Port: 5060}}},
	}
	dg := NewDispatchGroup("g", "consistent_hash", []resources.TrunkWeight{
		{Name: "a", Weight: 100},
		{Name: "b", Weight: 100},
	}, trunks)
	first := dg.Pick("call-abc")
	for i := 0; i < 20; i++ {
		got := dg.Pick("call-abc")
		if got.Metadata.Name != first.Metadata.Name {
			t.Fatalf("hash not stable: %s vs %s", got.Metadata.Name, first.Metadata.Name)
		}
	}
}

func TestDispatchEjectUnhealthy(t *testing.T) {
	trunks := map[string]*resources.Trunk{
		"a": {Metadata: resources.ObjectMeta{Name: "a"}},
		"b": {Metadata: resources.ObjectMeta{Name: "b"}},
	}
	dg := NewDispatchGroup("g", "round_robin", []resources.TrunkWeight{
		{Name: "a"}, {Name: "b"},
	}, trunks)
	for i := 0; i < 5; i++ {
		dg.Mark("a", false, 5)
	}
	if dg.IsHealthy("a") {
		t.Fatal("a should be ejected")
	}
	for i := 0; i < 10; i++ {
		got := dg.Pick("x")
		if got == nil || got.Metadata.Name != "b" {
			t.Fatalf("expected only b, got %+v", got)
		}
	}
}
