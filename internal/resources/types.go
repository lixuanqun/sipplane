package resources

import "time"

const APIVersion = "sipplane.io/v1alpha1"

// ObjectMeta is shared metadata for all resources.
type ObjectMeta struct {
	Name   string            `yaml:"name" json:"name"`
	Tenant string            `yaml:"tenant,omitempty" json:"tenant,omitempty"`
	Labels map[string]string `yaml:"labels,omitempty" json:"labels,omitempty"`
}

// Tenant isolates credentials, routes, and quotas.
type Tenant struct {
	APIVersion string     `yaml:"apiVersion" json:"apiVersion"`
	Kind       string     `yaml:"kind" json:"kind"`
	Metadata   ObjectMeta `yaml:"metadata" json:"metadata"`
	Spec       TenantSpec `yaml:"spec" json:"spec"`
}

type TenantSpec struct {
	DisplayName string       `yaml:"displayName,omitempty" json:"displayName,omitempty"`
	Quotas      TenantQuotas `yaml:"quotas,omitempty" json:"quotas,omitempty"`
}

type TenantQuotas struct {
	MaxEndpoints int `yaml:"maxEndpoints,omitempty" json:"maxEndpoints,omitempty"`
	MaxCPS       int `yaml:"maxCPS,omitempty" json:"maxCPS,omitempty"`
}

// Endpoint is an authenticated UA / PBX.
type Endpoint struct {
	APIVersion string       `yaml:"apiVersion" json:"apiVersion"`
	Kind       string       `yaml:"kind" json:"kind"`
	Metadata   ObjectMeta   `yaml:"metadata" json:"metadata"`
	Spec       EndpointSpec `yaml:"spec" json:"spec"`
}

type EndpointSpec struct {
	AORs  []string     `yaml:"aors" json:"aors"`
	Auth  EndpointAuth `yaml:"auth" json:"auth"`
	Allow EndpointAllow `yaml:"allow,omitempty" json:"allow,omitempty"`
}

type EndpointAuth struct {
	Username          string `yaml:"username" json:"username"`
	PasswordSecretRef string `yaml:"passwordSecretRef,omitempty" json:"passwordSecretRef,omitempty"`
	// Password is lab-only; prefer passwordSecretRef in real deployments.
	Password string `yaml:"password,omitempty" json:"password,omitempty"`
}

type EndpointAllow struct {
	Register *bool `yaml:"register,omitempty" json:"register,omitempty"`
	Invite   *bool `yaml:"invite,omitempty" json:"invite,omitempty"`
}

func (a EndpointAllow) CanRegister() bool {
	if a.Register == nil {
		return true
	}
	return *a.Register
}

func (a EndpointAllow) CanInvite() bool {
	if a.Invite == nil {
		return true
	}
	return *a.Invite
}

// Trunk is a carrier / peer interconnection.
type Trunk struct {
	APIVersion string     `yaml:"apiVersion" json:"apiVersion"`
	Kind       string     `yaml:"kind" json:"kind"`
	Metadata   ObjectMeta `yaml:"metadata" json:"metadata"`
	Spec       TrunkSpec  `yaml:"spec" json:"spec"`
}

type TrunkSpec struct {
	Destination TrunkDestination `yaml:"destination" json:"destination"`
	Auth        TrunkAuth        `yaml:"auth,omitempty" json:"auth,omitempty"`
	Options     TrunkOptions     `yaml:"options,omitempty" json:"options,omitempty"`
}

type TrunkDestination struct {
	Host      string `yaml:"host" json:"host"`
	Port      int    `yaml:"port,omitempty" json:"port,omitempty"`
	Transport string `yaml:"transport,omitempty" json:"transport,omitempty"`
}

func (d TrunkDestination) HostPort() string {
	port := d.Port
	if port == 0 {
		port = 5060
	}
	return d.Host + ":" + itoa(port)
}

type TrunkAuth struct {
	Outbound *TrunkOutboundAuth `yaml:"outbound,omitempty" json:"outbound,omitempty"`
}

type TrunkOutboundAuth struct {
	Username          string `yaml:"username" json:"username"`
	PasswordSecretRef string `yaml:"passwordSecretRef,omitempty" json:"passwordSecretRef,omitempty"`
	Password          string `yaml:"password,omitempty" json:"password,omitempty"`
}

type TrunkOptions struct {
	SendOptionsPing bool          `yaml:"sendOptionsPing,omitempty" json:"sendOptionsPing,omitempty"`
	PingInterval    time.Duration `yaml:"pingInterval,omitempty" json:"pingInterval,omitempty"`
}

// Route matches requests to actions.
type Route struct {
	APIVersion string     `yaml:"apiVersion" json:"apiVersion"`
	Kind       string     `yaml:"kind" json:"kind"`
	Metadata   ObjectMeta `yaml:"metadata" json:"metadata"`
	Spec       RouteSpec  `yaml:"spec" json:"spec"`
}

type RouteSpec struct {
	Priority int         `yaml:"priority,omitempty" json:"priority,omitempty"`
	Match    RouteMatch  `yaml:"match" json:"match"`
	Action   RouteAction `yaml:"action" json:"action"`
}

type RouteMatch struct {
	Methods    []string     `yaml:"methods,omitempty" json:"methods,omitempty"`
	RequestURI *URIMatch    `yaml:"requestUri,omitempty" json:"requestUri,omitempty"`
	FromURI    *URIMatch    `yaml:"fromUri,omitempty" json:"fromUri,omitempty"`
}

type URIMatch struct {
	Prefix string `yaml:"prefix,omitempty" json:"prefix,omitempty"`
	Exact  string `yaml:"exact,omitempty" json:"exact,omitempty"`
	Regex  string `yaml:"regex,omitempty" json:"regex,omitempty"`
}

type RouteAction struct {
	Type      string            `yaml:"type" json:"type"`
	Target    string            `yaml:"target,omitempty" json:"target,omitempty"` // for proxy: host:port or sip URI
	Trunks    []TrunkWeight     `yaml:"trunks,omitempty" json:"trunks,omitempty"`
	Algorithm string            `yaml:"algorithm,omitempty" json:"algorithm,omitempty"` // loadBalance: round_robin | weighted | consistent_hash
	Code      int               `yaml:"code,omitempty" json:"code,omitempty"`
	Reason    string            `yaml:"reason,omitempty" json:"reason,omitempty"`
	Extra     map[string]string `yaml:"extra,omitempty" json:"extra,omitempty"`
}

type TrunkWeight struct {
	Name   string `yaml:"name" json:"name"`
	Weight int    `yaml:"weight,omitempty" json:"weight,omitempty"`
}

// Snapshot is an immutable applied resource set.
type Snapshot struct {
	Revision  int64
	Tenants   map[string]*Tenant
	Endpoints []*Endpoint
	Trunks    map[string]*Trunk // key: tenant/name or name
	Routes    []*Route
	Secrets   map[string]string // secretRef -> password
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
