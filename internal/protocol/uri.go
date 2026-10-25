package protocol

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
)

const DefaultPort = 13378

// Endpoint: Provides the Endpoint operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Parses and validates Amaquet URI schemes, credentials policy, host and port.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
type Endpoint struct {
	TLS      bool
	Host     string
	Port     int
	Database string
	APIKey   string
}

// ParseEndpoint: Parse an amaquet:// or amaquets:// address and reject unsupported schemes and paths.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func ParseEndpoint(raw string) (Endpoint, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return Endpoint{}, err
	}

	if u.Scheme != "amaquet" && u.Scheme != "amaquets" {
		return Endpoint{}, fmt.Errorf("scheme must be amaquet or amaquets")
	}

	host := u.Hostname()
	if host == "" {
		return Endpoint{}, fmt.Errorf("host is required")
	}

	port := DefaultPort
	if u.Port() != "" {
		p, e := strconv.Atoi(u.Port())
		if e != nil || p < 1 || p > 65535 {
			return Endpoint{}, fmt.Errorf("invalid port")
		}

		port = p
	}

	db := u.Path
	if len(db) > 0 && db[0] == '/' {
		db = db[1:]
	}

	if db != "" {
		return Endpoint{}, fmt.Errorf("logical database paths are not supported; omit /%s", db)
	}

	apiKey := ""
	if u.User != nil {
		if p, ok := u.User.Password(); ok {
			apiKey = p
		} else {
			apiKey = u.User.Username()
		}
	}

	if q := u.Query().Get("api_key"); q != "" {
		apiKey = q
	}

	return Endpoint{
		TLS:      u.Scheme == "amaquets",
		Host:     host,
		Port:     port,
		Database: db,
		APIKey:   apiKey,
	}, nil
}

// Address: Provides the Address operation or value as implemented by this Amaquet SDK component; follow the server protocol for accepted arguments.
// Failures: Invalid arguments, protocol errors, remote errors, timeout, or connection loss may be reported as appropriate to this API.
func (e Endpoint) Address() string {
	return net.JoinHostPort(e.Host, strconv.Itoa(e.Port))
}
