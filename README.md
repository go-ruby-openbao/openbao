<p align="center"><img src="https://go-ruby-openbao.github.io/logo.png" alt="go-ruby-openbao/openbao" width="720"></p>

# openbao — go-ruby-openbao

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-openbao.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) reimplementation of the deterministic core of Ruby's
[`vault`](https://github.com/hashicorp/vault-ruby) gem** — the official OpenBao /
HashiCorp Vault client (`Vault::Client` / `OpenBao::Client`). It reproduces the
client configuration, the Logical / KV / Transit / Sys / Auth endpoint groups,
the `Secret` response model, and the HTTP-status→error mapping the gem performs
around a transport — **without any Ruby runtime**.

It is the OpenBao/Vault client for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but a **standalone,
reusable** module — a sibling of
[go-ruby-faraday](https://github.com/go-ruby-faraday/faraday) and
[go-ruby-net-http](https://github.com/go-ruby-net-http/net-http).

> **What it is — and isn't.** Everything the `vault` gem does *around* the wire is
> deterministic and needs **no interpreter**, so it lives here as pure Go:
> resolving the address / token / namespace from config or the environment,
> building the `/v1/` request path, JSON-encoding request bodies, decoding the
> `Secret` envelope (`data` / `lease_id` / `renewable` / `warnings` / `auth`), and
> mapping a non-2xx status onto the error tree. The **HTTP round-trip itself is a
> host seam**: the terminal `Doer` performs the transport. The default production
> `Doer` is backed by `net/http`; **tests inject a `DoerFunc` stub — or drive the
> default `Doer` against an in-process `httptest` mock OpenBao — and the core
> opens no socket of its own.** A future rbgo binding wires the seam to Ruby's
> `Net::HTTP`.

## Features

Faithful port of the `vault` gem's client core:

- **Client** — `NewClient(Config{Address, Token, Namespace, Timeout, Doer})` with
  environment fallback (`VAULT_ADDR`/`BAO_ADDR`, `VAULT_TOKEN`/`BAO_TOKEN`,
  `VAULT_NAMESPACE`/`BAO_NAMESPACE`), `Token()`/`SetToken`,
  `Namespace()`/`SetNamespace`, and `AdoptToken(secret)` to adopt a login's
  `client_token`.
- **Logical** — `Read`/`Write`/`List`/`Delete` on an arbitrary path; the
  `Secret` carries `Data`/`LeaseID`/`LeaseDuration`/`Renewable`/`Warnings`. A 404
  read/list yields a nil secret, mirroring the gem.
- **KV v1 & v2** — `KVv1(mount)` (flat) and `KVv2(mount)` with the version-2
  `data/`/`metadata/`/`delete/`/`undelete/`/`destroy/` path rewriting and the
  versioned request bodies: `Read`/`ReadVersion`/`Write`/`List`/`Delete`/
  `DeleteVersions`/`Undelete`/`Destroy`/`ReadMetadata`/`WriteMetadata`/
  `DeleteMetadata`.
- **Transit** — `Transit(mount)`: `Encrypt`/`Decrypt`/`Rewrap`/`Sign`/`Verify`/
  `GenerateDataKey` (base64 handling done for you).
- **Sys** — `Health`/`SealStatus`/`Mounts`/`EnableMount`/`DisableMount`/
  `Policies`/`Policy`/`PutPolicy`/`DeletePolicy`/`RenewLease`/`RevokeLease`/
  `LookupLease`.
- **Auth** — `Auth().Token()` (`LookupSelf`/`RenewSelf`/`RevokeSelf`),
  `Auth().AppRole(mount).Login(roleID, secretID)` and
  `Auth().Userpass(mount).Login(username, password)` — each returns a `Secret`
  whose `Auth.ClientToken` the client can adopt.
- **Transport seam** — `Doer`/`DoerFunc`; the default is `net/http`, **the core
  never opens a socket itself**.
- **Error tree** — a single `VaultError` type whose `Kind` places it in the gem's
  hierarchy (`Vault::VaultError` → `HTTPError` →
  `HTTPClientError`/`HTTPServerError`, plus `HTTPConnectionError`), matched with
  `errors.Is` against the `Err*` sentinels (a superclass matches its subclasses)
  and carrying the HTTP status and the API `errors` array.

CGO-free, dependency-free (stdlib only), **100% test coverage**, `gofmt` +
`go vet` clean, and green across the six 64-bit Go targets (amd64, arm64,
riscv64, loong64, ppc64le, **s390x** — big-endian) plus `js/wasm` and
`wasip1/wasm`.

## Install

```sh
go get github.com/go-ruby-openbao/openbao
```

## Usage

```go
package main

import (
	"fmt"

	"github.com/go-ruby-openbao/openbao"
)

func main() {
	client := openbao.NewClient(openbao.Config{
		Address: "https://127.0.0.1:8200",
		Token:   "s.sometoken",
	})

	// Arbitrary logical path.
	secret, err := client.Logical().Read("secret/data/foo")
	if err != nil {
		// a *openbao.VaultError: errors.Is(err, openbao.ErrHTTPClientError), etc.
		return
	}
	fmt.Println(secret.Data)

	// KV v2 (rewrites secret/foo -> secret/data/foo, wraps the write body).
	_, _ = client.KVv2("secret").Write("foo", map[string]any{"pw": "s3cr3t"})

	// Transit encryption.
	enc, _ := client.Transit("transit").Encrypt("mykey", []byte("hello"))
	fmt.Println(enc.Data["ciphertext"])

	// AppRole login, then adopt the returned client token.
	login, _ := client.Auth().AppRole("approle").Login("role-id", "secret-id")
	client.AdoptToken(login)
}
```

### Injecting a transport (tests / hosts)

```go
client := openbao.NewClient(openbao.Config{
	Address: "https://vault.example",
	Token:   "t",
	Doer: openbao.DoerFunc(func(req *openbao.Request) (*openbao.Response, error) {
		return &openbao.Response{
			StatusCode: 200,
			Body:       []byte(`{"data":{"foo":"bar"}}`),
		}, nil
	}),
})
secret, _ := client.Logical().Read("secret/data/foo")
// secret.Data["foo"] == "bar"
```

## Value model

| gem                                          | this package                                    |
| -------------------------------------------- | ----------------------------------------------- |
| `Vault::Client.new(address:, token:, …)`     | `NewClient(Config{Address, Token, Namespace})`  |
| `client.logical.read/write/list/delete`      | `client.Logical().Read/Write/List/Delete`       |
| `client.kv(mount).read/write/…`              | `client.KVv2(mount).Read/Write/…`               |
| `client.transit.encrypt/decrypt/…`           | `client.Transit(mount).Encrypt/Decrypt/…`       |
| `client.sys.health / mounts / policies`      | `client.Sys().Health/Mounts/Policies`           |
| `client.auth.approle.login(id, secret)`      | `client.Auth().AppRole(m).Login(id, secret)`    |
| `Vault::Secret#data/#lease_id/#auth`         | `(*Secret).Data / .LeaseID / .Auth`             |
| `Vault::HTTPError` subtree                    | `*VaultError` + `Err*` sentinels (`errors.Is`)  |
| the Persistent HTTP connection               | `Doer` (host seam; `DoerFunc` in tests)         |

## Tests & coverage

The suite is deterministic and socket-local: an in-process `httptest` mock
OpenBao answers the default `net/http` transport for the happy-path integration
tests, while `DoerFunc` stubs and a fake `net/http` client drive every error
branch (transport failure, bad JSON, non-2xx status class, 404-to-nil). **No test
opens a real socket**, so the cross-arch qemu lanes and the Windows lane all hold
coverage at **100%**.

```sh
COVERPKG=$(go list ./... | paste -sd, -)
go test -race -coverpkg="$COVERPKG" -coverprofile=cover.out ./...
go tool cover -func=cover.out | tail -1   # 100.0%
```

## WebAssembly

Being pure Go (CGO=0), this library also compiles to **WebAssembly** — both
`GOOS=js GOARCH=wasm` (browser / Node.js) and `GOOS=wasip1 GOARCH=wasm` (WASI).
CI builds both targets on every push, alongside the six 64-bit native/qemu arches.

```sh
GOOS=js     GOARCH=wasm go build ./...   # browser / Node
GOOS=wasip1 GOARCH=wasm go build ./...   # WASI (wasmtime, wasmer, wasmedge, …)
```

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright the go-ruby-openbao/openbao authors.
