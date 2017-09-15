// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package rpc

import (
	"errors"
	"sync"
	"time"
)

// DefaultPort is the port uses by default
// to launch the RPC cache server.
const DefaultPort = 1010

// DefaultTimeout is the default timeout in ms.
const DefaultTimeout = 100 * time.Millisecond

// ErrNotFound is triggered when the data is not found
// in the remote cache.
var ErrNotFound = errors.New("not found")

// Cache represents the service to access data as a remote cache.
type Cache struct {
	data  map[string]interface{}
	stats *Metrics
	mu    *sync.RWMutex
	up    time.Time
}

// Item represents a data to store.
type Item struct {
	Key   string
	Value interface{}
}

// Metrics exposes some data about the cache usage.
type Metrics struct {
	Items  uint64
	UpTime time.Duration
	Requests
}

// Requests lists all available methods of the service.
type Requests struct {
	Bulk, Clear, Delete, Get, Put uint64
}

// New returns a new instance of Cache.
func New() *Cache {
	return &Cache{
		data:  make(map[string]interface{}),
		stats: &Metrics{},
		mu:    &sync.RWMutex{},
		up:    time.Now(),
	}
}

// Bulk applies the item's modifications on the cache in one batch.
// Item with nil value will be deleted.
func (c *Cache) Bulk(batch []*Item, ack *bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Applies the modifications.
	for _, i := range batch {
		_, found := c.data[i.Key]
		if i.Value == nil {
			if found {
				delete(c.data, i.Key)
				c.stats.Items--
			}
		} else {
			if !found {
				c.stats.Items++
			}
			c.data[i.Key] = i.Value
		}
	}
	*ack = true

	// Increments the statistics.
	c.stats.Bulk++

	return nil
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
	c.stats.Delete++
	c.stats.Items--

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
	c.stats.Clear++
	c.stats.Items = 0

	return nil
}

// Get gets the value of the given key or an error if it not exists.
// resp contains the data to return to clients.
func (c *Cache) Get(key string, resp *Item) error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Retrieves the item.
	value, found := c.data[key]
	if !found {
		return ErrNotFound
	}
	*resp = Item{key, value}

	// Increments the statistics.
	c.stats.Get++
	return nil
}

// Put puts this item in the cache.
// ack is used to return acknowledgements to clients.
func (c *Cache) Put(item *Item, ack *bool) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Puts the item.
	c.data[item.Key] = item.Value
	*ack = true

	// Increments the statistics.
	c.stats.Put++
	c.stats.Items++

	return nil
}

// Stats returns various statistics about this cache's instance.
func (c *Cache) Stats(all bool, data *Metrics) error {
	// Number of seconds since the last restart of the server.
	c.stats.UpTime = time.Since(c.up)
	*data = *c.stats
	return nil
}
