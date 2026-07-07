// Copyright (c) the go-ruby-openbao/openbao authors
//
// SPDX-License-Identifier: BSD-3-Clause

package openbao

// Auth is the auth-method endpoint group (Vault::Client#auth): token
// self-management and the AppRole / Userpass login flows, each of which yields a
// [Secret] whose Auth.ClientToken the client can adopt with
// [Client.AdoptToken].
type Auth struct {
	c *Client
}

// Token returns the token self-management helper (auth/token).
func (a *Auth) Token() *TokenAuth { return &TokenAuth{c: a.c} }

// AppRole returns the AppRole login helper rooted at mount (default "approle").
func (a *Auth) AppRole(mount string) *AppRoleAuth {
	return &AppRoleAuth{c: a.c, mount: mountOr(mount, "approle")}
}

// Userpass returns the Userpass login helper rooted at mount (default
// "userpass").
func (a *Auth) Userpass(mount string) *UserpassAuth {
	return &UserpassAuth{c: a.c, mount: mountOr(mount, "userpass")}
}

// TokenAuth is the token self-management helper (auth/token).
type TokenAuth struct {
	c *Client
}

// LookupSelf looks up the current token (GET auth/token/lookup-self).
func (t *TokenAuth) LookupSelf() (*Secret, error) {
	return t.c.do("GET", "auth/token/lookup-self", nil)
}

// RenewSelf renews the current token, requesting increment seconds of additional
// TTL (POST auth/token/renew-self).
func (t *TokenAuth) RenewSelf(increment int) (*Secret, error) {
	return t.c.do("POST", "auth/token/renew-self", map[string]any{"increment": increment})
}

// RevokeSelf revokes the current token (POST auth/token/revoke-self).
func (t *TokenAuth) RevokeSelf() error {
	_, err := t.c.do("POST", "auth/token/revoke-self", nil)
	return err
}

// AppRoleAuth is the AppRole login helper rooted at a mount (default "approle").
type AppRoleAuth struct {
	c     *Client
	mount string
}

// Login exchanges a role_id/secret_id for a token
// (POST auth/<mount>/login). The returned secret's Auth.ClientToken is the
// issued token.
func (a *AppRoleAuth) Login(roleID, secretID string) (*Secret, error) {
	return a.c.do("POST", "auth/"+a.mount+"/login", map[string]any{
		"role_id":   roleID,
		"secret_id": secretID,
	})
}

// UserpassAuth is the Userpass login helper rooted at a mount (default
// "userpass").
type UserpassAuth struct {
	c     *Client
	mount string
}

// Login exchanges a username/password for a token
// (POST auth/<mount>/login/<username>). The returned secret's Auth.ClientToken
// is the issued token.
func (u *UserpassAuth) Login(username, password string) (*Secret, error) {
	return u.c.do("POST", "auth/"+u.mount+"/login/"+username, map[string]any{
		"password": password,
	})
}
