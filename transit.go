// Copyright (c) the go-ruby-openbao/openbao authors
//
// SPDX-License-Identifier: BSD-3-Clause

package openbao

import "encoding/base64"

// Transit is a transit secrets-engine helper rooted at a mount (default
// "transit"): encryption, decryption, rewrapping, signing, verification and
// data-key generation as cryptographic operations performed by the server
// without exposing the key.
type Transit struct {
	c     *Client
	mount string
}

// op builds "<mount>/<operation>/<key>".
func (t *Transit) op(operation, key string) string {
	return t.mount + "/" + operation + "/" + key
}

// Encrypt encrypts plaintext with the named key (POST <mount>/encrypt/<key>).
// The plaintext is base64-encoded as the engine requires; an optional context
// (for a keyed/derived key) is base64-encoded too. The returned secret's Data
// carries "ciphertext".
func (t *Transit) Encrypt(key string, plaintext []byte, context ...[]byte) (*Secret, error) {
	body := map[string]any{"plaintext": base64.StdEncoding.EncodeToString(plaintext)}
	if len(context) > 0 {
		body["context"] = base64.StdEncoding.EncodeToString(context[0])
	}
	return t.c.do("POST", t.op("encrypt", key), body)
}

// Decrypt decrypts ciphertext with the named key (POST <mount>/decrypt/<key>).
// The returned secret's Data carries "plaintext" (base64-encoded by the engine).
func (t *Transit) Decrypt(key, ciphertext string, context ...[]byte) (*Secret, error) {
	body := map[string]any{"ciphertext": ciphertext}
	if len(context) > 0 {
		body["context"] = base64.StdEncoding.EncodeToString(context[0])
	}
	return t.c.do("POST", t.op("decrypt", key), body)
}

// Rewrap rewraps ciphertext to the key's latest version
// (POST <mount>/rewrap/<key>).
func (t *Transit) Rewrap(key, ciphertext string, context ...[]byte) (*Secret, error) {
	body := map[string]any{"ciphertext": ciphertext}
	if len(context) > 0 {
		body["context"] = base64.StdEncoding.EncodeToString(context[0])
	}
	return t.c.do("POST", t.op("rewrap", key), body)
}

// Sign signs input with the named key (POST <mount>/sign/<key>); input is
// base64-encoded. The returned secret's Data carries "signature".
func (t *Transit) Sign(key string, input []byte) (*Secret, error) {
	body := map[string]any{"input": base64.StdEncoding.EncodeToString(input)}
	return t.c.do("POST", t.op("sign", key), body)
}

// Verify verifies signature over input with the named key
// (POST <mount>/verify/<key>). The returned secret's Data carries "valid".
func (t *Transit) Verify(key string, input []byte, signature string) (*Secret, error) {
	body := map[string]any{
		"input":     base64.StdEncoding.EncodeToString(input),
		"signature": signature,
	}
	return t.c.do("POST", t.op("verify", key), body)
}

// GenerateDataKey generates a new high-entropy data key wrapped (and optionally
// returned in plaintext) by the named key
// (POST <mount>/datakey/<keyType>/<key>). keyType is "plaintext" or "wrapped".
func (t *Transit) GenerateDataKey(key, keyType string) (*Secret, error) {
	return t.c.do("POST", t.mount+"/datakey/"+keyType+"/"+key, map[string]any{})
}
