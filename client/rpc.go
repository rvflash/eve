// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package client

import (
	"net"
	"net/rpc"
	"time"

	cache "github.com/rvflash/eve/rpc"
)

// RPC is client with connection to cache'service.
type RPC struct {
	c Caller
}

// NewRPC returns a new instance of RPC.
func NewRPC(conn Caller) *RPC {
	return &RPC{c: conn}
}

// OpenRPC returns an instance of RPC with a TCP connection into it.
// DSN is in the form of "localhost:9090".
// If the connection fails, it returns the error.
func OpenRPC(dsn string, timeout time.Duration) (*RPC, error) {
	c, err := net.DialTimeout("tcp", dsn, timeout)
	if err != nil {
		return nil, err
	}
	return NewRPC(rpc.NewClient(c)), nil
}

// Bulk applies the item modifications on the cache.
// It acknowledges the boolean if it succeeds.
// An error occurs if the call fails.
func (r *RPC) Bulk(batch map[string]interface{}) error {
	var i int
	if i = len(batch); i == 0 {
		return nil
	}
	items := make([]*cache.Item, i)
	for k, v := range batch {
		i--
		items[i] = &cache.Item{Key: k, Value: v}
	}
	var bulked bool
	if err := r.c.Call("Cache.Bulk", items, &bulked); err != nil {
		return err
	}
	if !bulked {
		return ErrFailure
	}
	return nil
}

// Clear resets the cache and acknowledges the boolean if it succeeds.
// An error occurs if the call fails.
func (r *RPC) Clear() error {
	var cleared bool
	if err := r.c.Call("Cache.Clear", true, &cleared); err != nil {
		return err
	}
	if !cleared {
		return ErrFailure
	}
	return nil
}

// Delete removes this key in the cache and acknowledges the boolean if it succeeds.
// An error occurs if the call fails.
func (r *RPC) Delete(key string) error {
	var deleted bool
	if err := r.c.Call("Cache.Delete", key, &deleted); err != nil {
		return err
	}
	if !deleted {
		return ErrFailure
	}
	return nil
}

// Get returns the value behind the key in the cache.
func (r *RPC) Get(key string) interface{} {
	value, _ := r.Raw(key)
	return value
}

// Lookup gets the value of the environment variable named by the key.
// If the variable is present in the environment, the value (which may be empty)
// is returned and the boolean is true.
// Otherwise the returned value will be empty and the boolean will be false.
func (r *RPC) Lookup(key string) (interface{}, bool) {
	value, err := r.Raw(key)
	return value, err == nil
}

// NeedAssert implements the Getter interface.
func (r *RPC) NeedAssert() bool {
	return false
}

// Raw returns the value behind the key or an error if it not exists
func (r *RPC) Raw(key string) (interface{}, error) {
	var item cache.Item
	if err := r.c.Call("Cache.Get", key, &item); err != nil {
		return nil, err
	}
	return item.Value, nil
}

// Set saves the item and acknowledges the boolean if it succeeds.
// An error occurs if the call fails.
func (r *RPC) Set(key string, value interface{}) error {
	var added bool
	item := &cache.Item{Key: key, Value: value}
	if err := r.c.Call("Cache.Put", item, &added); err != nil {
		return err
	}
	if !added {
		return ErrFailure
	}
	return nil
}

// Stats returns statistics about the current server.
// An error occurs and returned if the call fails.
func (r *RPC) Stats() (*cache.Metrics, error) {
	req := &cache.Metrics{}
	err := r.c.Call("Cache.Stats", true, req)
	return req, err
}
