// Copyright (c) the go-ruby-openbao/openbao authors
//
// SPDX-License-Identifier: BSD-3-Clause

package openbao

// Secret is the decoded OpenBao/Vault response envelope (Vault::Secret). Reads
// and writes that return data populate Data; leased secrets populate
// LeaseID/LeaseDuration/Renewable; login and token responses populate Auth. Any
// server-side warnings are surfaced in Warnings.
type Secret struct {
	// RequestID is the server-assigned request identifier.
	RequestID string `json:"request_id"`
	// LeaseID is the lease identifier for a leased secret ("" when not leased).
	LeaseID string `json:"lease_id"`
	// LeaseDuration is the lease TTL in seconds.
	LeaseDuration int `json:"lease_duration"`
	// Renewable reports whether the lease can be renewed.
	Renewable bool `json:"renewable"`
	// Data is the secret payload (Vault::Secret#data).
	Data map[string]any `json:"data"`
	// Warnings carries any server-side warnings (Vault::Secret#warnings).
	Warnings []string `json:"warnings"`
	// Auth carries the authentication response for a login/token call
	// (Vault::Secret#auth); nil otherwise.
	Auth *SecretAuth `json:"auth"`
	// WrapInfo carries response-wrapping metadata when the call was wrapped; nil
	// otherwise (Vault::Secret#wrap_info).
	WrapInfo map[string]any `json:"wrap_info"`
}

// SecretAuth is the authentication half of a [Secret] (Vault::Secret#auth),
// populated by login and token endpoints.
type SecretAuth struct {
	// ClientToken is the issued Vault token (auth.client_token).
	ClientToken string `json:"client_token"`
	// Accessor is the token accessor.
	Accessor string `json:"accessor"`
	// Policies are the token's policies.
	Policies []string `json:"policies"`
	// TokenPolicies are the token's token-scoped policies.
	TokenPolicies []string `json:"token_policies"`
	// Metadata is the token metadata map.
	Metadata map[string]any `json:"metadata"`
	// LeaseDuration is the token TTL in seconds.
	LeaseDuration int `json:"lease_duration"`
	// Renewable reports whether the token can be renewed.
	Renewable bool `json:"renewable"`
}

// TokenID returns the auth client token carried by the secret, or "" when the
// secret has no auth block. It is the convenient accessor a host uses to adopt a
// login's token.
func (s *Secret) TokenID() string {
	if s == nil || s.Auth == nil {
		return ""
	}
	return s.Auth.ClientToken
}
