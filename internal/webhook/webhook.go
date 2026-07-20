package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/emiago/sipgo/sip"
	"github.com/sipplane/sipplane/internal/resources"
)

// Request is sent to an external policy webhook.
type Request struct {
	Method     string            `json:"method"`
	RequestURI string            `json:"requestUri"`
	CallID     string            `json:"callId"`
	From       string            `json:"from"`
	To         string            `json:"to"`
	Source     string            `json:"source"`
	Tenant     string            `json:"tenant,omitempty"`
	Route      string            `json:"route,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
}

// Response from external policy.
type Response struct {
	Action string `json:"action"` // continue | reject | proxy
	Code   int    `json:"code,omitempty"`
	Reason string `json:"reason,omitempty"`
	Target string `json:"target,omitempty"` // host:port for proxy
}

// Client calls an HTTP webhook with hard timeout + fallback (APISIX-style).
type Client struct {
	HTTP    *http.Client
	URL     string
	Timeout time.Duration
	// Fallback used when webhook errors / times out.
	Fallback Response
}

func New(url string, timeout time.Duration, fallback Response) *Client {
	if timeout == 0 {
		timeout = 500 * time.Millisecond
	}
	if fallback.Action == "" {
		fallback = Response{Action: "reject", Code: 503, Reason: "Webhook Unavailable"}
	}
	return &Client{
		HTTP:     &http.Client{Timeout: timeout},
		URL:      url,
		Timeout:  timeout,
		Fallback: fallback,
	}
}

// Decide asks the webhook; on failure returns Fallback.
func (c *Client) Decide(ctx context.Context, req *sip.Request, route *resources.Route) Response {
	if c == nil || c.URL == "" {
		return Response{Action: "continue"}
	}
	body := Request{
		Method:     req.Method.String(),
		RequestURI: req.Recipient.String(),
		Source:     req.Source(),
		Headers:    map[string]string{},
	}
	if cid := req.CallID(); cid != nil {
		body.CallID = cid.Value()
	}
	if f := req.From(); f != nil {
		body.From = f.Address.String()
	}
	if t := req.To(); t != nil {
		body.To = t.Address.String()
	}
	if route != nil {
		body.Tenant = route.Metadata.Tenant
		body.Route = route.Metadata.Name
	}
	raw, _ := json.Marshal(body)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.URL, bytes.NewReader(raw))
	if err != nil {
		return c.Fallback
	}
	httpReq.Header.Set("Content-Type", "application/json")
	res, err := c.HTTP.Do(httpReq)
	if err != nil {
		return c.Fallback
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil || res.StatusCode >= 300 {
		return c.Fallback
	}
	var out Response
	if err := json.Unmarshal(data, &out); err != nil || out.Action == "" {
		return c.Fallback
	}
	return out
}

// URLFromAction extracts webhook URL from route action extra/target.
func URLFromAction(a resources.RouteAction) string {
	if a.Target != "" {
		return a.Target
	}
	if a.Extra != nil {
		if u := a.Extra["url"]; u != "" {
			return u
		}
	}
	return ""
}

// ValidateURL is a tiny helper for tests.
func ValidateURL(u string) error {
	if u == "" {
		return fmt.Errorf("empty webhook url")
	}
	return nil
}
