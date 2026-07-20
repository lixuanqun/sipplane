package transform

import (
	"regexp"
	"strings"
)

// Rule rewrites a SIP URI user part (number transform / LCR helper).
type Rule struct {
	Name    string
	Match   string // regex against user part
	Replace string // replacement; $1,$2 supported via regexp
}

// ApplyUser transforms the user portion of a SIP URI string.
// Input examples: "sip:+8613800138000@acme.example" or "+8613800138000".
func ApplyUser(uriOrUser string, rules []Rule) string {
	user, prefix, suffix := splitUser(uriOrUser)
	for _, r := range rules {
		if r.Match == "" {
			continue
		}
		re, err := regexp.Compile(r.Match)
		if err != nil {
			continue
		}
		if !re.MatchString(user) {
			continue
		}
		out := re.ReplaceAllString(user, r.Replace)
		return prefix + out + suffix
	}
	return uriOrUser
}

func splitUser(s string) (user, prefix, suffix string) {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, "@") && !strings.HasPrefix(strings.ToLower(s), "sip:") {
		return s, "", ""
	}
	// sip:user@host;params
	lower := strings.ToLower(s)
	start := 0
	if strings.HasPrefix(lower, "sip:") {
		start = 4
		prefix = s[:4]
	} else if strings.HasPrefix(lower, "sips:") {
		start = 5
		prefix = s[:5]
	}
	rest := s[start:]
	at := strings.IndexByte(rest, '@')
	if at < 0 {
		return rest, prefix, ""
	}
	return rest[:at], prefix, rest[at:]
}
