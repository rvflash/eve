// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package eve

import (
	"strconv"

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

// Client represents the EVE client to handle the data sources.
type Client struct {
	project,
	firstEnv, secondEnv string
	Handler
}

// New returns an instance of a Client.
// The first parameter is the project's identifier.
// The second, optional, represents a list of dsn to use as remote rpc cache.
// The syntax of one addr is "host:port", like "127.0.0.1:9090".
// If host is omitted, addr must be like ":9090".
func New(project string, addr ...string) (*Client, error) {
	c := &Client{
		project: project,
		Handler: DefaultHandler,
	}
	if len(addr) == 0 {
		return nil, ErrDataSource
	}
	for _, dsn := range addr {
		rpc, err := client.OpenRPC(dsn, client.DefaultCacheDuration)
		if err != nil {
			return c, err
		}
		c.Handler.Add(rpc)
	}
	return c, nil
}

// Lazy is like New but it returns on error only if
// in the list of requested remote RPC services, none are alive.
func Lazy(project string, addr ...string) (*Client, error) {
	c := &Client{
		project: project,
		Handler: DefaultHandler,
	}
	e := ErrDataSource
	for _, dsn := range addr {
		rpc, err := client.OpenRPC(dsn, client.DefaultCacheDuration)
		if err != nil {
			e = errors.WithMessage(e, err.Error())
			continue
		}
		c.Handler.Add(rpc)
	}
	if e != nil && len(c.Handler) == len(DefaultHandler) {
		return c, e
	}
	return c, nil
}

// SetEnvs allows to define until 2 environments.
// The adding's order is important, the first must be
// the first environment defined in the EVE's project.
// It returns an error if the number of environment is unexpected.
func (c *Client) SetEnvs(envs ...string) error {
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
func (c *Client) Lookup(key string) (v interface{}, ok bool) {
	key = c.deployKey(key)
	for _, h := range c.Handler {
		if v, ok = h.Lookup(key); ok {
			return
		}
	}
	return
}

// todo get values on each element of a struct.
// func (c *Client) Process(spec interface{}) error
// func (c *Client) MustProcess(spec interface{})

// Bool uses the key to get the variable's value behind as a boolean.
func (c *Client) Bool(key string) (bool, error) {
	d, ok := c.Lookup(key)
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
	d, ok := c.Lookup(key)
	if !ok {
		c.fatal("Bool", key, ErrNotFound)
	}
	return d.(bool)
}

// Int uses the key to get the variable's value behind as an int.
func (c *Client) Int(key string) (int, error) {
	d, ok := c.Lookup(key)
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
	d, ok := c.Lookup(key)
	if !ok {
		c.fatal("Int", key, ErrNotFound)
	}
	return d.(int)
}

// Float32 uses the key to get the variable's value behind as a float32.
func (c *Client) Float32(key string) (float32, error) {
	d, ok := c.Lookup(key)
	if !ok {
		return 0, ErrNotFound
	}
	f, ok := d.(float32)
	if !ok {
		return 0, ErrInvalid
	}
	return f, nil
}

// MustFloat32 is like Float32 but panics if the variable cannot be retrieved.
func (c *Client) MustFloat32(key string) float32 {
	d, ok := c.Lookup(key)
	if !ok {
		c.fatal("Float32", key, ErrNotFound)
	}
	return d.(float32)
}

// Float64 uses the key to get the variable's value behind as a float64.
func (c *Client) Float64(key string) (float64, error) {
	d, ok := c.Lookup(key)
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
	d, ok := c.Lookup(key)
	if !ok {
		c.fatal("Float64", key, ErrNotFound)
	}
	return d.(float64)
}

// String uses the key to get the variable's value behind as a string.
func (c *Client) String(key string) (string, error) {
	d, ok := c.Lookup(key)
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
	d, ok := c.Lookup(key)
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

func (c *Client) deployKey(key string) string {
	return deploy.Key(c.project, c.firstEnv, c.secondEnv, key)
}

// DefaultHandler defines the sort order to use to fetch data source.
// By default, eve tries to get the variable's value:
// > In its own memory cache.
// > In the list of available environment variables.
var DefaultHandler = Handler{0: Cache, 1: OS}

// Handler returns the list of data sources in the order in which they are used.
type Handler map[int]client.GetSetter

// Add adds a client to the scheduler.
func (h Handler) Add(src client.GetSetter) Handler {
	h[len(h)] = src
	return h
}
