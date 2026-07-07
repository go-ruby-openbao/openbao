// Copyright (c) the go-ruby-openbao/openbao authors
//
// SPDX-License-Identifier: BSD-3-Clause

package openbao

import (
	"encoding/json"
	"os"
	"strings"
	"time"
)

// DefaultAddress is the address used when neither the config nor the environment
// supplies one, matching the vault gem's default.
const DefaultAddress = "https://127.0.0.1:8200"

// Config configures a [Client]. Any zero field is resolved from the environment
// (Address from VAULT_ADDR/BAO_ADDR, Token from VAULT_TOKEN/BAO_TOKEN, Namespace
// from VAULT_NAMESPACE/BAO_NAMESPACE) and finally from the built-in defaults.
type Config struct {
	// Address is the OpenBao/Vault server address (scheme+host+port).
	Address string
	// Token is the Vault token sent as X-Vault-Token.
	Token string
	// Namespace is the Vault namespace sent as X-Vault-Namespace (Enterprise /
	// OpenBao namespaces).
	Namespace string
	// Timeout is the per-request timeout of the default net/http transport; 0
	// means no timeout. Ignored when Doer is set.
	Timeout time.Duration
	// Doer overrides the transport seam. When nil, a net/http-backed Doer is
	// used. Tests and hosts inject their own.
	Doer Doer
}

// Client is a configured OpenBao/Vault API client (Vault::Client /
// OpenBao::Client). It carries the resolved address, token and namespace, and
// the transport [Doer]; the endpoint groups are reached through
// [Client.Logical], [Client.KVv1], [Client.KVv2], [Client.Transit],
// [Client.Sys] and [Client.Auth].
type Client struct {
	address   string
	token     string
	namespace string
	doer      Doer
}

// NewClient builds a [Client] from cfg, resolving unset fields from the
// environment and the defaults.
func NewClient(cfg Config) *Client {
	doer := cfg.Doer
	if doer == nil {
		doer = defaultDoer(cfg.Timeout)
	}
	return &Client{
		address:   resolveAddress(cfg.Address),
		token:     firstNonEmpty(cfg.Token, os.Getenv("VAULT_TOKEN"), os.Getenv("BAO_TOKEN")),
		namespace: firstNonEmpty(cfg.Namespace, os.Getenv("VAULT_NAMESPACE"), os.Getenv("BAO_NAMESPACE")),
		doer:      doer,
	}
}

// resolveAddress picks the explicit address, then the environment, then the
// default, and trims any trailing slash.
func resolveAddress(explicit string) string {
	addr := firstNonEmpty(explicit, os.Getenv("VAULT_ADDR"), os.Getenv("BAO_ADDR"), DefaultAddress)
	return strings.TrimRight(addr, "/")
}

// firstNonEmpty returns the first non-empty string in vals, or "".
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// Address returns the resolved server address.
func (c *Client) Address() string { return c.address }

// Token returns the current Vault token.
func (c *Client) Token() string { return c.token }

// SetToken sets the Vault token and returns the client for chaining.
func (c *Client) SetToken(token string) *Client {
	c.token = token
	return c
}

// Namespace returns the current Vault namespace.
func (c *Client) Namespace() string { return c.namespace }

// SetNamespace sets the Vault namespace and returns the client for chaining.
func (c *Client) SetNamespace(ns string) *Client {
	c.namespace = ns
	return c
}

// AdoptToken adopts the client_token from a login/auth [Secret] as the client's
// token (Vault::Client#token=), returning the client for chaining. A nil secret
// or one without an auth client token leaves the token unchanged.
func (c *Client) AdoptToken(s *Secret) *Client {
	if s != nil && s.Auth != nil && s.Auth.ClientToken != "" {
		c.token = s.Auth.ClientToken
	}
	return c
}

// Logical returns the logical-backend endpoint group (Vault::Client#logical).
func (c *Client) Logical() *Logical { return &Logical{c: c} }

// KVv1 returns a KV version-1 helper rooted at mount (default "secret").
func (c *Client) KVv1(mount string) *KVv1 {
	return &KVv1{c: c, mount: mountOr(mount, "secret")}
}

// KVv2 returns a KV version-2 helper rooted at mount (default "secret").
func (c *Client) KVv2(mount string) *KVv2 {
	return &KVv2{c: c, mount: mountOr(mount, "secret")}
}

// Transit returns a transit-backend helper rooted at mount (default "transit").
func (c *Client) Transit(mount string) *Transit {
	return &Transit{c: c, mount: mountOr(mount, "transit")}
}

// Sys returns the system-backend endpoint group (Vault::Client#sys).
func (c *Client) Sys() *Sys { return &Sys{c: c} }

// Auth returns the auth-method endpoint group (Vault::Client#auth).
func (c *Client) Auth() *Auth { return &Auth{c: c} }

// mountOr returns mount, or def when mount is empty.
func mountOr(mount, def string) string {
	if mount == "" {
		return def
	}
	return strings.Trim(mount, "/")
}

// request performs a single API call: it JSON-encodes body (when non-nil),
// builds the /v1/ [Request], runs it through the [Doer], maps a transport
// failure to an HTTPConnectionError and a non-2xx status to the matching
// HTTP error, and returns the raw 2xx [Response].
func (c *Client) request(method, path string, body any) (*Response, error) {
	var raw []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, newError(err)
		}
		raw = b
	}

	req := &Request{
		Method:  method,
		URL:     c.address + "/v1/" + strings.TrimLeft(path, "/"),
		Headers: map[string]string{},
		Body:    raw,
	}
	if c.token != "" {
		req.Headers["X-Vault-Token"] = c.token
	}
	if c.namespace != "" {
		req.Headers["X-Vault-Namespace"] = c.namespace
	}
	if raw != nil {
		req.Headers["Content-Type"] = "application/json"
	}

	resp, err := c.doer.Do(req)
	if err != nil {
		return nil, newConnectionError(err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, newResponseError(resp.StatusCode, parseErrors(resp.Body))
	}
	return resp, nil
}

// do runs request and decodes a 2xx body into a [Secret]. An empty body (204 No
// Content) yields a nil Secret and nil error.
func (c *Client) do(method, path string, body any) (*Secret, error) {
	resp, err := c.request(method, path, body)
	if err != nil {
		return nil, err
	}
	if len(resp.Body) == 0 {
		return nil, nil
	}
	var s Secret
	if err := json.Unmarshal(resp.Body, &s); err != nil {
		return nil, newError(err)
	}
	return &s, nil
}

// doRaw runs request and decodes a 2xx body into a generic map, for endpoints
// (health, seal-status, mounts) whose payload is not wrapped in the Secret
// envelope. An empty body yields a nil map and nil error.
func (c *Client) doRaw(method, path string, body any) (map[string]any, error) {
	resp, err := c.request(method, path, body)
	if err != nil {
		return nil, err
	}
	if len(resp.Body) == 0 {
		return nil, nil
	}
	var m map[string]any
	if err := json.Unmarshal(resp.Body, &m); err != nil {
		return nil, newError(err)
	}
	return m, nil
}

// parseErrors extracts the API "errors" array from a response body, returning
// nil when the body is empty or not the expected shape.
func parseErrors(body []byte) []string {
	if len(body) == 0 {
		return nil
	}
	var e struct {
		Errors []string `json:"errors"`
	}
	if err := json.Unmarshal(body, &e); err != nil {
		return nil
	}
	return e.Errors
}
