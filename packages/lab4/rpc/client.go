package rpc

import (
	"context"
	"net/rpc"

	"github.com/andyp1xe1/network_programming_labs/packages/lab4/common"
)

// KVrpcClient implements the store.Store interface over RPC
type KVrpcClient struct {
	*rpc.Client
}

// NewKVrpcClient creates a new KVrpcClient.
func NewKVrpcClient(address string) (*KVrpcClient, error) {
	client, err := rpc.DialHTTP("tcp", address)
	if err != nil {
		return nil, err
	}
	return &KVrpcClient{client}, nil
}

// Set implements store.Store.Set over RPC
func (c *KVrpcClient) Set(ctx context.Context, key, value string) error {
	id, _ := common.IDFromCtx(ctx)
	return c.Call("KVrpc.Set", &SetArgs{ID: id, Key: key, Value: value}, &SetReply{})
}

// Get implements store.Store.Get over RPC
func (c *KVrpcClient) Get(ctx context.Context, key string) (string, error) {
	var reply GetReply
	err := c.Call("KVrpc.Get", &GetArgs{Key: key}, &reply)
	if err != nil {
		return "", err
	}
	return reply.Value, nil
}

// Delete implements store.Store.Delete over RPC
func (c *KVrpcClient) Delete(ctx context.Context, key string) error {
	id, _ := common.IDFromCtx(ctx)
	return c.Call("KVrpc.Delete", &DeleteArgs{ID: id, Key: key}, &DeleteReply{})
}

// Exists implements store.Store.Exists over RPC
func (c *KVrpcClient) Exists(ctx context.Context, key string) (bool, error) {
	var exists bool
	err := c.Call("KVrpc.Exists", &GetArgs{Key: key}, &exists)
	if err != nil {
		return false, err
	}
	return exists, nil
}
