// Copyright (c) the go-ruby-openbao/openbao authors
//
// SPDX-License-Identifier: BSD-3-Clause

// Package openbao is a pure-Go (CGO-free) reimplementation of the deterministic
// core of Ruby's `vault` gem — the official OpenBao / HashiCorp Vault client
// (Vault::Client / OpenBao::Client). It reproduces the client configuration,
// the Logical/KV/Transit/Sys/Auth endpoint groups, the Secret response model,
// and the HTTP-status→error mapping the gem performs around a transport —
// without any Ruby runtime.
//
// It is the OpenBao/Vault client for
// [go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a
// standalone, reusable module — a sibling of go-ruby-faraday/faraday and
// go-ruby-net-http/net-http.
//
// # What it is — and isn't
//
// Everything the vault gem does around the wire is deterministic and needs no
// interpreter, so it lives here as pure Go: resolving the address/token/
// namespace from config or the environment, building the /v1/ request path,
// JSON-encoding request bodies, decoding the Secret response
// (data/lease_id/renewable/warnings/auth), and mapping a non-2xx status onto
// the error tree. The HTTP round-trip itself is a host seam: the terminal
// [Doer] performs the transport. The default production Doer is backed by
// net/http; tests inject a [DoerFunc] stub (or drive the default Doer against an
// in-process httptest server), and the core opens no socket of its own. This
// mirrors the gem, whose Persistent HTTP connection is the only piece that
// touches the network.
//
// # Flow
//
//	client := openbao.NewClient(openbao.Config{
//		Address: "https://127.0.0.1:8200",
//		Token:   "s.sometoken",
//	})
//
//	// arbitrary logical path
//	secret, err := client.Logical().Read("secret/data/foo")
//
//	// KV v2 helper (rewrites secret/foo -> secret/data/foo)
//	secret, err = client.KVv2("secret").Read("foo")
//
//	// transit encrypt
//	enc, err := client.Transit("transit").Encrypt("mykey", []byte("hello"))
//
//	// approle login, then adopt the returned client token
//	login, err := client.Auth().AppRole("approle").Login(roleID, secretID)
//	client.AdoptToken(login)
//
// # Value model
//
// A [Secret] carries the decoded response fields (Data, LeaseID, LeaseDuration,
// Renewable, Warnings, Auth). Errors are a single [VaultError] type whose Kind
// places it in the gem's error tree (VaultError → HTTPError →
// HTTPClientError/HTTPServerError, plus HTTPConnectionError), matched with
// errors.Is against the Err* sentinels so a superclass matches its subclasses.
// A host (go-embedded-ruby / rbgo) maps its Ruby Vault::Client / Secret /
// VaultError objects to and from these shapes.
package openbao
