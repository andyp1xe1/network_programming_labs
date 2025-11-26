// Package common contains shared types and utilities for lab4.
package common

import (
	"context"
)

type IDctxKey struct{}

func CtxWithID(id string) context.Context {
	return context.WithValue(context.Background(), IDctxKey{}, id)
}

func IDFromCtx(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(IDctxKey{}).(string)
	return id, ok
}
