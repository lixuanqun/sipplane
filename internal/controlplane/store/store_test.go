package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/sipplane/sipplane/internal/controlplane/store"
	"github.com/sipplane/sipplane/internal/resources"
)

func TestMemoryApplyDryRunWatch(t *testing.T) {
	m := store.NewMemory(nil)
	yaml := []byte(`
apiVersion: sipplane.io/v1alpha1
kind: Route
metadata:
  name: r1
  tenant: acme
spec:
  priority: 10
  match:
    methods: ["INVITE"]
  action:
    type: registerLookup
`)
	ctx := context.Background()
	rev, err := m.Apply(ctx, "test", yaml, true)
	if err != nil {
		t.Fatal(err)
	}
	if rev != 0 {
		t.Fatalf("dry-run should not bump revision, got %d", rev)
	}
	cur, _ := m.Revision(ctx)
	if cur != 0 {
		t.Fatalf("revision=%d", cur)
	}

	rev, err = m.Apply(ctx, "test", yaml, false)
	if err != nil {
		t.Fatal(err)
	}
	if rev != 1 {
		t.Fatalf("revision=%d", rev)
	}
	snap, err := m.Snapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Routes) != 1 || snap.Revision != 1 {
		t.Fatalf("snap=%+v", snap)
	}

	bad := []byte(`
kind: Route
metadata:
  name: bad
spec:
  action:
    type: not-a-real-action
`)
	_, err = m.Apply(ctx, "test", bad, false)
	if err == nil {
		t.Fatal("expected validation error")
	}

	done := make(chan int64, 1)
	go func() {
		r, err := m.Watch(context.Background(), 1)
		if err != nil {
			t.Errorf("watch: %v", err)
			return
		}
		done <- r
	}()
	time.Sleep(50 * time.Millisecond)
	_, _ = m.Apply(ctx, "test", yaml, false)
	select {
	case r := <-done:
		if r < 2 {
			t.Fatalf("watch rev=%d", r)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("watch timeout")
	}
}

func TestValidateSnapshot(t *testing.T) {
	err := store.ValidateSnapshot(&resources.Snapshot{
		Routes: []*resources.Route{{
			Metadata: resources.ObjectMeta{Name: "x"},
			Spec:     resources.RouteSpec{Action: resources.RouteAction{Type: "nope"}},
		}},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestMemoryWatchTimeout(t *testing.T) {
	m := store.NewMemory(nil)
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Millisecond)
	defer cancel()
	_, err := m.Watch(ctx, 100)
	if err == nil {
		t.Fatal("expected watch timeout/cancel error")
	}
}
