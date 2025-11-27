// Package common contains shared types and utilities for lab4.
package common

import (
	"context"
)

type IDctxKey struct{}
type ExpectedVersionCtxKey struct{}
type VersionCtxKey struct{}

func CtxWithID(id string) context.Context {
	return context.WithValue(context.Background(), IDctxKey{}, id)
}

func IDFromCtx(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(IDctxKey{}).(string)
	return id, ok
}

// CtxWithExpectedVersion adds expected version for optimistic concurrency control
func CtxWithExpectedVersion(ctx context.Context, version int) context.Context {
	return context.WithValue(ctx, ExpectedVersionCtxKey{}, version)
}

// ExpectedVersionFromCtx retrieves expected version from context
func ExpectedVersionFromCtx(ctx context.Context) (int, bool) {
	version, ok := ctx.Value(ExpectedVersionCtxKey{}).(int)
	return version, ok
}

// CtxWithVersion adds version information (used for replication)
func CtxWithVersion(ctx context.Context, version int) context.Context {
	return context.WithValue(ctx, VersionCtxKey{}, version)
}

// VersionFromCtx retrieves version from context
func VersionFromCtx(ctx context.Context) (int, bool) {
	version, ok := ctx.Value(VersionCtxKey{}).(int)
	return version, ok
}
