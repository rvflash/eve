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
	c *rpc.Client
}

// OpenRPC returns an instance of RPC with a TCP connection to it.
// DSN is in the form of "localhost:1010".
func OpenRPC(dsn string, timeout time.Duration) (*RPC, error) {
	c, err := net.DialTimeout("tcp", dsn, timeout)
	if err != nil {
		return nil, err
	}
	return &RPC{c: rpc.NewClient(c)}, nil
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

// Raw returns the value behind the key or an error if it not exists
func (r *RPC) Raw(key string) (interface{}, error) {
	var item *cache.CacheItem
	if err := r.c.Call("Cache.Get", key, &item); err != nil {
		return nil, err
	}
	return item.Value, nil
}

// Set saves the item and acknowledges the boolean if it succeeds.
// An error occurs if the call fails.
func (r *RPC) Set(key string, value interface{}) error {
	item := &cache.CacheItem{Key: key, Value: value}
	var added bool
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
func (r *RPC) Stats() (*cache.Requests, error) {
	req := &cache.Requests{}
	err := r.c.Call("Cache.Stats", true, req)
	return req, err
}
