// Copyright (c) the go-ruby-openbao/openbao authors
//
// SPDX-License-Identifier: BSD-3-Clause

package openbao

import "testing"

// TestLogicalIntegration drives the logical verbs through the default net/http
// transport against the in-process mock server.
func TestLogicalIntegration(t *testing.T) {
	c := newIntegrationClient(t)
	l := c.Logical()

	s, err := l.Read("secret/data/foo")
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if s.Data["foo"] != "bar" {
		t.Errorf("Data[foo] = %v", s.Data["foo"])
	}
	if s.LeaseID != "lease-abc" || s.LeaseDuration != 3600 || !s.Renewable {
		t.Errorf("lease fields: %+v", s)
	}
	if len(s.Warnings) != 1 || s.RequestID != "req-123" {
		t.Errorf("warnings/request id: %+v", s)
	}

	if _, err := l.Write("secret/x", map[string]any{"k": "v"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := l.List("secret/"); err != nil {
		t.Fatalf("List: %v", err)
	}
	del, err := l.Delete("secret/x")
	if err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if del != nil {
		t.Errorf("Delete should return nil secret on 204, got %+v", del)
	}
}

// TestKVv1Integration drives the KV v1 helper.
func TestKVv1Integration(t *testing.T) {
	kv := newIntegrationClient(t).KVv1("secret")
	if _, err := kv.Write("app/config", map[string]any{"k": "v"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := kv.Read("app/config"); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if _, err := kv.List("app/"); err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, err := kv.Delete("app/config"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

// TestKVv2Integration drives every KV v2 verb.
func TestKVv2Integration(t *testing.T) {
	kv := newIntegrationClient(t).KVv2("secret")
	if _, err := kv.Write("app", map[string]any{"k": "v"}); err != nil {
		t.Fatalf("Write: %v", err)
	}
	if _, err := kv.Read("app"); err != nil {
		t.Fatalf("Read: %v", err)
	}
	if _, err := kv.ReadVersion("app", 2); err != nil {
		t.Fatalf("ReadVersion: %v", err)
	}
	if _, err := kv.List("app"); err != nil {
		t.Fatalf("List: %v", err)
	}
	if _, err := kv.DeleteVersions("app", 1, 2); err != nil {
		t.Fatalf("DeleteVersions: %v", err)
	}
	if _, err := kv.Undelete("app", 1); err != nil {
		t.Fatalf("Undelete: %v", err)
	}
	if _, err := kv.Destroy("app", 1); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	if _, err := kv.ReadMetadata("app"); err != nil {
		t.Fatalf("ReadMetadata: %v", err)
	}
	if _, err := kv.WriteMetadata("app", map[string]any{"max_versions": 5}); err != nil {
		t.Fatalf("WriteMetadata: %v", err)
	}
	if _, err := kv.Delete("app"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := kv.DeleteMetadata("app"); err != nil {
		t.Fatalf("DeleteMetadata: %v", err)
	}
}

// TestTransitIntegration drives every transit operation.
func TestTransitIntegration(t *testing.T) {
	tr := newIntegrationClient(t).Transit("transit")
	if _, err := tr.Encrypt("k", []byte("hello")); err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	if _, err := tr.Encrypt("k", []byte("hello"), []byte("ctx")); err != nil {
		t.Fatalf("Encrypt+ctx: %v", err)
	}
	if _, err := tr.Decrypt("k", "vault:v1:abcd"); err != nil {
		t.Fatalf("Decrypt: %v", err)
	}
	if _, err := tr.Decrypt("k", "vault:v1:abcd", []byte("ctx")); err != nil {
		t.Fatalf("Decrypt+ctx: %v", err)
	}
	if _, err := tr.Rewrap("k", "vault:v1:abcd"); err != nil {
		t.Fatalf("Rewrap: %v", err)
	}
	if _, err := tr.Rewrap("k", "vault:v1:abcd", []byte("ctx")); err != nil {
		t.Fatalf("Rewrap+ctx: %v", err)
	}
	if _, err := tr.Sign("k", []byte("data")); err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if _, err := tr.Verify("k", []byte("data"), "vault:v1:sig"); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if _, err := tr.GenerateDataKey("k", "plaintext"); err != nil {
		t.Fatalf("GenerateDataKey: %v", err)
	}
}

// TestSysIntegration drives every sys endpoint.
func TestSysIntegration(t *testing.T) {
	sys := newIntegrationClient(t).Sys()
	h, err := sys.Health()
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if h["initialized"] != true {
		t.Errorf("health: %+v", h)
	}
	if _, err := sys.SealStatus(); err != nil {
		t.Fatalf("SealStatus: %v", err)
	}
	if _, err := sys.Mounts(); err != nil {
		t.Fatalf("Mounts: %v", err)
	}
	if err := sys.EnableMount("kv", "kv-v2", map[string]any{"description": "d"}); err != nil {
		t.Fatalf("EnableMount: %v", err)
	}
	if err := sys.DisableMount("kv"); err != nil {
		t.Fatalf("DisableMount: %v", err)
	}
	if _, err := sys.Policies(); err != nil {
		t.Fatalf("Policies: %v", err)
	}
	if _, err := sys.Policy("default"); err != nil {
		t.Fatalf("Policy: %v", err)
	}
	if err := sys.PutPolicy("web", "path \"secret/*\" { capabilities = [\"read\"] }"); err != nil {
		t.Fatalf("PutPolicy: %v", err)
	}
	if err := sys.DeletePolicy("web"); err != nil {
		t.Fatalf("DeletePolicy: %v", err)
	}
	if _, err := sys.RenewLease("lease-abc", 60); err != nil {
		t.Fatalf("RenewLease: %v", err)
	}
	if err := sys.RevokeLease("lease-abc"); err != nil {
		t.Fatalf("RevokeLease: %v", err)
	}
	if _, err := sys.LookupLease("lease-abc"); err != nil {
		t.Fatalf("LookupLease: %v", err)
	}
}

// TestAuthIntegration drives token self-management and the login flows, then
// adopts the returned token.
func TestAuthIntegration(t *testing.T) {
	c := newIntegrationClient(t)
	auth := c.Auth()

	if _, err := auth.Token().LookupSelf(); err != nil {
		t.Fatalf("LookupSelf: %v", err)
	}
	if _, err := auth.Token().RenewSelf(60); err != nil {
		t.Fatalf("RenewSelf: %v", err)
	}
	if err := auth.Token().RevokeSelf(); err != nil {
		t.Fatalf("RevokeSelf: %v", err)
	}

	login, err := auth.AppRole("approle").Login("role-id", "secret-id")
	if err != nil {
		t.Fatalf("AppRole.Login: %v", err)
	}
	if login.Auth.ClientToken != "s.newtoken" {
		t.Errorf("client_token = %q", login.Auth.ClientToken)
	}
	if login.TokenID() != "s.newtoken" {
		t.Errorf("TokenID = %q", login.TokenID())
	}

	up, err := auth.Userpass("userpass").Login("bob", "pw")
	if err != nil {
		t.Fatalf("Userpass.Login: %v", err)
	}

	// The client can adopt the login's client token.
	if c.AdoptToken(up).Token() != "s.newtoken" {
		t.Errorf("AdoptToken did not adopt the login token")
	}
}
