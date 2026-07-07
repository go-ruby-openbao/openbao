// Copyright (c) the go-ruby-openbao/openbao authors
//
// SPDX-License-Identifier: BSD-3-Clause

package openbao

// Logical is the logical-backend endpoint group (Vault::Client#logical): the
// generic read/write/list/delete verbs over an arbitrary API path, the primitive
// every higher-level helper (KV, transit, sys, auth) builds on.
type Logical struct {
	c *Client
}

// Read reads the secret at path (GET /v1/<path>), mirroring Vault::Logical#read:
// a 404 yields a nil secret and nil error, so a missing path is not an error.
func (l *Logical) Read(path string) (*Secret, error) {
	return l.readOrNil("GET", path, nil)
}

// Write writes data to path (PUT /v1/<path>, Vault::Logical#write) and returns
// the response secret (nil for a 204 No Content).
func (l *Logical) Write(path string, data map[string]any) (*Secret, error) {
	return l.c.do("PUT", path, data)
}

// List lists the keys at path (LIST /v1/<path>, Vault::Logical#list): a 404
// yields a nil secret and nil error.
func (l *Logical) List(path string) (*Secret, error) {
	return l.readOrNil("LIST", path, nil)
}

// Delete deletes the secret at path (DELETE /v1/<path>, Vault::Logical#delete).
func (l *Logical) Delete(path string) (*Secret, error) {
	return l.c.do("DELETE", path, nil)
}

// readOrNil performs a read-style call, translating a 404 into a nil secret.
func (l *Logical) readOrNil(method, path string, body any) (*Secret, error) {
	s, err := l.c.do(method, path, body)
	if err != nil {
		if IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}
