package amaquet

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type adminSeen struct {
	Method string
	Path   string
	Query  string
	Bearer string
	Body   map[string]any
}

func TestSDKAdminRouteCoverage(t *testing.T) {
	var mu sync.Mutex
	seen := []adminSeen{}
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var body map[string]any
		if r.Header.Get("Content-Type") == "application/json" {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}
		mu.Lock()
		seen = append(seen, adminSeen{r.Method, r.URL.Path, r.URL.RawQuery, r.Header.Get("Authorization"), body})
		mu.Unlock()
		if r.URL.Path == "/api/session" && r.Method == "POST" {
			http.SetCookie(w, &http.Cookie{Name: "amaquet_session", Value: "sid", Path: "/"})
		}
		if r.URL.Path == "/metrics" {
			w.Header().Set("Content-Type", "text/plain")
			_, _ = w.Write([]byte("amaquet_keys 12\n"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":{}}`))
	}))
	defer h.Close()
	a, err := NewAdminClient(h.URL, AdminOptions{Token: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	type step struct {
		path, method string
		fn           func() error
	}
	raw := func(v json.RawMessage, e error) error { return e }
	steps := []step{
		{"/openapi.json", "GET", func() error { return raw(a.OpenAPI(ctx)) }},
		{"/api/health", "GET", func() error { return raw(a.Health(ctx)) }},
		{"/api/ready", "GET", func() error { return raw(a.Ready(ctx)) }},
		{"/metrics", "GET", func() error {
			s, e := a.Metrics(ctx)
			if s != "amaquet_keys 12\n" {
				t.Error(s)
			}
			return e
		}},
		{"/api/auth/check", "GET", func() error { return raw(a.CheckAuth(ctx)) }},
		{"/api/session", "POST", func() error { return raw(a.CreateSession(ctx, "token", "123456")) }},
		{"/api/session", "DELETE", func() error { return a.DeleteSession(ctx) }},
		{"/api/identity/accept-invite", "POST", func() error { return raw(a.AcceptInvite(ctx, "inv")) }},
		{"/api/identity/mfa", "POST", func() error { return raw(a.BeginMFA(ctx)) }},
		{"/api/identity/mfa", "PUT", func() error { return raw(a.ConfirmMFA(ctx, "123456")) }},
		{"/api/identity/mfa", "DELETE", func() error { return raw(a.DisableMFA(ctx, "123456")) }},
		{"/api/overview", "GET", func() error { return raw(a.Overview(ctx)) }},
		{"/api/organization", "GET", func() error { return raw(a.Organization(ctx)) }},
		{"/api/organization", "PUT", func() error { return raw(a.UpdateOrganization(ctx, "name", "slug")) }},
		{"/api/members", "GET", func() error { return raw(a.Members(ctx)) }},
		{"/api/members", "POST", func() error { return raw(a.AddMember(ctx, "a", "a@b.c", "admin")) }},
		{"/api/members/abc", "PATCH", func() error { return raw(a.PatchMember(ctx, "abc", "admin", "active")) }},
		{"/api/members/abc", "DELETE", func() error { return a.DeleteMember(ctx, "abc") }},
		{"/api/members/abc/invite", "POST", func() error { return raw(a.InviteMember(ctx, "abc", 24)) }},
		{"/api/rbac", "GET", func() error { return raw(a.RBAC(ctx)) }},
		{"/api/rbac", "PUT", func() error { return raw(a.UpdateRBAC(ctx, map[string][]string{"admin": {"data.read"}})) }},
		{"/api/api-keys", "GET", func() error { return raw(a.APIKeys(ctx)) }},
		{"/api/api-keys", "POST", func() error { return raw(a.CreateAPIKey(ctx, "name", "developer", nil)) }},
		{"/api/api-keys/abc", "DELETE", func() error { return a.RevokeAPIKey(ctx, "abc") }},
		{"/api/system", "GET", func() error { return raw(a.System(ctx)) }},
		{"/api/system", "PUT", func() error { return raw(a.UpdateSystem(ctx, map[string]any{"demo": true})) }},
		{"/api/persistence/checkpoint", "POST", func() error { return raw(a.Checkpoint(ctx, "nightly.aof")) }},
		{"/api/persistence/compact", "POST", func() error { return a.Compact(ctx) }},
		{"/api/data/keys", "GET", func() error { return raw(a.DataKeys(ctx, "cursor", "foo", 20)) }},
		{"/api/data/key", "GET", func() error { return raw(a.DataKey(ctx, "a&b?#")) }},
		{"/api/data/key", "DELETE", func() error { return a.DeleteDataKey(ctx, "a&b?#") }},
		{"/api/data/command", "POST", func() error { return raw(a.DataCommand(ctx, "PING", nil)) }},
		{"/api/audit", "GET", func() error { return raw(a.Audit(ctx, 50)) }},
	}
	for i, s := range steps {
		if err := s.fn(); err != nil {
			t.Errorf("step %d: %s %s: %v", i, s.method, s.path, err)
		}
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != len(steps) {
		t.Fatalf("received %d of %d requests", len(seen), len(steps))
	}
	for i, s := range steps {
		if seen[i].Method != s.method || seen[i].Path != s.path || seen[i].Bearer != "Bearer secret" {
			t.Errorf("step %d mismatch: %#v", i, seen[i])
		}
	}
	if !strings.Contains(seen[len(steps)-3].Query, "key=a%26b%3F%23") {
		t.Errorf("query not escaped: %q", seen[len(steps)-3].Query)
	}
}
func TestSDKAdminSecurityFailures(t *testing.T) {
	if _, err := NewAdminClient("https://user:pass@host", AdminOptions{}); err == nil {
		t.Fatal("accepted credentials in URL")
	}
	if _, err := NewAdminClient("http://host/a", AdminOptions{}); err == nil {
		t.Fatal("accepted base path")
	}
	if _, err := NewAdminClient("file:///etc/passwd", AdminOptions{}); err == nil {
		t.Fatal("accepted file scheme")
	}
	if _, err := adminID("../evil"); err == nil {
		t.Fatal("accepted slash in id")
	}
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/bad":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"ok":false,"error":{"code":"FORBIDDEN","message":"no"}}`))
		case "/large":
			_, _ = w.Write([]byte(strings.Repeat("x", 100)))
		case "/redirect":
			http.Redirect(w, r, "https://example.com", http.StatusFound)
		default:
			_, _ = w.Write([]byte(`{"ok":true}`))
		}
	}))
	defer h.Close()
	a, err := NewAdminClient(h.URL, AdminOptions{MaxResponseBytes: 64})
	if err != nil {
		t.Fatal(err)
	}
	var v json.RawMessage
	err = a.RequestJSON(context.Background(), "GET", "/bad", nil, &v)
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.StatusCode != 403 || httpErr.Code != "FORBIDDEN" {
		t.Fatal(err)
	}
	err = a.RequestJSON(context.Background(), "GET", "/large", nil, &v)
	if err == nil || !strings.Contains(err.Error(), "exceeds") {
		t.Fatal(err)
	}
	err = a.RequestJSON(context.Background(), "GET", "/redirect", nil, &v)
	if !errors.As(err, &httpErr) || httpErr.StatusCode != 302 {
		t.Fatal(err)
	}
	err = a.RequestJSON(context.Background(), "GET", "/../secret", nil, &v)
	if err == nil {
		t.Fatal("path traversal")
	}
}
func TestSDKAdminUsesSessionCookie(t *testing.T) {
	var cookieSeen bool
	h := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/session" {
			http.SetCookie(w, &http.Cookie{Name: "amaquet_session", Value: "sid", Path: "/", Expires: time.Now().Add(time.Hour)})
			_, _ = w.Write([]byte(`{"ok":true}`))
			return
		}
		cookieSeen = r.Header.Get("Cookie") == "amaquet_session=sid"
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer h.Close()
	a, e := NewAdminClient(h.URL, AdminOptions{})
	if e != nil {
		t.Fatal(e)
	}
	if _, e = a.CreateSession(context.Background(), "token", ""); e != nil {
		t.Fatal(e)
	}
	if _, e = a.CheckAuth(context.Background()); e != nil {
		t.Fatal(e)
	}
	if !cookieSeen {
		t.Fatal("session cookie not sent")
	}
}
