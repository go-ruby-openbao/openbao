// Copyright (c) the go-ruby-openbao/openbao authors
//
// SPDX-License-Identifier: BSD-3-Clause

package openbao

import (
	"fmt"
	"strings"
)

// VaultError is the root of the vault gem's error tree (Vault::VaultError).
// Every error carries a human message; errors raised from an HTTP response also
// carry the response StatusCode and the API "errors" array so callers can
// inspect what the server reported. Transport failures carry the underlying
// error as [VaultError.Cause].
//
// The concrete kinds are distinguished by [VaultError.Kind]; the predicate
// helpers ([IsHTTPError], [IsHTTPClientError], …) and the sentinel values
// ([ErrHTTPError], …) let callers match with errors.Is, mirroring Ruby's rescue
// of a Vault::HTTPError subclass.
type VaultError struct {
	// Kind names the specific gem error subclass (see the Err* sentinels).
	Kind ErrorKind
	// Message is the error text (VaultError#message).
	Message string
	// StatusCode is the HTTP status for an HTTPError (0 for a connection error).
	StatusCode int
	// Errors is the API "errors" array returned in the response body, if any.
	Errors []string
	// Cause is the underlying transport error for an HTTPConnectionError.
	Cause error
}

// ErrorKind identifies a vault-gem error subclass.
type ErrorKind string

// The vault-gem error subclasses, named as in the gem.
const (
	KindVaultError          ErrorKind = "Vault::VaultError"
	KindHTTPError           ErrorKind = "Vault::HTTPError"
	KindHTTPConnectionError ErrorKind = "Vault::HTTPConnectionError"
	KindHTTPClientError     ErrorKind = "Vault::HTTPClientError"
	KindHTTPServerError     ErrorKind = "Vault::HTTPServerError"
	KindMissingRequired     ErrorKind = "Vault::MissingRequiredStateError"
)

// Sentinel errors for errors.Is matching. Each names an error kind; a concrete
// [VaultError] with that Kind (or a subtree of it) matches via [VaultError.Is].
var (
	ErrVaultError          = &VaultError{Kind: KindVaultError, Message: string(KindVaultError)}
	ErrHTTPError           = &VaultError{Kind: KindHTTPError, Message: string(KindHTTPError)}
	ErrHTTPConnectionError = &VaultError{Kind: KindHTTPConnectionError, Message: string(KindHTTPConnectionError)}
	ErrHTTPClientError     = &VaultError{Kind: KindHTTPClientError, Message: string(KindHTTPClientError)}
	ErrHTTPServerError     = &VaultError{Kind: KindHTTPServerError, Message: string(KindHTTPServerError)}
	ErrMissingRequired     = &VaultError{Kind: KindMissingRequired, Message: string(KindMissingRequired)}
)

// errorParents maps each kind to its parent kind in the gem hierarchy. HTTPError
// and MissingRequiredStateError descend from VaultError; the connection and
// client/server errors descend from HTTPError. The root, VaultError, has no
// parent.
var errorParents = map[ErrorKind]ErrorKind{
	KindHTTPError:           KindVaultError,
	KindMissingRequired:     KindVaultError,
	KindHTTPConnectionError: KindHTTPError,
	KindHTTPClientError:     KindHTTPError,
	KindHTTPServerError:     KindHTTPError,
}

// Error implements the error interface (VaultError#message).
func (e *VaultError) Error() string { return e.Message }

// Unwrap exposes the underlying transport error for errors.Is/As on the cause.
func (e *VaultError) Unwrap() error { return e.Cause }

// Is reports whether e matches target: true when target is a [*VaultError] whose
// Kind is e's Kind or an ancestor of it, so errors.Is(err, ErrHTTPError) matches
// any client/server/connection error and errors.Is(err, ErrVaultError) matches
// every gem error — mirroring Ruby's rescue of a superclass.
func (e *VaultError) Is(target error) bool {
	t, ok := target.(*VaultError)
	if !ok {
		return false
	}
	for k := e.Kind; ; {
		if k == t.Kind {
			return true
		}
		parent, ok := errorParents[k]
		if !ok {
			return false
		}
		k = parent
	}
}

// httpErrorKind maps an HTTP status code to the matching gem error kind: 4xx →
// HTTPClientError, everything else (5xx and any other non-2xx) → HTTPServerError.
func httpErrorKind(status int) ErrorKind {
	if status >= 400 && status < 500 {
		return KindHTTPClientError
	}
	return KindHTTPServerError
}

// newResponseError builds a [VaultError] for a non-2xx response, classifying it
// by status and carrying the parsed API "errors" array.
func newResponseError(status int, errs []string) *VaultError {
	msg := fmt.Sprintf("the server responded with status %d", status)
	if len(errs) > 0 {
		msg += ": " + strings.Join(errs, ", ")
	}
	return &VaultError{
		Kind:       httpErrorKind(status),
		Message:    msg,
		StatusCode: status,
		Errors:     errs,
	}
}

// newConnectionError builds an HTTPConnectionError wrapping the transport cause.
func newConnectionError(cause error) *VaultError {
	return &VaultError{
		Kind:    KindHTTPConnectionError,
		Message: cause.Error(),
		Cause:   cause,
	}
}

// newError builds a plain VaultError (e.g. a body-encoding or body-decoding
// failure that is neither an HTTP status nor a transport error).
func newError(cause error) *VaultError {
	return &VaultError{Kind: KindVaultError, Message: cause.Error(), Cause: cause}
}

// IsHTTPError reports whether err is any gem HTTPError (client/server/connection).
func IsHTTPError(err error) bool { return isKind(err, ErrHTTPError) }

// IsHTTPClientError reports whether err is a gem 4xx HTTPClientError.
func IsHTTPClientError(err error) bool { return isKind(err, ErrHTTPClientError) }

// IsHTTPServerError reports whether err is a gem 5xx HTTPServerError.
func IsHTTPServerError(err error) bool { return isKind(err, ErrHTTPServerError) }

// IsNotFound reports whether err is an HTTPError with a 404 status.
func IsNotFound(err error) bool {
	e, ok := err.(*VaultError)
	return ok && e.StatusCode == 404
}

// isKind is the errors.Is shim used by the predicate helpers.
func isKind(err error, sentinel *VaultError) bool {
	e, ok := err.(*VaultError)
	return ok && e.Is(sentinel)
}
