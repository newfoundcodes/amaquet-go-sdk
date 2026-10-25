package amaquet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// AdminClient uses the separate HTTP control-plane listener, not the Amaquet TCP port.
// AdminClient: Provides the Admin Client operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Provides the separate HTTP control-plane API, including sessions, organizations, RBAC, API keys, audit, metrics, and configuration.
// Control-plane requests use a separate HTTP(S) base URL and may require administrative authorization.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type AdminClient struct {
	endpoint    *url.URL
	client      *http.Client
	token       string
	maxResponse int64
}

// AdminOptions controls HTTP authentication, transport and defensive response limits.
// AdminOptions: Provides the Admin Options operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Provides the separate HTTP control-plane API, including sessions, organizations, RBAC, API keys, audit, metrics, and configuration.
// Control-plane requests use a separate HTTP(S) base URL and may require administrative authorization.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type AdminOptions struct {
	// Token is the administrative bearer token or session credential.
	Token string
	// HTTPClient supplies the HTTP transport used for administration requests.
	HTTPClient *http.Client
	// MaxResponseBytes limits the size of an administration response body.
	MaxResponseBytes int64
}

// HTTPError includes the HTTP status and the server's JSON error details.
// HTTPError: Represents an Amaquet httperror with diagnostic information for callers; avoid logging credentials or private payloads.
// Provides the separate HTTP control-plane API, including sessions, organizations, RBAC, API keys, audit, metrics, and configuration.
// Control-plane requests use a separate HTTP(S) base URL and may require administrative authorization.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type HTTPError struct {
	// StatusCode is the HTTP status returned by the administration endpoint.
	StatusCode int
	// Code is the server-defined machine-readable error code.
	Code string
	// Message is the server-defined human-readable error message.
	Message string
	// Body contains the raw response body when structured error details are unavailable.
	Body string
}

// Error: Represents an Amaquet error with diagnostic information for callers; avoid logging credentials or private payloads.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (e *HTTPError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("amaquet HTTP %d %s: %s", e.StatusCode, e.Code, e.Message)
	}

	return fmt.Sprintf("amaquet HTTP %d: %s", e.StatusCode, e.Body)
}

// NewAdminClient: Provides the New Admin Client operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `endpoint`: Host and port of the native service or HTTP administration origin.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func NewAdminClient(endpoint string, opts AdminOptions) (*AdminClient, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	if (u.Scheme != "http" && u.Scheme != "https") ||
		u.Host == "" ||
		u.User != nil ||
		u.RawQuery != "" ||
		u.Fragment != "" ||
		u.Path != "" && u.Path != "/" {
		return nil, errors.New("admin endpoint must be a bare http(s) origin without credentials, query or path")
	}

	hc := opts.HTTPClient
	if hc == nil {
		hc = &http.Client{Timeout: 30 * time.Second}
	}

	copyClient := *hc
	if copyClient.Jar == nil {
		jar, e := cookiejar.New(nil)
		if e != nil {
			return nil, e
		}

		copyClient.Jar = jar
	}

	if copyClient.CheckRedirect == nil {
		copyClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	limit := opts.MaxResponseBytes
	if limit <= 0 {
		limit = 16 << 20
	}

	return &AdminClient{
		endpoint:    u,
		client:      &copyClient,
		token:       opts.Token,
		maxResponse: limit,
	}, nil
}

// URL: Provides the URL operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) URL() string {
	if a == nil {
		return ""
	}

	return a.endpoint.String()
}

// RequestJSON makes a single request to a relative administration path. It does not follow redirects.
// Use nil out for responses that have no JSON body (for example DELETE /api/session).
// RequestJSON: Provides the Request JSON operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `out`: Destination to decode the server response; preserve numeric precision.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) RequestJSON(ctx context.Context, method, path string, in, out any) error {
	_, err := a.request(ctx, method, path, in, out, false)
	return err
}

// request: Submit an operation with a request ID and await its matching response or a timeout or connection error.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `out`: Destination to decode the server response; preserve numeric precision.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) request(ctx context.Context, method, path string, in, out any, textResponse bool) (string, error) {
	if a == nil || a.endpoint == nil {
		return "", errors.New("nil admin client")
	}

	if !strings.HasPrefix(path, "/") || strings.HasPrefix(path, "//") || strings.Contains(path, "#") || strings.Contains(path, "\\") {
		return "", errors.New("path must be relative to admin origin")
	}

	p, err := url.ParseRequestURI(path)
	if err != nil {
		return "", err
	}

	if strings.HasPrefix(p.Path, "//") ||
		strings.Contains(p.Path, "/../") ||
		strings.HasSuffix(p.Path, "/..") ||
		strings.Contains(p.Path, "/./") {
		return "", errors.New("path traversal rejected")
	}

	var rd io.Reader
	if in != nil {
		b, e := json.Marshal(in)
		if e != nil {
			return "", e
		}
		rd = bytes.NewReader(b)
	}

	target := a.endpoint.ResolveReference(p)
	req, err := http.NewRequestWithContext(ctx, method, target.String(), rd)

	if err != nil {
		return "", err
	}

	req.Header.Set("Accept", "application/json")
	if textResponse {
		req.Header.Set("Accept", "text/plain")
	}

	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if a.token != "" {
		req.Header.Set("Authorization", "Bearer "+a.token)
	}

	resp, err := a.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, a.maxResponse+1))
	if err != nil {
		return "", err
	}

	if int64(len(data)) > a.maxResponse {
		return "", fmt.Errorf("admin response exceeds %d bytes", a.maxResponse)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		e := &HTTPError{StatusCode: resp.StatusCode, Body: string(data)}

		var msg struct {
			Code    string          `json:"code"`
			Error   json.RawMessage `json:"error"`
			Message string          `json:"message"`
		}

		if json.Unmarshal(data, &msg) == nil {
			e.Code = msg.Code
			e.Message = msg.Message

			if len(msg.Error) > 0 {
				var s string
				if json.Unmarshal(msg.Error, &s) == nil {
					e.Message = s
				} else {
					var x struct {
						Code    string `json:"code"`
						Message string `json:"message"`
					}

					if json.Unmarshal(msg.Error, &x) == nil {
						if e.Code == "" {
							e.Code = x.Code
						}

						if e.Message == "" {
							e.Message = x.Message
						}
					}
				}
			}
		}

		return "", e
	}

	if textResponse {
		return string(data), nil
	}

	if out != nil && len(data) > 0 {
		if err := decodeNumber(data, out); err != nil {
			return "", err
		}
	}

	return "", nil
}

// AdminResult keeps the server's native JSON shape intact across different versions.
type AdminResult = json.RawMessage

// raw: Provides the raw operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) raw(ctx context.Context, method, path string, in any) (json.RawMessage, error) {
	var v json.RawMessage
	err := a.RequestJSON(ctx, method, path, in, &v)
	return v, err
}

// OpenAPI: Provides the Open API operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) OpenAPI(ctx context.Context) (json.RawMessage, error) {
	return a.raw(ctx, "GET", "/openapi.json", nil)
}

// Health: Query the administration service health endpoint; it does not authenticate a native connection.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) Health(ctx context.Context) (json.RawMessage, error) {
	return a.raw(ctx, "GET", "/api/health", nil)
}

// Ready: Ready using the implementation's validation, error, and resource-lifetime rules.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) Ready(ctx context.Context) (json.RawMessage, error) {
	return a.raw(ctx, "GET", "/api/ready", nil)
}

// Metrics: Retrieve Prometheus-format text from the HTTP control plane, not a JSON document.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) Metrics(ctx context.Context) (string, error) {
	return a.request(ctx, "GET", "/metrics", nil, nil, true)
}

// CheckAuth: Provides the Check Auth operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) CheckAuth(ctx context.Context) (json.RawMessage, error) {
	return a.raw(ctx, "GET", "/api/auth/check", nil)
}

// CreateSession: Provides the Create Session operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `token`: Administrative token; avoid sending it over untrusted HTTP.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) CreateSession(ctx context.Context, token, mfaCode string) (json.RawMessage, error) {
	return a.raw(ctx, "POST", "/api/session", map[string]any{"token": token, "mfa_code": mfaCode})
}

// DeleteSession: Provides the Delete Session operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) DeleteSession(ctx context.Context) error {
	return a.RequestJSON(ctx, "DELETE", "/api/session", nil, nil)
}

// AcceptInvite: Provides the Accept Invite operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) AcceptInvite(ctx context.Context, invite string) (json.RawMessage, error) {
	return a.raw(ctx, "POST", "/api/identity/accept-invite", map[string]any{"invite": invite})
}

// BeginMFA: Provides the Begin MFA operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) BeginMFA(ctx context.Context) (json.RawMessage, error) {
	return a.raw(ctx, "POST", "/api/identity/mfa", nil)
}

// ConfirmMFA: Provides the Confirm MFA operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) ConfirmMFA(ctx context.Context, code string) (json.RawMessage, error) {
	return a.raw(ctx, "PUT", "/api/identity/mfa", map[string]string{"code": code})
}

// DisableMFA: Provides the Disable MFA operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) DisableMFA(ctx context.Context, code string) (json.RawMessage, error) {
	return a.raw(ctx, "DELETE", "/api/identity/mfa", map[string]string{"code": code})
}

// Overview: Provides the Overview operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) Overview(ctx context.Context) (json.RawMessage, error) {
	return a.raw(ctx, "GET", "/api/overview", nil)
}

// Organization: Provides the Organization operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) Organization(ctx context.Context) (json.RawMessage, error) {
	return a.raw(ctx, "GET", "/api/organization", nil)
}

// UpdateOrganization: Provides the Update Organization operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `name`: Native command or type operation name sent to the server.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) UpdateOrganization(ctx context.Context, name, slug string) (json.RawMessage, error) {
	return a.raw(ctx, "PUT", "/api/organization", map[string]string{"name": name, "slug": slug})
}

// Members: Provides the Members operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) Members(ctx context.Context) (json.RawMessage, error) {
	return a.raw(ctx, "GET", "/api/members", nil)
}

// AddMember: Provides the Add Member operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `name`: Native command or type operation name sent to the server.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) AddMember(ctx context.Context, name, email, role string) (json.RawMessage, error) {
	return a.raw(ctx, "POST", "/api/members", map[string]string{
		"name":  name,
		"email": email,
		"role":  role,
	})
}

// adminID: Provides the admin ID operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Control-plane requests use a separate HTTP(S) base URL and may require administrative authorization.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func adminID(id string) (string, error) {
	if id == "" || strings.ContainsAny(id, "/\\\x00") || id == "." || id == ".." {
		return "", errors.New("invalid resource id")
	}

	return url.PathEscape(id), nil
}

// PatchMember: Provides the Patch Member operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) PatchMember(ctx context.Context, id, role, status string) (json.RawMessage, error) {
	key, e := adminID(id)
	if e != nil {
		return nil, e
	}

	m := map[string]string{"role": role}
	if status != "" {
		m["status"] = status
	}

	return a.raw(ctx, "PATCH", "/api/members/"+key, m)
}

// DeleteMember: Provides the Delete Member operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) DeleteMember(ctx context.Context, id string) error {
	key, e := adminID(id)
	if e != nil {
		return e
	}

	return a.RequestJSON(ctx, "DELETE", "/api/members/"+key, nil, nil)
}

// InviteMember: Provides the Invite Member operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) InviteMember(ctx context.Context, id string, ttlHours int) (json.RawMessage, error) {
	key, e := adminID(id)
	if e != nil {
		return nil, e
	}

	return a.raw(ctx, "POST", "/api/members/"+key+"/invite", map[string]int{
		"ttl_hours": ttlHours,
	})
}

// RBAC: Provides the RBAC operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) RBAC(ctx context.Context) (json.RawMessage, error) {
	return a.raw(ctx, "GET", "/api/rbac", nil)
}

// UpdateRBAC: Provides the Update RBAC operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) UpdateRBAC(ctx context.Context, permissions map[string][]string) (json.RawMessage, error) {
	return a.raw(ctx, "PUT", "/api/rbac", permissions)
}

// APIKeys: Provides the APIKeys operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) APIKeys(ctx context.Context) (json.RawMessage, error) {
	return a.raw(ctx, "GET", "/api/api-keys", nil)
}

// CreateAPIKey: Provides the Create APIKey operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `name`: Native command or type operation name sent to the server.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) CreateAPIKey(ctx context.Context, name, role string, expiresAt *time.Time) (json.RawMessage, error) {
	return a.raw(ctx, "POST", "/api/api-keys", map[string]any{
		"name":       name,
		"role":       role,
		"expires_at": expiresAt,
	})
}

// RevokeAPIKey: Provides the Revoke APIKey operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) RevokeAPIKey(ctx context.Context, id string) error {
	key, e := adminID(id)
	if e != nil {
		return e
	}

	return a.RequestJSON(ctx, "DELETE", "/api/api-keys/"+key, nil, nil)
}

// System: Provides the System operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) System(ctx context.Context) (json.RawMessage, error) {
	return a.raw(ctx, "GET", "/api/system", nil)
}

// UpdateSystem: Provides the Update System operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) UpdateSystem(ctx context.Context, fullConfig any) (json.RawMessage, error) {
	return a.raw(ctx, "PUT", "/api/system", fullConfig)
}

// Checkpoint: Provides the Checkpoint operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `name`: Native command or type operation name sent to the server.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) Checkpoint(ctx context.Context, name string) (json.RawMessage, error) {
	return a.raw(ctx, "POST", "/api/persistence/checkpoint", map[string]string{"name": name})
}

// Compact: Provides the Compact operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) Compact(ctx context.Context) error {
	return a.RequestJSON(ctx, "POST", "/api/persistence/compact", nil, nil)
}

// DataKeys: Provides the Data Keys operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) DataKeys(ctx context.Context, cursor, prefix string, count int) (json.RawMessage, error) {
	q := url.Values{}
	q.Set("cursor", cursor)
	q.Set("prefix", prefix)
	q.Set("count", fmt.Sprint(count))

	return a.raw(ctx, "GET", "/api/data/keys?"+q.Encode(), nil)
}

// DataKey: Provides the Data Key operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) DataKey(ctx context.Context, key string) (json.RawMessage, error) {
	return a.raw(ctx, "GET", "/api/data/key?key="+url.QueryEscape(key), nil)
}

// DeleteDataKey: Provides the Delete Data Key operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `key`: Target Amaquet key; use unique keys for examples and tests.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) DeleteDataKey(ctx context.Context, key string) error {
	return a.RequestJSON(ctx, "DELETE", "/api/data/key?key="+url.QueryEscape(key), nil, nil)
}

// DataCommand: Provides the Data Command operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Parameter `args`: Exact JSON argument properties supported by the server; names are not translated.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) DataCommand(ctx context.Context, command string, args map[string]any) (json.RawMessage, error) {
	return a.raw(ctx, "POST", "/api/data/command", map[string]any{
		"command": command,
		"args":    args,
	})
}

// Audit: Provides the Audit operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parameter `ctx`: Request-scoped cancellation and deadline context.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (a *AdminClient) Audit(ctx context.Context, limit int) (json.RawMessage, error) {
	return a.raw(ctx, "GET", "/api/audit?limit="+fmt.Sprint(limit), nil)
}
