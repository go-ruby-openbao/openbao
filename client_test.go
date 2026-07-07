// Copyright (c) the go-ruby-openbao/openbao authors
//
// SPDX-License-Identifier: BSD-3-Clause

package openbao

import (
	"errors"
	"testing"
)

func TestConfigResolution(t *testing.T) {
	// Explicit config wins.
	c := NewClient(Config{Address: "https://explicit:8200/", Token: "tok", Namespace: "ns"})
	if c.Address() != "https://explicit:8200" { // trailing slash trimmed
		t.Errorf("Address = %q", c.Address())
	}
	if c.Token() != "tok" || c.Namespace() != "ns" {
		t.Errorf("token/ns = %q/%q", c.Token(), c.Namespace())
	}
	if c.doer == nil {
		t.Error("default doer should be non-nil")
	}

	// Environment fallback (VAULT_* then BAO_*).
	t.Setenv("VAULT_ADDR", "https://env-vault:8200")
	t.Setenv("VAULT_TOKEN", "env-tok")
	t.Setenv("VAULT_NAMESPACE", "env-ns")
	c = NewClient(Config{})
	if c.Address() != "https://env-vault:8200" || c.Token() != "env-tok" || c.Namespace() != "env-ns" {
		t.Errorf("env resolution: %q %q %q", c.Address(), c.Token(), c.Namespace())
	}

	// BAO_* fallback when VAULT_* is empty.
	t.Setenv("VAULT_ADDR", "")
	t.Setenv("VAULT_TOKEN", "")
	t.Setenv("VAULT_NAMESPACE", "")
	t.Setenv("BAO_ADDR", "https://bao:8200")
	t.Setenv("BAO_TOKEN", "bao-tok")
	t.Setenv("BAO_NAMESPACE", "bao-ns")
	c = NewClient(Config{})
	if c.Address() != "https://bao:8200" || c.Token() != "bao-tok" || c.Namespace() != "bao-ns" {
		t.Errorf("BAO_* resolution: %q %q %q", c.Address(), c.Token(), c.Namespace())
	}

	// Default address and empty token/namespace when nothing is set.
	t.Setenv("BAO_ADDR", "")
	t.Setenv("BAO_TOKEN", "")
	t.Setenv("BAO_NAMESPACE", "")
	c = NewClient(Config{})
	if c.Address() != DefaultAddress || c.Token() != "" || c.Namespace() != "" {
		t.Errorf("default resolution: %q %q %q", c.Address(), c.Token(), c.Namespace())
	}
}

func TestSettersAndMount(t *testing.T) {
	c := NewClient(Config{Address: "http://x"})
	if c.SetToken("t2").Token() != "t2" {
		t.Error("SetToken")
	}
	if c.SetNamespace("n2").Namespace() != "n2" {
		t.Error("SetNamespace")
	}

	if got := mountOr("", "secret"); got != "secret" {
		t.Errorf("mountOr default = %q", got)
	}
	if got := mountOr("/kv/", "secret"); got != "kv" {
		t.Errorf("mountOr trim = %q", got)
	}
	// Constructors pick the right default mounts.
	if c.KVv1("").mount != "secret" || c.Transit("").mount != "transit" {
		t.Error("default mounts")
	}
	if c.Auth().AppRole("").mount != "approle" || c.Auth().Userpass("").mount != "userpass" {
		t.Error("default auth mounts")
	}
}

func TestAdoptTokenAndTokenID(t *testing.T) {
	c := NewClient(Config{Address: "http://x", Token: "orig"})

	// nil secret and no-auth secret leave the token unchanged.
	if c.AdoptToken(nil).Token() != "orig" {
		t.Error("AdoptToken(nil) changed token")
	}
	if c.AdoptToken(&Secret{}).Token() != "orig" {
		t.Error("AdoptToken(no-auth) changed token")
	}
	// a secret with an auth token is adopted.
	s := &Secret{Auth: &SecretAuth{ClientToken: "adopted"}}
	if c.AdoptToken(s).Token() != "adopted" {
		t.Error("AdoptToken(auth) did not adopt")
	}

	// TokenID on nil / no-auth / auth.
	var nilSecret *Secret
	if nilSecret.TokenID() != "" {
		t.Error("nil TokenID")
	}
	if (&Secret{}).TokenID() != "" {
		t.Error("no-auth TokenID")
	}
	if s.TokenID() != "adopted" {
		t.Error("auth TokenID")
	}
}

func TestErrorTree(t *testing.T) {
	// A client error is-a HTTPError is-a VaultError.
	clientErr := newResponseError(403, []string{"denied"})
	if !errors.Is(clientErr, ErrHTTPClientError) ||
		!errors.Is(clientErr, ErrHTTPError) ||
		!errors.Is(clientErr, ErrVaultError) {
		t.Error("client error hierarchy")
	}
	if errors.Is(clientErr, ErrHTTPServerError) {
		t.Error("client error should not match server error")
	}

	// A connection error is-a HTTPError but not a client error.
	connErr := newConnectionError(errors.New("refused"))
	if !errors.Is(connErr, ErrHTTPError) || errors.Is(connErr, ErrHTTPClientError) {
		t.Error("connection error hierarchy")
	}

	// MissingRequired descends from VaultError only.
	if !errors.Is(ErrMissingRequired, ErrVaultError) || errors.Is(ErrMissingRequired, ErrHTTPError) {
		t.Error("MissingRequired hierarchy")
	}

	// Is against a non-VaultError target is false.
	if clientErr.Is(errors.New("other")) {
		t.Error("Is(non-VaultError) should be false")
	}
	// The predicate shims reject non-VaultError values.
	if IsHTTPError(errors.New("plain")) || IsNotFound(errors.New("plain")) {
		t.Error("predicates should reject non-VaultError")
	}
	// Root message and Error().
	if ErrVaultError.Error() != string(KindVaultError) {
		t.Error("Error() message")
	}
}
