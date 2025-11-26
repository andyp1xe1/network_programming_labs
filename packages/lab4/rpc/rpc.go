// Package rpc implements an RPC server for a key-value store.
package rpc

import (
	"context"
	"net"
	"net/http"
	"net/rpc"

	"github.com/andyp1xe1/network_programming_labs/packages/lab4/common"
	"github.com/andyp1xe1/network_programming_labs/packages/lab4/store"
)

type KVrpc struct {
	store.Store
}

func newKVrpc(store store.Store) *KVrpc {
	return &KVrpc{store}
}

func ExposeRPC(st store.Store, address string) error {
	frpc := newKVrpc(st)
	rpc.Register(frpc)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	http.Serve(l, nil)
	return nil
}

func (kvrpc *KVrpc) Set(args *SetArgs, reply *SetReply) error {
	ctx := common.CtxWithID(args.ID)
	return kvrpc.Store.Set(ctx, args.Key, args.Value)
}

func (kvrpc *KVrpc) Get(args *GetArgs, reply *GetReply) error {
	value, err := kvrpc.Store.Get(context.Background(), args.Key)
	if err != nil {
		return err
	}
	reply.Value = value
	return nil
}

func (kvrpc *KVrpc) Delete(args *DeleteArgs, reply *DeleteReply) error {
	ctx := common.CtxWithID(args.ID)
	return kvrpc.Store.Delete(ctx, args.Key)
}

func (kvrpc *KVrpc) Exists(args *GetArgs, reply *bool) error {
	exists, err := kvrpc.Store.Exists(context.Background(), args.Key)
	if err != nil {
		return err
	}
	*reply = exists
	return nil
}
