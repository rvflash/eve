// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package eve

import (
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/rvflash/eve/client"
	"github.com/rvflash/eve/deploy"
)

// Error messages
var (
	ErrInvalid    = errors.New("invalid data")
	ErrNotFound   = errors.New("not found")
	ErrDataSource = errors.New("no available rpc service")
)

// Initializes the data sources.
var (
	Cache = client.NewCache(client.DefaultCacheDuration)
	OS    = &client.OS{}
)

// Time duration to sleep before checking if at least one RPC cache
// is available.
var Tick = time.Minute

// Handler returns the list of data sources in the order
// in which they are used.
type Handler map[int]client.Getter

// Add adds a client to the scheduler.
func (h Handler) Add(src client.Getter) Handler {
	h[len(h)] = src
	return h
}

// Client represents the EVE client to handle the data sources.
type Client struct {
	project,
	firstEnv, secondEnv string
	alive *time.Ticker
	Handler
}

// New returns an instance of a Client.
// The first parameter is the project's identifier.
// The second, optional, represents a list of data getter.
// By default, eve tries to get the variable's value:
// > In its own memory cache.
// > In the list of available environment variables.
// > in the other date getter like RPC cache.
// The Eve client only sets variables in its own cache.
func New(project string, servers ...client.Getter) *Client {
	c := &Client{
		project: project,
		Handler: Handler{0: Cache, 1: OS},
		alive:   time.NewTicker(Tick),
	}
	// Adds more servers as data source.
	for i := 0; i < len(servers); i++ {
		c.Handler.Add(servers[i])
	}
	// Checks if at least one server is alive.
	go func() {
		for range c.alive.C {
			c.fresh()
		}
	}()
	return c
}

// Checks if at least one RPC cache is available.
// If not, we need to deactivate the cache expiration of
// the local cache to preserve its values.
func (c *Client) fresh() {
	var alive bool
	for _, h := range c.Handler {
		if c, ok := h.(client.Checker); ok {
			if alive = c.Available(); alive {
				return
			}
		}
	}
	// Returns the local cache if used.
	cache := func() *client.Cache {
		for _, h := range c.Handler {
			if cache, ok := h.(*client.Cache); ok {
				return cache
			}
		}
		return nil
	}()
	if cache == nil {
		// No local cache as handler.
		return
	}
	if alive {
		if !cache.WithExpiration() {
			// The local cache has no expiration process
			// but at least one RPC cache is now alive, we can enable it.
			cache.UseExpiration()
		}
	} else if cache.WithExpiration() {
		// All RPC caches are down, we temporary disable
		// the expiration of the local cache.
		cache.NoExpiration()
	}
}

// Envs allows to define until 2 environments.
// The adding's order is important, the first must be
// the first environment defined in the EVE's project.
// It returns an error if the number of environment is unexpected.
func (c *Client) Envs(envs ...string) error {
	switch len(envs) {
	case 2:
		c.secondEnv = envs[1]
		fallthrough
	case 1:
		c.firstEnv = envs[0]
	default:
		return ErrInvalid
	}
	return nil
}

// UseHandler defines a new handler to use.
// It returns the updated client.
func (c *Client) UseHandler(h Handler) *Client {
	c.Handler = h
	return c
}

// Lookup retrieves the value of the environment variable named by the key.
// If it not exists, the second boolean will be false.
func (c *Client) Lookup(key string) (interface{}, bool) {
	return c.assert(key, client.StringVal)
}

func (c *Client) assert(key string, kind int) (v interface{}, ok bool) {
	key = c.deployKey(key)
	for _, h := range c.Handler {
		if v, ok = h.Lookup(key); ok {
			if ha, needAssert := h.(client.Asserter); needAssert {
				v, ok = ha.Assert(v, kind)
			}
			return
		}
	}
	return
}

func (c *Client) deployKey(key string) string {
	return deploy.Key(c.project, c.firstEnv, c.secondEnv, key)
}

// todo get values on each element of a struct.
// func (c *Client) Process(spec interface{}) error
// func (c *Client) MustProcess(spec interface{})

// Bool uses the key to get the variable's value behind as a boolean.
func (c *Client) Bool(key string) (bool, error) {
	d, ok := c.assert(key, client.BoolVal)
	if !ok {
		return false, ErrNotFound
	}
	b, ok := d.(bool)
	if !ok {
		return false, ErrInvalid
	}
	return b, nil
}

// MustBool is like Bool but panics if the variable cannot be retrieved.
func (c *Client) MustBool(key string) bool {
	d, ok := c.assert(key, client.BoolVal)
	if !ok {
		c.fatal("Bool", key, ErrNotFound)
	}
	return d.(bool)
}

// Int uses the key to get the variable's value behind as an int.
func (c *Client) Int(key string) (int, error) {
	d, ok := c.assert(key, client.IntVal)
	if !ok {
		return 0, ErrNotFound
	}
	i, ok := d.(int)
	if !ok {
		return 0, ErrInvalid
	}
	return i, nil
}

// MustInt is like Int but panics if the variable cannot be retrieved.
func (c *Client) MustInt(key string) int {
	d, ok := c.assert(key, client.IntVal)
	if !ok {
		c.fatal("Int", key, ErrNotFound)
	}
	return d.(int)
}

// Float64 uses the key to get the variable's value behind as a float64.
func (c *Client) Float64(key string) (float64, error) {
	d, ok := c.assert(key, client.FloatVal)
	if !ok {
		return 0, ErrNotFound
	}
	f, ok := d.(float64)
	if !ok {
		return 0, ErrInvalid
	}
	return f, nil
}

// MustFloat64 is like Float64 but panics if the variable cannot be retrieved.
func (c *Client) MustFloat64(key string) float64 {
	d, ok := c.assert(key, client.FloatVal)
	if !ok {
		c.fatal("Float64", key, ErrNotFound)
	}
	return d.(float64)
}

// String uses the key to get the variable's value behind as a string.
func (c *Client) String(key string) (string, error) {
	d, ok := c.assert(key, client.StringVal)
	if !ok {
		return "", ErrNotFound
	}
	s, ok := d.(string)
	if !ok {
		return "", ErrInvalid
	}
	return s, nil
}

// MustString is like String but panics if the variable cannot be retrieved.
func (c *Client) MustString(key string) string {
	d, ok := c.assert(key, client.StringVal)
	if !ok {
		c.fatal("String", key, ErrNotFound)
	}
	return d.(string)
}

func (c *Client) fatal(method, key string, err error) {
	quote := func(s string) string {
		if strconv.CanBackquote(s) {
			return "`" + s + "`"
		}
		return strconv.Quote(s)
	}
	key = c.deployKey(key)
	panic(`eve: ` + method + `(` + quote(key) + `): ` + err.Error())
}

// Caches try to connect each net addr and returns them.
func Servers(addr ...string) (caches []client.Getter, err error) {
	replicate := len(addr)
	if replicate == 0 {
		return nil, ErrDataSource
	}
	caches = make([]client.Getter, replicate)
	for p, dsn := range addr {
		caches[p], err = client.OpenRPC(dsn, client.DefaultCacheDuration)
		if err != nil {
			return
		}
	}
	return
}
