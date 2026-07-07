// Copyright (c) the go-ruby-openbao/openbao authors
//
// SPDX-License-Identifier: BSD-3-Clause

package openbao

import (
	"bytes"
	"io"
	"net/http"
	"time"
)

// Request is the prepared HTTP request the client hands to a [Doer]: the fully
// built URL (Address + "/v1/" + path), the method, the header set (including
// X-Vault-Token / X-Vault-Namespace when configured) and the JSON-encoded body
// (nil for a bodyless request). It is a transport-agnostic value so a host can
// wire it to any HTTP stack.
type Request struct {
	// Method is the upper-case HTTP method ("GET", "PUT", "POST", "DELETE",
	// "LIST").
	Method string
	// URL is the fully-built request URL.
	URL string
	// Headers are the outgoing headers.
	Headers map[string]string
	// Body is the JSON-encoded request body, or nil.
	Body []byte
}

// Response is the transport result a [Doer] returns: the HTTP status code, the
// raw response body, and the response headers. The client decodes the body into
// a [Secret] (2xx) or maps the status onto a [VaultError] (non-2xx).
type Response struct {
	// StatusCode is the HTTP status code.
	StatusCode int
	// Body is the raw response body.
	Body []byte
	// Header carries the response headers.
	Header map[string][]string
}

// Doer is the transport host seam: given a prepared [Request] it performs the
// HTTP round-trip and returns the [Response], or a transport error. The default
// production Doer ([defaultDoer]) is backed by net/http; the core opens no
// socket of its own, so tests inject a [DoerFunc] stub (or drive the default
// Doer against an in-process httptest server). A future rbgo binding wires this
// to Ruby's Net::HTTP.
type Doer interface {
	Do(req *Request) (*Response, error)
}

// DoerFunc adapts a function to the [Doer] interface — the convenient way to
// inject a stub transport in tests or a custom transport in a host.
type DoerFunc func(req *Request) (*Response, error)

// Do invokes f(req).
func (f DoerFunc) Do(req *Request) (*Response, error) { return f(req) }

// httpClient is the minimal net/http surface [netHTTPDoer] depends on,
// indirected so the request-building and response/error mapping can be exercised
// without opening a socket.
type httpClient interface {
	Do(req *http.Request) (*http.Response, error)
}

// netHTTPDoer is the default [Doer]: it turns a [Request] into a net/http
// request, executes it with its client, and maps the response (or a transport
// failure) back onto a [Response].
type netHTTPDoer struct {
	client httpClient
}

// defaultDoer returns the default net/http-backed [Doer] with the given overall
// timeout (0 means no timeout).
func defaultDoer(timeout time.Duration) Doer {
	return &netHTTPDoer{client: &http.Client{Timeout: timeout}}
}

// Do performs the HTTP round-trip for req with net/http.
func (d *netHTTPDoer) Do(req *Request) (*Response, error) {
	var body io.Reader
	if len(req.Body) > 0 {
		body = bytes.NewReader(req.Body)
	}
	hreq, err := http.NewRequest(req.Method, req.URL, body)
	if err != nil {
		return nil, err
	}
	for k, v := range req.Headers {
		hreq.Header.Set(k, v)
	}

	hresp, err := d.client.Do(hreq)
	if err != nil {
		return nil, err
	}
	defer hresp.Body.Close()

	raw, err := io.ReadAll(hresp.Body)
	if err != nil {
		return nil, err
	}
	return &Response{
		StatusCode: hresp.StatusCode,
		Body:       raw,
		Header:     hresp.Header,
	}, nil
}
