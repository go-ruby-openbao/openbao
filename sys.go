// Copyright (c) the go-ruby-openbao/openbao authors
//
// SPDX-License-Identifier: BSD-3-Clause

package openbao

// Sys is the system-backend endpoint group (Vault::Client#sys): server health
// and seal status, secrets-engine mount management, ACL policy management and
// lease management.
type Sys struct {
	c *Client
}

// Health returns the server health (GET sys/health). Its payload is a flat
// object (initialized/sealed/standby/version/…), so it is returned as a raw map
// rather than a Secret envelope.
func (s *Sys) Health() (map[string]any, error) {
	return s.c.doRaw("GET", "sys/health", nil)
}

// SealStatus returns the server seal status (GET sys/seal-status) as a raw map
// (sealed/t/n/progress/version/…).
func (s *Sys) SealStatus() (map[string]any, error) {
	return s.c.doRaw("GET", "sys/seal-status", nil)
}

// Mounts lists the mounted secrets engines (GET sys/mounts) as a raw map keyed
// by mount path.
func (s *Sys) Mounts() (map[string]any, error) {
	return s.c.doRaw("GET", "sys/mounts", nil)
}

// EnableMount mounts a secrets engine of the given type at path
// (POST sys/mounts/<path>). Extra options (description, config, options) are
// merged into the request body.
func (s *Sys) EnableMount(path, engineType string, opts map[string]any) error {
	body := map[string]any{"type": engineType}
	for k, v := range opts {
		body[k] = v
	}
	_, err := s.c.do("POST", "sys/mounts/"+path, body)
	return err
}

// DisableMount unmounts the secrets engine at path (DELETE sys/mounts/<path>).
func (s *Sys) DisableMount(path string) error {
	_, err := s.c.do("DELETE", "sys/mounts/"+path, nil)
	return err
}

// Policies lists the ACL policy names (GET sys/policies/acl). The names are in
// the returned secret's Data under "keys"/"policies".
func (s *Sys) Policies() (*Secret, error) {
	return s.c.do("GET", "sys/policies/acl", nil)
}

// Policy reads the named ACL policy (GET sys/policies/acl/<name>).
func (s *Sys) Policy(name string) (*Secret, error) {
	return s.c.do("GET", "sys/policies/acl/"+name, nil)
}

// PutPolicy writes the named ACL policy from an HCL/JSON rules string
// (PUT sys/policies/acl/<name>).
func (s *Sys) PutPolicy(name, policy string) error {
	_, err := s.c.do("PUT", "sys/policies/acl/"+name, map[string]any{"policy": policy})
	return err
}

// DeletePolicy deletes the named ACL policy (DELETE sys/policies/acl/<name>).
func (s *Sys) DeletePolicy(name string) error {
	_, err := s.c.do("DELETE", "sys/policies/acl/"+name, nil)
	return err
}

// RenewLease renews a lease by id, requesting increment seconds of additional
// TTL (PUT sys/leases/renew).
func (s *Sys) RenewLease(leaseID string, increment int) (*Secret, error) {
	return s.c.do("PUT", "sys/leases/renew", map[string]any{
		"lease_id":  leaseID,
		"increment": increment,
	})
}

// RevokeLease revokes a lease by id (PUT sys/leases/revoke).
func (s *Sys) RevokeLease(leaseID string) error {
	_, err := s.c.do("PUT", "sys/leases/revoke", map[string]any{"lease_id": leaseID})
	return err
}

// LookupLease looks up a lease by id (PUT sys/leases/lookup).
func (s *Sys) LookupLease(leaseID string) (*Secret, error) {
	return s.c.do("PUT", "sys/leases/lookup", map[string]any{"lease_id": leaseID})
}
