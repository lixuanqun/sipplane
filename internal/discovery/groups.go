package discovery

import "github.com/sipplane/sipplane/internal/resources"

// GroupsFromPingTrunks builds DispatchGroups for trunks with SendOptionsPing.
func GroupsFromPingTrunks(trunks map[string]*resources.Trunk) []*DispatchGroup {
	if len(trunks) == 0 {
		return nil
	}
	var members []resources.TrunkWeight
	selected := make(map[string]*resources.Trunk)
	for name, tr := range trunks {
		if tr == nil || !tr.Spec.Options.SendOptionsPing {
			continue
		}
		members = append(members, resources.TrunkWeight{Name: name, Weight: 1})
		selected[name] = tr
	}
	if len(members) == 0 {
		return nil
	}
	return []*DispatchGroup{NewDispatchGroup("options-ping", "round_robin", members, selected)}
}
