// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package client

import (
	"net"
	"net/rpc"
	"sync"
	"time"

	cache "github.com/rvflash/eve/rpc"
)

// OpenRPC returns an instance of RPC with a TCP connection into it.
// DSN is in the form of "localhost:9090".
// If the connection fails, it returns the error.
// Unlike NewRPC, OpenRPC has an internal mechanism to reconnect on failure.
func OpenRPC(dsn string, timeout time.Duration) (*RPC, error) {
	conn, err := newRPC(dsn, timeout)
	if err != nil {
		return nil, err
	}
	c := &RPC{
		c:       conn,
		dsn:     dsn,
		tick:    time.NewTicker(time.Second),
		timeout: timeout,
	}
	go func() {
		for range c.tick.C {
			c.reconnectOnFail()
		}
	}()
	return c, nil
}

func newRPC(dsn string, timeout time.Duration) (*rpc.Client, error) {
	c, err := net.DialTimeout("tcp", dsn, timeout)
	if err != nil {
		return nil, err
	}
	return rpc.NewClient(c), nil
}

// NewRPC returns a new instance of RPC.
// This instance has no mechanism to reconnect on failure.
func NewRPC(conn Caller) *RPC {
	return &RPC{c: conn}
}

// RPC is client with connection to cache'service.
type RPC struct {
	c       Caller
	dsn     string
	mu      sync.Mutex
	tick    *time.Ticker
	timeout time.Duration
}

func (r *RPC) reconnectOnFail() {
	if r.Available() {
		return
	}
	c, err := newRPC(r.dsn, r.timeout)
	if err != nil {
		return
	}
	r.mu.Lock()
	r.c = c
	r.mu.Unlock()
}

// Available implements the Checker interface.
func (r *RPC) Available() bool {
	_, err := r.Stats()
	return err == nil
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
	if err := r.call("Cache.Bulk", items, &bulked); err != nil {
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
	if err := r.call("Cache.Clear", true, &cleared); err != nil {
		return err
	}
	if !cleared {
		return ErrFailure
	}
	return nil
}

// Close closes the connection.
func (r *RPC) Close() error {
	if r.tick != nil {
		r.tick.Stop()
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.c != nil {
		return r.c.Close()
	}
	return nil
}

// Delete removes this key in the cache and acknowledges the boolean if it succeeds.
// An error occurs if the call fails.
func (r *RPC) Delete(key string) error {
	var deleted bool
	if err := r.call("Cache.Delete", key, &deleted); err != nil {
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
	var item cache.Item
	if err := r.call("Cache.Get", key, &item); err != nil {
		return nil, err
	}
	return item.Value, nil
}

// Set saves the item and acknowledges the boolean if it succeeds.
// An error occurs if the call fails.
func (r *RPC) Set(key string, value interface{}) error {
	var added bool
	item := &cache.Item{Key: key, Value: value}
	if err := r.call("Cache.Put", item, &added); err != nil {
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
	err := r.call("Cache.Stats", true, req)
	return req, err
}

func (r *RPC) call(service string, args, reply interface{}) error {
	if r.c == nil {
		return ErrConn
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.c.Call(service, args, reply)
}
