package routing

import (
	"regexp"
	"strings"
	"sync"

	"github.com/emiago/sipgo/sip"
	"github.com/sipplane/sipplane/internal/discovery"
	"github.com/sipplane/sipplane/internal/resources"
)

// Decision is the result of route matching.
type Decision struct {
	Route    *resources.Route
	Action   resources.RouteAction
	Trunk    *resources.Trunk
	DestAddr string // host:port for SetDestination
}

// Engine evaluates Routes against a SIP request using a resource snapshot.
type Engine struct {
	mu     sync.Mutex
	snap   *resources.Snapshot
	groups map[string]*discovery.DispatchGroup // route name -> LB group
}

func NewEngine(snap *resources.Snapshot) *Engine {
	e := &Engine{}
	e.ReplaceSnapshot(snap)
	return e
}

func (e *Engine) Snapshot() *resources.Snapshot {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.snap
}

func (e *Engine) ReplaceSnapshot(snap *resources.Snapshot) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.snap = snap
	e.groups = buildLBGroups(snap)
}

// Match finds the highest-priority matching route.
func (e *Engine) Match(req *sip.Request) (*Decision, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.snap == nil {
		return nil, false
	}
	method := req.Method.String()
	ruri := req.Recipient.String()
	from := ""
	if f := req.From(); f != nil {
		from = f.Address.String()
	}
	callID := ""
	if c := req.CallID(); c != nil {
		callID = c.Value()
	}

	for _, route := range e.snap.Routes {
		if !matchRoute(route, method, ruri, from) {
			continue
		}
		d := &Decision{Route: route, Action: route.Spec.Action}
		switch route.Spec.Action.Type {
		case "proxy":
			d.DestAddr = resolveProxyTarget(route.Spec.Action.Target)
		case "loadBalance":
			trunk := e.pickTrunkLocked(route, callID)
			if trunk != nil {
				d.Trunk = trunk
				d.DestAddr = trunk.Spec.Destination.HostPort()
			}
		case "registerLookup":
			// Destination filled by caller via location store.
		case "reject":
			// no dest
		}
		return d, true
	}
	return nil, false
}

func (e *Engine) pickTrunkLocked(route *resources.Route, callID string) *resources.Trunk {
	if g := e.groups[route.Metadata.Name]; g != nil {
		return g.Pick(callID)
	}
	// Fallback: highest weight (legacy)
	trunks := route.Spec.Action.Trunks
	if len(trunks) == 0 || e.snap == nil {
		return nil
	}
	best := trunks[0]
	for _, tw := range trunks[1:] {
		w := tw.Weight
		if w == 0 {
			w = 1
		}
		bw := best.Weight
		if bw == 0 {
			bw = 1
		}
		if w > bw {
			best = tw
		}
	}
	return e.snap.GetTrunk(route.Metadata.Tenant, best.Name)
}

func buildLBGroups(snap *resources.Snapshot) map[string]*discovery.DispatchGroup {
	out := make(map[string]*discovery.DispatchGroup)
	if snap == nil {
		return out
	}
	for _, route := range snap.Routes {
		if route == nil || route.Spec.Action.Type != "loadBalance" {
			continue
		}
		members := route.Spec.Action.Trunks
		if len(members) == 0 {
			continue
		}
		trunks := make(map[string]*resources.Trunk)
		for _, m := range members {
			if tr := snap.GetTrunk(route.Metadata.Tenant, m.Name); tr != nil {
				trunks[m.Name] = tr
			}
		}
		algo := route.Spec.Action.Algorithm
		if algo == "" && route.Spec.Action.Extra != nil {
			algo = route.Spec.Action.Extra["algorithm"]
		}
		if algo == "" {
			algo = "weighted"
		}
		out[route.Metadata.Name] = discovery.NewDispatchGroup(route.Metadata.Name, algo, members, trunks)
	}
	return out
}

func matchRoute(route *resources.Route, method, ruri, from string) bool {
	m := route.Spec.Match
	if len(m.Methods) > 0 {
		ok := false
		for _, x := range m.Methods {
			if strings.EqualFold(x, method) {
				ok = true
				break
			}
		}
		if !ok {
			return false
		}
	}
	if m.RequestURI != nil && !matchURI(m.RequestURI, ruri) {
		return false
	}
	if m.FromURI != nil && !matchURI(m.FromURI, from) {
		return false
	}
	return true
}

func matchURI(m *resources.URIMatch, value string) bool {
	if m.Exact != "" {
		return strings.EqualFold(m.Exact, value)
	}
	if m.Prefix != "" {
		return strings.HasPrefix(strings.ToLower(value), strings.ToLower(m.Prefix))
	}
	if m.Regex != "" {
		re, err := regexp.Compile(m.Regex)
		if err != nil {
			return false
		}
		return re.MatchString(value)
	}
	return true
}

func resolveProxyTarget(target string) string {
	target = strings.TrimSpace(target)
	if target == "" {
		return ""
	}
	if strings.Contains(target, "://") || strings.HasPrefix(target, "sip:") {
		u := sip.Uri{}
		if err := sip.ParseUri(target, &u); err == nil {
			host := u.Host
			port := u.Port
			if port == 0 {
				port = 5060
			}
			return host + ":" + itoa(port)
		}
	}
	if !strings.Contains(target, ":") {
		return target + ":5060"
	}
	return target
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
