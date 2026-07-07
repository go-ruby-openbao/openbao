// Copyright (c) the go-ruby-openbao/openbao authors
//
// SPDX-License-Identifier: BSD-3-Clause

package openbao

import (
	"strconv"
	"strings"
)

// KVv1 is a KV version-1 secrets-engine helper rooted at a mount (default
// "secret"). Paths are joined under the mount; the payload is stored flat, so
// the verbs delegate straight to [Logical].
type KVv1 struct {
	c     *Client
	mount string
}

// path joins the mount and the relative secret path.
func (k *KVv1) path(p string) string {
	return k.mount + "/" + strings.TrimLeft(p, "/")
}

// Read reads the KV v1 secret at path.
func (k *KVv1) Read(path string) (*Secret, error) { return k.c.Logical().Read(k.path(path)) }

// Write writes the KV v1 secret at path.
func (k *KVv1) Write(path string, data map[string]any) (*Secret, error) {
	return k.c.Logical().Write(k.path(path), data)
}

// List lists the keys under the KV v1 path.
func (k *KVv1) List(path string) (*Secret, error) { return k.c.Logical().List(k.path(path)) }

// Delete deletes the KV v1 secret at path.
func (k *KVv1) Delete(path string) (*Secret, error) { return k.c.Logical().Delete(k.path(path)) }

// KVv2 is a KV version-2 secrets-engine helper rooted at a mount (default
// "secret"). It performs the version-2 path rewriting (data/, metadata/,
// delete/, undelete/, destroy/) and the versioned request bodies the engine
// expects.
type KVv2 struct {
	c     *Client
	mount string
}

// prefixed builds "<mount>/<segment>/<path>".
func (k *KVv2) prefixed(segment, p string) string {
	return k.mount + "/" + segment + "/" + strings.TrimLeft(p, "/")
}

// Read reads the latest version of the KV v2 secret at path
// (GET <mount>/data/<path>).
func (k *KVv2) Read(path string) (*Secret, error) {
	return k.c.Logical().Read(k.prefixed("data", path))
}

// ReadVersion reads a specific version of the KV v2 secret at path
// (GET <mount>/data/<path>?version=N).
func (k *KVv2) ReadVersion(path string, version int) (*Secret, error) {
	p := k.prefixed("data", path) + "?version=" + strconv.Itoa(version)
	return k.c.Logical().Read(p)
}

// Write writes a new version of the KV v2 secret at path, wrapping data in the
// version-2 {"data": …} envelope (PUT <mount>/data/<path>); the returned secret
// carries the new version metadata.
func (k *KVv2) Write(path string, data map[string]any) (*Secret, error) {
	return k.c.Logical().Write(k.prefixed("data", path), map[string]any{"data": data})
}

// List lists the keys under the KV v2 path (LIST <mount>/metadata/<path>).
func (k *KVv2) List(path string) (*Secret, error) {
	return k.c.Logical().List(k.prefixed("metadata", path))
}

// Delete soft-deletes the latest version of the KV v2 secret at path
// (DELETE <mount>/data/<path>).
func (k *KVv2) Delete(path string) (*Secret, error) {
	return k.c.Logical().Delete(k.prefixed("data", path))
}

// DeleteVersions soft-deletes specific versions of the KV v2 secret at path
// (POST <mount>/delete/<path>).
func (k *KVv2) DeleteVersions(path string, versions ...int) (*Secret, error) {
	return k.c.do("POST", k.prefixed("delete", path), versionsBody(versions))
}

// Undelete restores previously soft-deleted versions of the KV v2 secret at path
// (POST <mount>/undelete/<path>).
func (k *KVv2) Undelete(path string, versions ...int) (*Secret, error) {
	return k.c.do("POST", k.prefixed("undelete", path), versionsBody(versions))
}

// Destroy permanently destroys specific versions of the KV v2 secret at path
// (POST <mount>/destroy/<path>).
func (k *KVv2) Destroy(path string, versions ...int) (*Secret, error) {
	return k.c.do("POST", k.prefixed("destroy", path), versionsBody(versions))
}

// ReadMetadata reads the KV v2 metadata for path
// (GET <mount>/metadata/<path>).
func (k *KVv2) ReadMetadata(path string) (*Secret, error) {
	return k.c.Logical().Read(k.prefixed("metadata", path))
}

// WriteMetadata writes the KV v2 metadata for path
// (PUT <mount>/metadata/<path>).
func (k *KVv2) WriteMetadata(path string, meta map[string]any) (*Secret, error) {
	return k.c.Logical().Write(k.prefixed("metadata", path), meta)
}

// DeleteMetadata deletes the KV v2 metadata (and every version) for path
// (DELETE <mount>/metadata/<path>).
func (k *KVv2) DeleteMetadata(path string) (*Secret, error) {
	return k.c.Logical().Delete(k.prefixed("metadata", path))
}

// versionsBody builds the {"versions":[…]} request body the delete/undelete/
// destroy endpoints expect.
func versionsBody(versions []int) map[string]any {
	return map[string]any{"versions": versions}
}
