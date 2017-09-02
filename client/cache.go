// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package client

import (
	"sync"
	"time"
)

// CacheDefaultDuration is the default duration to keep data in cache.
var DefaultCacheDuration = 15 * time.Minute

// Cache represents the service to access data as a memory cache.
type Cache struct {
	data       map[string]*cacheItem
	mu         *sync.RWMutex
	recycle    *time.Ticker
	expiration time.Duration
}

// NewCache returns a new instance of the cache and starts the recycler.
// The Close method must be called to properly close the recycler
// and avoids leaks.
func NewCache(duration time.Duration) *Cache {
	c := &Cache{
		data:       make(map[string]*cacheItem),
		mu:         &sync.RWMutex{},
		recycle:    time.NewTicker(duration),
		expiration: duration,
	}
	go func() {
		for range c.recycle.C {
			c.clean()
		}
	}()
	return c
}

// Delete removes the data behind the key in the cache.
func (c *Cache) Delete(key string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.data, key)
	return nil
}

// Close stops the ticker to clean the cache.
func (c *Cache) Close() error {
	c.recycle.Stop()
	return nil
}

// Get returns the value behind the key in cache.
func (c *Cache) Get(key string) interface{} {
	d, _ := c.lookup(key)
	return d
}

// Lookup returns the value behind the key in cache.
// If the key is not found, the boolean is false.
func (c *Cache) Lookup(key string) (interface{}, bool) {
	return c.lookup(key)
}

// Sets puts the value in cache with an expiration date
// fixed by the cache duration.
func (c *Cache) Set(key string, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Creates the item and saves it in cache.
	c.data[key] = &cacheItem{
		data:    value,
		expires: time.Now().Add(c.expiration),
	}
	return nil
}

func (c *Cache) clean() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Removes all expired items.
	for key, item := range c.data {
		if item.expired() {
			delete(c.data, key)
		}
	}
}

func (c *Cache) lookup(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Retrieves the item in cache.
	item, ok := c.data[key]
	if !ok {
		return nil, ok
	}
	if item.expired() {
		// Item has expired, deletes it.
		delete(c.data, key)
		return nil, false
	}
	return item.data, true
}

// cacheItem represents the data to store with its expire date.
type cacheItem struct {
	data    interface{}
	expires time.Time
}

// expired returns true if the item in cache has expired.
func (i *cacheItem) expired() bool {
	if !i.expires.IsZero() {
		return time.Now().After(i.expires)
	}
	return false
}
