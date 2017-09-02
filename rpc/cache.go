// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package rpc

import (
	"errors"
	"sync"
)

// DefaultPort is the port uses by default
// to launch the RPC cache server.
const DefaultPort = 1010

// ErrNotFound is triggered when the data is not found
// in the remote cache.
var ErrNotFound = errors.New("not found")

// Cache represents the service to access data as a remote cache.
type Cache struct {
	data map[string]interface{}
	mu   *sync.RWMutex
	r    *Requests
}

// CacheItem represents a data to store.
type CacheItem struct {
	Key   string
	Value interface{}
}

// Requests lists all available methods of the service.
type Requests struct {
	Get, Put, Delete, Clear uint64
}

// New returns a new instance of Cache.
func New() *Cache {
	return &Cache{
		data: make(map[string]interface{}),
		mu:   &sync.RWMutex{},
		r:    &Requests{},
	}
}

// Delete deletes the key in the cache.
// ack is used to return acknowledgements to clients.
func (c *Cache) Delete(key string, ack *bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Deletes the item.
	if _, found := c.data[key]; !found {
		return ErrNotFound
	}
	delete(c.data, key)
	*ack = true

	// Increments the statistics.
	c.r.Delete++

	return nil
}

// Clear clears all cache items, acknowledges clear
// ack is used to return acknowledgements to clients.
func (c *Cache) Clear(skip bool, ack *bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Resets the cache.
	c.data = make(map[string]interface{})
	*ack = true

	// Increments the statistics.
	c.r.Clear++

	return nil
}

// Get gets the value of the given key or an error if it not exists.
// resp contains the data to return to clients.
func (c *Cache) Get(key string, resp *CacheItem) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Retrieves the item.
	value, found := c.data[key]
	if !found {
		return ErrNotFound
	}
	*resp = CacheItem{key, value}

	// Increments the statistics.
	c.r.Get++
	return nil
}

// Put puts this item in the cache.
// ack is used to return acknowledgements to clients.
func (c *Cache) Put(item *CacheItem, ack *bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Puts the item.
	c.data[item.Key] = item.Value
	*ack = true

	// Increments the statistics.
	c.r.Put++

	return nil
}

// Stats returns various statistics about this cache's instance.
func (c *Cache) Stats(all bool, requests *Requests) error {
	*requests = *c.r
	return nil
}
