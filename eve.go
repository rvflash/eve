// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package eve

import (
	"reflect"
	"strconv"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/rvflash/eve/caseconv"
	"github.com/rvflash/eve/client"
	"github.com/rvflash/eve/deploy"
)

// Error messages
var (
	ErrInvalid    = errors.New("invalid data")
	ErrNotFound   = errors.New("not found")
	ErrDataSource = errors.New("no available rpc service")
	ErrNoPointer  = errors.New("mandatory struct pointer")
)

// Initializes the data sources.
var (
	Cache = client.NewCache(client.DefaultCacheDuration)
	OS    = &client.OS{}
)

// Tick is the time duration to sleep before checking
// if at least one RPC cache is available.
var Tick = time.Minute

// Handler returns the list of data sources in the order
// in which they are used.
type Handler map[int]client.Getter

// Add adds a client to the scheduler.
func (h Handler) AddHandler(src client.Getter) Handler {
	h[len(h)] = src
	return h
}

// Servers tries to connect to each net address and returns them.
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

// Client represents the EVE client to handle the data sources.
type Client struct {
	project,
	firstEnv, secondEnv string
	alive *time.Ticker
	mu    sync.Mutex
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
		c.Handler.AddHandler(servers[i])
	}
	// Checks if at least one server is alive.
	go func() {
		for range c.alive.C {
			c.mu.Lock()
			c.fresh()
			c.mu.Unlock()
		}
	}()
	return c
}

// Returns the local cache if used as handler.
func (c *Client) cache() *client.Cache {
	for _, h := range c.Handler {
		if cache, ok := h.(*client.Cache); ok {
			return cache
		}
	}
	return nil
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
	if lc := c.cache(); lc != nil {
		if alive {
			if !lc.WithExpiration() {
				// The local cache has no expiration process
				// but at least one RPC cache is now alive, we can enable it.
				lc.UseExpiration()
			}
		} else if lc.WithExpiration() {
			// All RPC lcs are down, we temporary disable
			// the expiration of the local cache.
			lc.NoExpiration()
		}
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
	c.mu.Lock()
	c.Handler = h
	c.mu.Unlock()
	return c
}

// Get retrieves the value of the environment variable named by the key.
// If it not exists, a nil value is returned.
func (c *Client) Get(key string) interface{} {
	if v, ok := c.assert(key, client.StringVal); ok {
		return v
	}
	return nil
}

// Lookup retrieves the value of the environment variable named by the key.
// If it not exists, the second boolean will be false.
func (c *Client) Lookup(key string) (interface{}, bool) {
	return c.assert(key, client.StringVal)
}

// Tries to get the value of the variable by it key.
// Asserts the value if the client needs it.
// It returns a boolean as second parameter to indicate if the key was found.
func (c *Client) assert(key string, typ client.Kind) (v interface{}, ok bool) {
	key = c.deployKey(key)
	for _, h := range c.Handler {
		if v, ok = h.Lookup(key); ok {
			if ha, needAssert := h.(client.Asserter); needAssert {
				v, ok = ha.Assert(v, typ)
			}
			// If the current handler is the local cache,
			// no need to save the data.
			if _, k := h.(*client.Cache); k {
				return
			}
			if lc := c.cache(); lc != nil {
				// Saves the data in the local cache.
				_ = lc.Set(key, v)
			}
			return
		}
	}
	return
}

// Returns a deploy key by building it with all its pieces,
// the project name, environments values and variable name.
func (c *Client) deployKey(key string) string {
	return deploy.Key(c.project, c.firstEnv, c.secondEnv, key)
}

// All information about the specification struct to feed.
type varInfo struct {
	Field    reflect.StructField
	Key      string
	Value    reflect.Value
	Required bool
}

// Reads each field on the struct to get its properties.
func readStruct(spec interface{}) ([]varInfo, error) {
	rv := reflect.ValueOf(spec)
	if rv.Kind() != reflect.Ptr {
		return nil, ErrNoPointer
	}
	rv = rv.Elem()
	if rv.Kind() != reflect.Struct {
		return nil, ErrNoPointer
	}
	// Retrieves the data key to use.
	// First by fetching the EVE tag name and then the field name itself.
	key := func(field reflect.StructField) string {
		if tag := field.Tag.Get("eve"); tag != "" {
			return tag
		}
		return caseconv.SnakeCase(field.Name)
	}
	// Returns true if the field is tagged as mandatory.
	required := func(field reflect.StructField) bool {
		b, _ := strconv.ParseBool(field.Tag.Get("required"))
		return b
	}
	kind := rv.Type()
	info := make([]varInfo, 0, rv.NumField())
	for i := 0; i < rv.NumField(); i++ {
		f := rv.Field(i)
		ft := kind.Field(i)
		if !f.CanSet() {
			continue
		}
		for f.Kind() == reflect.Ptr {
			if f.IsNil() {
				if f.Type().Elem().Kind() != reflect.Struct {
					// nil pointer to a non-struct
					break
				}
				// create an zero struct
				f.Set(reflect.New(f.Type().Elem()))
			}
			f = f.Elem()
		}
		info = append(info, varInfo{
			Field:    ft,
			Key:      key(ft),
			Value:    f,
			Required: required(ft),
		})
	}
	return info, nil
}

// Process uses the reflection to assign values on each element.
// It returns on error if one on its fields can not be retrieved.
func (c *Client) Process(spec interface{}) error {
	infos, err := readStruct(spec)
	if err != nil {
		return err
	}
	// Sets the value of the given field.
	feed := func(f varInfo) error {
		typ := f.Value.Type()
		if typ.Kind() == reflect.Ptr {
			typ = typ.Elem()
			if f.Value.IsNil() {
				f.Value.Set(reflect.New(typ))
			}
			f.Value = f.Value.Elem()
		}
		switch typ.Kind() {
		case reflect.String:
			v, err := c.String(f.Key)
			if err != nil && f.Required {
				return err
			}
			f.Value.SetString(v)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			v, err := c.Int(f.Key)
			if err != nil {
				// Second chance by expecting time duration in string like 300ms.
				if f.Value.Kind() == reflect.Int64 && typ.PkgPath() == "time" && typ.Name() == "Duration" {
					var s string
					if s, err = c.String(f.Key); err == nil {
						var d time.Duration
						d, err = time.ParseDuration(s)
						v = int(d)

					}
				}
			}
			if err != nil && f.Required {
				return err
			}
			f.Value.SetInt(int64(v))
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			v, err := c.Int(f.Key)
			if err != nil && f.Required {
				return err
			}
			f.Value.SetUint(uint64(v))
		case reflect.Bool:
			v, err := c.Bool(f.Key)
			if err != nil && f.Required {
				return err
			}
			f.Value.SetBool(v)
		case reflect.Float32, reflect.Float64:
			v, err := c.Float64(f.Key)
			if err != nil && f.Required {
				return err
			}
			f.Value.SetFloat(v)
		default:
			//todo Manages time.time from String var.
		}

		return nil
	}
	for _, info := range infos {
		if err := feed(info); err != nil {
			return err
		}
	}
	return nil
}

// MustProcess is like Process but panics if it fails to feed the spec.
func (c *Client) MustProcess(spec interface{}) {
	if err := c.Process(spec); err != nil {
		panic(`eve: ` + err.Error())
	}
}

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
	d, err := c.Bool(key)
	if err != nil {
		c.fatal("Bool", key, err)
	}
	return d
}

// Int uses the key to get the variable's value behind as an int.
func (c *Client) Int(key string) (int, error) {
	d, ok := c.assert(key, client.IntVal)
	if !ok {
		return 0, ErrNotFound
	}
	i, ok := d.(int)
	if !ok {
		// JSON unmarshal stores the numbers as float64.
		// On restarting, the RPC cache retrieves the data from a JSON.
		// We needs to manage this behavior.
		f, ok := d.(float64)
		if !ok {
			return 0, ErrInvalid
		}
		i = int(f)
	}
	return i, nil
}

// MustInt is like Int but panics if the variable cannot be retrieved.
func (c *Client) MustInt(key string) int {
	d, err := c.Int(key)
	if err != nil {
		c.fatal("Int", key, err)
	}
	return d
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
	d, err := c.Float64(key)
	if err != nil {
		c.fatal("Float64", key, err)
	}
	return d
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
	d, err := c.String(key)
	if err != nil {
		c.fatal("String", key, err)
	}
	return d
}

// Panic!
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
