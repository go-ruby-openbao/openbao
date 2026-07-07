// Copyright (c) the go-ruby-openbao/openbao authors
//
// SPDX-License-Identifier: BSD-3-Clause

package openbao

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// secretJSON is the canonical Secret envelope the mock returns for reads/writes.
const secretJSON = `{
  "request_id": "req-123",
  "lease_id": "lease-abc",
  "lease_duration": 3600,
  "renewable": true,
  "data": {"foo": "bar", "ciphertext": "vault:v1:abcd", "plaintext": "aGVsbG8=", "signature": "vault:v1:sig", "valid": true},
  "warnings": ["a warning"],
  "wrap_info": null
}`

// authJSON is the envelope the mock returns for login/token endpoints.
const authJSON = `{
  "request_id": "req-456",
  "auth": {
    "client_token": "s.newtoken",
    "accessor": "acc-1",
    "policies": ["default"],
    "token_policies": ["default"],
    "metadata": {"role": "web"},
    "lease_duration": 3600,
    "renewable": true
  }
}`

// flatJSON is the un-enveloped payload the mock returns for sys/health etc.
const flatJSON = `{"initialized": true, "sealed": false, "standby": false, "version": "1.0.0"}`

// newMockServer returns an in-process httptest server that answers like an
// OpenBao/Vault server, and asserts that the auth token and namespace headers
// arrive as configured. It is the deterministic, socket-local backend the
// integration test drives the default net/http transport against.
func newMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("X-Vault-Token"); got != "s.roottoken" {
			t.Errorf("X-Vault-Token = %q, want s.roottoken", got)
		}
		if got := r.Header.Get("X-Vault-Namespace"); got != "team-a" {
			t.Errorf("X-Vault-Namespace = %q, want team-a", got)
		}
		path := r.URL.Path
		switch {
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && path == "/v1/sys/health":
			writeJSON(w, flatJSON)
		case r.Method == http.MethodGet && path == "/v1/sys/seal-status":
			writeJSON(w, flatJSON)
		case r.Method == http.MethodGet && path == "/v1/sys/mounts":
			writeJSON(w, flatJSON)
		case strings.Contains(path, "/login"):
			writeJSON(w, authJSON)
		default:
			writeJSON(w, secretJSON)
		}
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

// writeJSON writes body as a 200 JSON response.
func writeJSON(w http.ResponseWriter, body string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

// newIntegrationClient wires a Client to the mock server using the default
// net/http transport (no injected Doer), so the integration test exercises the
// real transport against a local, deterministic backend.
func newIntegrationClient(t *testing.T) *Client {
	srv := newMockServer(t)
	return NewClient(Config{
		Address:   srv.URL,
		Token:     "s.roottoken",
		Namespace: "team-a",
	})
}
