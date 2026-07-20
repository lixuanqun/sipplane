package discovery

import (
	"sync"

	"github.com/sipplane/sipplane/internal/resources"
)

// MemberHealth tracks trunk health for DispatchGroup.
type MemberHealth struct {
	Trunk   string
	Healthy bool
	Fails   int
}

// DispatchGroup manages backend selection + health (P3).
type DispatchGroup struct {
	Name      string
	Algorithm string // round_robin | weighted | consistent_hash
	Members   []resources.TrunkWeight
	Trunks    map[string]*resources.Trunk

	mu      sync.Mutex
	health  map[string]*MemberHealth
	rrIndex int
}

func NewDispatchGroup(name, algo string, members []resources.TrunkWeight, trunks map[string]*resources.Trunk) *DispatchGroup {
	dg := &DispatchGroup{
		Name:      name,
		Algorithm: algo,
		Members:   members,
		Trunks:    trunks,
		health:    make(map[string]*MemberHealth),
	}
	for _, m := range members {
		dg.health[m.Name] = &MemberHealth{Trunk: m.Name, Healthy: true}
	}
	return dg
}

func (d *DispatchGroup) healthyMembers() []resources.TrunkWeight {
	var healthy []resources.TrunkWeight
	for _, m := range d.Members {
		h := d.health[m.Name]
		if h == nil || h.Healthy {
			healthy = append(healthy, m)
		}
	}
	return healthy
}

func (d *DispatchGroup) Pick(callID string) *resources.Trunk {
	d.mu.Lock()
	defer d.mu.Unlock()
	healthy := d.healthyMembers()
	if len(healthy) == 0 {
		return nil
	}
	var name string
	switch d.Algorithm {
	case "consistent_hash", "call-id":
		name = healthy[hashString(callID)%len(healthy)].Name
	case "weighted":
		best := healthy[0]
		for _, m := range healthy[1:] {
			if m.Weight > best.Weight {
				best = m
			}
		}
		name = best.Name
	default:
		name = healthy[d.rrIndex%len(healthy)].Name
		d.rrIndex++
	}
	if d.Trunks == nil {
		return nil
	}
	return d.Trunks[name]
}

func (d *DispatchGroup) Mark(trunk string, ok bool, ejectAfter int) {
	d.mu.Lock()
	defer d.mu.Unlock()
	h := d.health[trunk]
	if h == nil {
		h = &MemberHealth{Trunk: trunk, Healthy: true}
		d.health[trunk] = h
	}
	if ok {
		h.Fails = 0
		h.Healthy = true
		return
	}
	h.Fails++
	if ejectAfter <= 0 {
		ejectAfter = 5
	}
	if h.Fails >= ejectAfter {
		h.Healthy = false
	}
}

func (d *DispatchGroup) IsHealthy(trunk string) bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	h := d.health[trunk]
	return h == nil || h.Healthy
}

func hashString(s string) int {
	h := 0
	for i := 0; i < len(s); i++ {
		h = 31*h + int(s[i])
	}
	if h < 0 {
		h = -h
	}
	return h
}
