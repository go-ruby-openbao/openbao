// Copyright (c) the go-ruby-openbao/openbao authors
//
// SPDX-License-Identifier: BSD-3-Clause

package openbao

import (
	"errors"
	"io"
	"net/http"
	"testing"
)

// stubClient wires a Client to an injected Doer, for the fault-injection tests.
func stubClient(doer Doer) *Client {
	return NewClient(Config{Address: "http://vault.test", Token: "t", Doer: doer})
}

// respFunc builds a DoerFunc returning a fixed status and body.
func respFunc(status int, body string) DoerFunc {
	return func(*Request) (*Response, error) {
		return &Response{StatusCode: status, Body: []byte(body)}, nil
	}
}

func TestConnectionError(t *testing.T) {
	sentinel := errors.New("dial tcp: connection refused")
	c := stubClient(DoerFunc(func(*Request) (*Response, error) { return nil, sentinel }))

	_, err := c.Logical().Read("secret/x")
	if !IsHTTPError(err) {
		t.Fatalf("want HTTPError, got %v", err)
	}
	var ve *VaultError
	if !errors.As(err, &ve) || ve.Kind != KindHTTPConnectionError {
		t.Fatalf("want connection error, got %+v", err)
	}
	if !errors.Is(err, sentinel) {
		t.Errorf("Unwrap should expose the transport cause")
	}
}

func TestMarshalError(t *testing.T) {
	c := stubClient(respFunc(200, secretJSON))
	_, err := c.Logical().Write("secret/x", map[string]any{"bad": make(chan int)})
	if err == nil || !errors.Is(err, ErrVaultError) {
		t.Fatalf("want VaultError from marshal failure, got %v", err)
	}
}

func TestDecodeErrors(t *testing.T) {
	c := stubClient(respFunc(200, "not json"))
	if _, err := c.Logical().Write("secret/x", map[string]any{"k": "v"}); err == nil {
		t.Fatal("want decode error from do")
	}
	// doRaw decode failure and empty-body / request-error paths.
	if _, err := c.doRaw("GET", "sys/health", nil); err == nil {
		t.Fatal("want decode error from doRaw")
	}
	empty := stubClient(respFunc(200, ""))
	if m, err := empty.doRaw("GET", "sys/health", nil); err != nil || m != nil {
		t.Fatalf("empty body doRaw = %v, %v", m, err)
	}
	boom := stubClient(DoerFunc(func(*Request) (*Response, error) {
		return nil, errors.New("boom")
	}))
	if _, err := boom.doRaw("GET", "sys/health", nil); err == nil {
		t.Fatal("want request error from doRaw")
	}
}

func TestResponseErrorClassification(t *testing.T) {
	// 404 with an errors array → client error, not-found.
	c := stubClient(respFunc(404, `{"errors":["no such path","try again"]}`))
	_, err := c.Sys().Policy("missing")
	if !IsHTTPClientError(err) || !IsNotFound(err) {
		t.Fatalf("want client/not-found, got %v", err)
	}
	var ve *VaultError
	errors.As(err, &ve)
	if len(ve.Errors) != 2 || ve.StatusCode != 404 {
		t.Errorf("errors/status not carried: %+v", ve)
	}

	// 500 with an empty body → server error, no errors array.
	c = stubClient(respFunc(500, ""))
	_, err = c.Sys().Policy("x")
	if !IsHTTPServerError(err) {
		t.Fatalf("want server error, got %v", err)
	}
	errors.As(err, &ve)
	if len(ve.Errors) != 0 {
		t.Errorf("errors should be empty: %+v", ve)
	}

	// 400 with a non-JSON body → parseErrors swallows the decode error.
	c = stubClient(respFunc(400, "<html>bad</html>"))
	_, err = c.Sys().Policy("x")
	if !IsHTTPClientError(err) {
		t.Fatalf("want client error, got %v", err)
	}
}

func TestReadOrNil(t *testing.T) {
	// 404 → nil secret, nil error.
	c := stubClient(respFunc(404, `{"errors":["missing"]}`))
	s, err := c.Logical().Read("secret/gone")
	if s != nil || err != nil {
		t.Fatalf("404 read = %v, %v; want nil, nil", s, err)
	}
	// non-404 error → surfaced.
	c = stubClient(respFunc(500, ""))
	if _, err := c.Logical().List("secret/"); err == nil {
		t.Fatal("want error from 500 list")
	}
}

// fakeHTTP is a stub net/http client for the transport error branches.
type fakeHTTP struct {
	resp *http.Response
	err  error
}

func (f *fakeHTTP) Do(*http.Request) (*http.Response, error) { return f.resp, f.err }

// errBody is a response body that fails on Read, to exercise the ReadAll branch.
type errBody struct{}

func (errBody) Read([]byte) (int, error) { return 0, errors.New("read failed") }
func (errBody) Close() error             { return nil }

func TestNetHTTPDoer(t *testing.T) {
	// http.NewRequest failure (invalid method).
	d := &netHTTPDoer{client: &fakeHTTP{}}
	if _, err := d.Do(&Request{Method: "GET\n", URL: "http://x"}); err == nil {
		t.Fatal("want NewRequest error")
	}

	// transport failure from the inner client.
	d = &netHTTPDoer{client: &fakeHTTP{err: errors.New("dial failed")}}
	if _, err := d.Do(&Request{Method: "GET", URL: "http://x"}); err == nil {
		t.Fatal("want client.Do error")
	}

	// response-body read failure.
	d = &netHTTPDoer{client: &fakeHTTP{resp: &http.Response{
		StatusCode: 200,
		Body:       errBody{},
		Header:     http.Header{},
	}}}
	if _, err := d.Do(&Request{Method: "POST", URL: "http://x", Body: []byte(`{}`)}); err == nil {
		t.Fatal("want ReadAll error")
	}

	// success path with a real body and header (fills the Response fields).
	d = &netHTTPDoer{client: &fakeHTTP{resp: &http.Response{
		StatusCode: 201,
		Body:       io.NopCloser(stringReader("ok")),
		Header:     http.Header{"X-Test": {"1"}},
	}}}
	resp, err := d.Do(&Request{Method: "GET", URL: "http://x", Headers: map[string]string{"A": "b"}})
	if err != nil || resp.StatusCode != 201 || string(resp.Body) != "ok" {
		t.Fatalf("success path: %+v, %v", resp, err)
	}
}

// stringReader is a tiny io.Reader over a string (avoids importing strings here).
type stringReader string

func (s stringReader) Read(p []byte) (int, error) {
	n := copy(p, s)
	if n == len(s) {
		return n, io.EOF
	}
	return n, nil
}
