package resources

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// LoadDir loads all .yaml/.yml files from a directory (or a single file path)
// into a Snapshot with revision 1.
func LoadDir(path string) (*Snapshot, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	var files []string
	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return nil, err
		}
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			name := e.Name()
			if isBootstrapFile(name) {
				continue
			}
			ext := strings.ToLower(filepath.Ext(name))
			if ext == ".yaml" || ext == ".yml" {
				files = append(files, filepath.Join(path, name))
			}
		}
		sort.Strings(files)
	} else {
		files = []string{path}
	}
	if len(files) == 0 {
		return nil, fmt.Errorf("resources: no yaml files in %s", path)
	}

	snap := &Snapshot{
		Revision:  1,
		Tenants:   make(map[string]*Tenant),
		Trunks:    make(map[string]*Trunk),
		Secrets:   make(map[string]string),
		Endpoints: nil,
		Routes:    nil,
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		if err := DecodeDocuments(data, snap); err != nil {
			return nil, fmt.Errorf("%s: %w", f, err)
		}
	}

	sort.SliceStable(snap.Routes, func(i, j int) bool {
		if snap.Routes[i].Spec.Priority != snap.Routes[j].Spec.Priority {
			return snap.Routes[i].Spec.Priority > snap.Routes[j].Spec.Priority
		}
		return snap.Routes[i].Metadata.Name < snap.Routes[j].Metadata.Name
	})
	return snap, nil
}

type kindMeta struct {
	APIVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind"`
	Metadata   ObjectMeta `yaml:"metadata"`
}

// ParseYAML parses multi-doc YAML into a new Snapshot (revision unset).
func ParseYAML(data []byte) (*Snapshot, error) {
	snap := &Snapshot{
		Tenants:   make(map[string]*Tenant),
		Trunks:    make(map[string]*Trunk),
		Secrets:   make(map[string]string),
		Endpoints: nil,
		Routes:    nil,
	}
	if err := DecodeDocuments(data, snap); err != nil {
		return nil, err
	}
	sort.SliceStable(snap.Routes, func(i, j int) bool {
		if snap.Routes[i].Spec.Priority != snap.Routes[j].Spec.Priority {
			return snap.Routes[i].Spec.Priority > snap.Routes[j].Spec.Priority
		}
		return snap.Routes[i].Metadata.Name < snap.Routes[j].Metadata.Name
	})
	return snap, nil
}

// DecodeDocuments appends resources from multi-doc YAML into snap.
func DecodeDocuments(data []byte, snap *Snapshot) error {
	return decodeInto(data, snap)
}

func decodeInto(data []byte, snap *Snapshot) error {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var raw map[string]any
		err := dec.Decode(&raw)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if raw == nil {
			continue
		}
		buf, err := yaml.Marshal(raw)
		if err != nil {
			return err
		}
		var km kindMeta
		if err := yaml.Unmarshal(buf, &km); err != nil {
			return err
		}
		if km.APIVersion == "" {
			km.APIVersion = APIVersion
		}
		switch km.Kind {
		case "Tenant":
			var t Tenant
			if err := yaml.Unmarshal(buf, &t); err != nil {
				return err
			}
			t.APIVersion = km.APIVersion
			t.Kind = km.Kind
			snap.Tenants[t.Metadata.Name] = &t
		case "Endpoint":
			var e Endpoint
			if err := yaml.Unmarshal(buf, &e); err != nil {
				return err
			}
			e.APIVersion = km.APIVersion
			e.Kind = km.Kind
			snap.Endpoints = append(snap.Endpoints, &e)
			if e.Spec.Auth.Password != "" && e.Spec.Auth.PasswordSecretRef != "" {
				snap.Secrets[e.Spec.Auth.PasswordSecretRef] = e.Spec.Auth.Password
			}
			if e.Spec.Auth.Password != "" && e.Spec.Auth.PasswordSecretRef == "" {
				key := "inline/" + e.Metadata.Tenant + "/" + e.Metadata.Name
				snap.Secrets[key] = e.Spec.Auth.Password
				e.Spec.Auth.PasswordSecretRef = key
			}
		case "Trunk":
			var tr Trunk
			if err := yaml.Unmarshal(buf, &tr); err != nil {
				return err
			}
			tr.APIVersion = km.APIVersion
			tr.Kind = km.Kind
			key := trunkKey(tr.Metadata.Tenant, tr.Metadata.Name)
			snap.Trunks[key] = &tr
			snap.Trunks[tr.Metadata.Name] = &tr
			if tr.Spec.Auth.Outbound != nil && tr.Spec.Auth.Outbound.Password != "" {
				ref := tr.Spec.Auth.Outbound.PasswordSecretRef
				if ref == "" {
					ref = "inline/trunk/" + key
					tr.Spec.Auth.Outbound.PasswordSecretRef = ref
				}
				snap.Secrets[ref] = tr.Spec.Auth.Outbound.Password
			}
		case "Route":
			var r Route
			if err := yaml.Unmarshal(buf, &r); err != nil {
				return err
			}
			r.APIVersion = km.APIVersion
			r.Kind = km.Kind
			snap.Routes = append(snap.Routes, &r)
		case "Secret":
			var sec struct {
				Metadata ObjectMeta `yaml:"metadata"`
				Spec     struct {
					Data map[string]string `yaml:"data"`
				} `yaml:"spec"`
			}
			if err := yaml.Unmarshal(buf, &sec); err != nil {
				return err
			}
			for k, v := range sec.Spec.Data {
				snap.Secrets[sec.Metadata.Name+"/"+k] = v
				snap.Secrets[sec.Metadata.Name] = v
			}
		default:
			return fmt.Errorf("unknown kind %q", km.Kind)
		}
	}
}

func trunkKey(tenant, name string) string {
	if tenant == "" {
		return name
	}
	return tenant + "/" + name
}

// FindEndpointByAOR returns the endpoint that owns the given AOR.
func (s *Snapshot) FindEndpointByAOR(aor string) *Endpoint {
	aor = strings.ToLower(strings.TrimSpace(aor))
	for _, ep := range s.Endpoints {
		for _, a := range ep.Spec.AORs {
			if strings.ToLower(a) == aor {
				return ep
			}
		}
	}
	return nil
}

// FindEndpointByUsername finds endpoint by auth username (optionally scoped by tenant).
func (s *Snapshot) FindEndpointByUsername(username, tenant string) *Endpoint {
	for _, ep := range s.Endpoints {
		if tenant != "" && ep.Metadata.Tenant != "" && ep.Metadata.Tenant != tenant {
			continue
		}
		if ep.Spec.Auth.Username == username {
			return ep
		}
	}
	return nil
}

// ResolvePassword returns the password for a secret ref.
func (s *Snapshot) ResolvePassword(ref string) (string, bool) {
	if ref == "" {
		return "", false
	}
	v, ok := s.Secrets[ref]
	return v, ok
}

// GetTrunk returns a trunk by name (with optional tenant prefix).
func (s *Snapshot) GetTrunk(tenant, name string) *Trunk {
	if t := s.Trunks[trunkKey(tenant, name)]; t != nil {
		return t
	}
	return s.Trunks[name]
}

// isBootstrapFile reports data-plane bootstrap YAML that must not be parsed as resources.
func isBootstrapFile(name string) bool {
	base := strings.ToLower(filepath.Base(name))
	return base == "bootstrap.yaml" || base == "bootstrap.yml" ||
		strings.HasPrefix(base, "bootstrap-") && (strings.HasSuffix(base, ".yaml") || strings.HasSuffix(base, ".yml"))
}
