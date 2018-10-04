// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package eve_test

import (
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/rvflash/eve"
	"github.com/rvflash/eve/client"
)

const (
	boolVal  = true
	intVal   = 42
	floatVal = 3.14
	strVal   = "rv"

	hostVal = "http://sh01.prod"
	portVal = 8080
	toVal   = "300ms"
)

type exFields struct {
	Addr    string `eve:"host" required:"true"`
	Port    int
	Timeout time.Duration `eve:"to"`
	Retry   bool
}

func Example() {
	vars := eve.New("test", server)
	if err := vars.Envs("qa", "fr"); err != nil {
		fmt.Println(err)
		return
	}
	if vars.MustBool("bool") {
		str, _ := vars.String("str")
		fmt.Print(str)
	}
	if d, ok := vars.Lookup("int"); ok {
		fmt.Printf(": %d", d.(int))
	}
	// Output: rv: 42
}

func ExampleClient_Process() {
	vars := eve.New("test", server)
	if err := vars.Envs("qa", "fr"); err != nil {
		fmt.Println(err)
		return
	}
	var mycnf exFields
	if err := vars.Process(&mycnf); err != nil {
		fmt.Println(err)
		return
	}
	fmt.Printf(
		"%s:%d, to=%.1fs, retry=%v",
		mycnf.Addr, mycnf.Port, mycnf.Timeout.Seconds(), mycnf.Retry,
	)
	// Output: http://sh01.prod:8080, to=0.3s, retry=false
}

// newClient returns a test client to fake a RPC cache.
func newClient(d time.Duration) *handler {
	c := &handler{}
	c.timer = time.AfterFunc(d, func() {
		c.mu.Lock()
		c.offline = true
		c.mu.Unlock()
	})
	return c
}

type handler struct {
	timer   *time.Timer
	offline bool
	mu      sync.Mutex
}

// Lookup implements the client.Getter interface.
func (c *handler) Lookup(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.offline {
		return nil, false
	}
	switch key {
	case "TEST_BOOL", "TEST_QA_FR_BOOL":
		return boolVal, true
	case "TEST_INT", "TEST_QA_FR_INT":
		return intVal, true
	case "TEST_FLOAT", "TEST_QA_FR_FLOAT":
		return floatVal, true
	case "TEST_STR", "TEST_QA_FR_STR":
		return strVal, true
	case "TEST_QA_FR_HOST":
		return hostVal, true
	case "TEST_QA_FR_PORT":
		return portVal, true
	case "TEST_QA_FR_TO":
		return toVal, true
	}
	return nil, false
}

// Lookup implements the client.Checker interface.
func (c *handler) Available() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return !c.offline
}

var (
	server       = newClient(time.Minute)
	unsafeServer = newClient(100 * time.Millisecond)
)

func TestClientProcess(t *testing.T) {
	type koSpec interface{}
	var (
		okRv exFields
		koRv koSpec
	)
	c := eve.New("test", server)
	if err := c.Envs("qa", "fr"); err != nil {
		t.Fatal(err)
	}
	if err := c.Process(&koRv); err != eve.ErrNoPointer {
		t.Fatalf("error mismatch: got=%q exp=%q", err, eve.ErrNoPointer)
	}
	if err := c.Process(okRv); err != eve.ErrNoPointer {
		t.Fatalf("error mismatch: got=%q exp=%q", err, eve.ErrNoPointer)
	}
	if err := c.Process(&okRv); err != nil {
		t.Fatal(err)
	}
}

func TestClientMustProcess(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic")
		}
	}()
	var okRv exFields
	c := eve.New("test", server)
	c.MustProcess(okRv)
}

func TestClientBool(t *testing.T) {
	c := eve.New("test", server)
	d, err := c.Bool("bool")
	if err != nil {
		t.Fatalf("expected no error: got=%v", err)
	}
	if !d {
		t.Errorf("content mismatch: got=%v exp=%v", d, boolVal)
	}
	if _, err = c.Bool("rv"); !reflect.DeepEqual(err, eve.ErrNotFound) {
		t.Errorf("error mismatch: got=%v exp=%v", err, eve.ErrNotFound)
	}
	if _, err = c.Bool("str"); !reflect.DeepEqual(err, eve.ErrInvalid) {
		t.Errorf("error mismatch: got=%v exp=%v", err, eve.ErrInvalid)
	}
}

func TestClientMustBool(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic")
		}
	}()
	c := eve.New("test", server)
	if d := c.MustBool("bool"); !d {
		t.Fatalf("content mismatch: got=%v exp=%v", d, boolVal)
	}
	_ = c.MustBool("rv")
}

func TestClientInt(t *testing.T) {
	c := eve.New("test", server)
	d, err := c.Int("int")
	if err != nil {
		t.Fatalf("expected no error: got=%v", err)
	}
	if d != intVal {
		t.Errorf("content mismatch: got=%v exp=%v", d, intVal)
	}
	if _, err = c.Int("rv"); !reflect.DeepEqual(err, eve.ErrNotFound) {
		t.Errorf("error mismatch: got=%v exp=%v", err, eve.ErrNotFound)
	}
	if _, err = c.Int("str"); !reflect.DeepEqual(err, eve.ErrInvalid) {
		t.Errorf("error mismatch: got=%v exp=%v", err, eve.ErrInvalid)
	}
}

func TestClientMustInt(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic")
		}
	}()
	c := eve.New("test", server)
	if d := c.MustInt("int"); d != intVal {
		t.Fatalf("content mismatch: got=%v exp=%v", d, intVal)
	}
	_ = c.MustInt("rv")
}

func TestClientFloat64(t *testing.T) {
	c := eve.New("test", server)
	d, err := c.Float64("float")
	if err != nil {
		t.Fatalf("expected no error: got=%v", err)
	}
	if d != floatVal {
		t.Errorf("content mismatch: got=%v exp=%v", d, floatVal)
	}
	if _, err = c.Float64("rv"); !reflect.DeepEqual(err, eve.ErrNotFound) {
		t.Errorf("error mismatch: got=%v exp=%v", err, eve.ErrNotFound)
	}
	if _, err = c.Float64("str"); !reflect.DeepEqual(err, eve.ErrInvalid) {
		t.Errorf("error mismatch: got=%v exp=%v", err, eve.ErrInvalid)
	}
}

func TestClientMustFloat64(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic")
		}
	}()
	c := eve.New("test", server)
	if d := c.MustFloat64("float"); d != floatVal {
		t.Fatalf("content mismatch: got=%v exp=%v", d, floatVal)
	}
	_ = c.MustFloat64("rv")
}

func TestClientString(t *testing.T) {
	c := eve.New("test", server)
	d, err := c.String("str")
	if err != nil {
		t.Fatalf("expected no error: got=%v", err)
	}
	if d != strVal {
		t.Errorf("content mismatch: got=%v exp=%v", d, strVal)
	}
	if _, err = c.String("rv"); !reflect.DeepEqual(err, eve.ErrNotFound) {
		t.Errorf("error mismatch: got=%v exp=%v", err, eve.ErrNotFound)
	}
	if _, err = c.String("int"); !reflect.DeepEqual(err, eve.ErrInvalid) {
		t.Errorf("error mismatch: got=%v exp=%v", err, eve.ErrInvalid)
	}
}

func TestClientMustString(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Errorf("expected panic")
		}
	}()
	c := eve.New("test", server)
	if d := c.MustString("str"); d != strVal {
		t.Fatalf("content mismatch: got=%v exp=%v", d, strVal)
	}
	_ = c.MustString("rv")
}

func TestClientGet(t *testing.T) {
	c := eve.New("test", server)
	if d := c.Get("rv"); d != nil {
		t.Fatalf("expected nil: got=%v", d)
	}
	var i interface{} = 42
	if d := c.Get("int"); d != i {
		t.Fatalf("content mismatch: got=%v exp=%v", d, i)
	}
}

func TestClientUseHandler(t *testing.T) {
	c := eve.New("test")
	if l := len(c.Handler); l != 2 {
		t.Fatalf("len mismatch: got=%v exp=%v", l, 2)
	}
	h := eve.Handler{}
	c.UseHandler(h)
	if l := len(c.Handler); l != 0 {
		t.Fatalf("len mismatch: got=%v exp=%v", l, 0)
	}
	h.AddHandler(server)
	if l := len(c.Handler); l != 1 {
		t.Fatalf("len mismatch: got=%v exp=%v", l, 1)
	}
	h.AddHandler(server)
	if l := len(c.Handler); l != 2 {
		t.Fatalf("len mismatch: got=%v exp=%v", l, 2)
	}
}

func TestWorkflow(t *testing.T) {
	// Prepares the workspace.
	eve.Tick = 50 * time.Millisecond
	// Tries with only one RPC cache as data source.
	h := eve.Handler{0: unsafeServer}
	c := eve.New("test").UseHandler(h)
	if i, err := c.Int("int"); err != nil {
		t.Fatalf("expected no error: got=%v", err)
	} else if i != intVal {
		t.Errorf("content mismatch: got=%v exp=%v", i, intVal)
	}
	time.Sleep(eve.Tick + 1)
	if _, err := c.Int("int"); reflect.DeepEqual(err, eve.ErrNotFound) {
		t.Fatalf("mismatch error: got=%q exp=%q", err, eve.ErrNotFound)
	}
	// Tries with local cache and one remote cache as handlers.
	h = eve.Handler{0: client.NewCache(100 * time.Millisecond), 1: unsafeServer}
	c = eve.New("test").UseHandler(h)
	if i, err := c.Int("int"); err != nil {
		t.Fatalf("expected no error: got=%v", err)
	} else if i != intVal {
		t.Errorf("content mismatch: got=%v exp=%v", i, intVal)
	}
	time.Sleep(eve.Tick + 1)
	if i, _ := c.Int("int"); i != intVal {
		t.Errorf("content mismatch: got=%v exp=%v", i, intVal)
	}
	time.Sleep(eve.Tick + 1)
	i, err := c.Int("int")
	if !reflect.DeepEqual(err, eve.ErrNotFound) {
		t.Fatalf("mismatch error: got=%q exp=%q", err, eve.ErrNotFound)
	}
	if i != 0 {
		t.Errorf("content mismatch: got=%v exp=%v", i, 0)
	}
}

func TestClientTooMuchEnvs(t *testing.T) {
	if err := eve.New("test").Envs("v1", "v2", "v3"); !reflect.DeepEqual(err, eve.ErrInvalid) {
		t.Fatalf("error mismatch: got=%q exp=%q", err, eve.ErrInvalid)
	}
}

func TestServers(t *testing.T) {
	var dt = []struct {
		addr    []string
		err     error
		partial bool
	}{
		{err: eve.ErrDataSource},
		{addr: []string{""}, err: errors.New(": dial tcp: missing address")},
	}

	var (
		err     error
		partial bool
	)
	for i, tt := range dt {
		_, err = eve.Servers(tt.addr...)
		if !equalErrs(err, tt.err) {
			t.Fatalf("%d. error mismatch: got=%q exp=%q", i, err, tt.err)
		}
		_, partial, err = eve.PartialServers(tt.addr...)
		if !equalErrs(err, tt.err) {
			t.Fatalf("%d. partial error mismatch: got=%q exp=%q", i, err, tt.err)
		}
		if partial != tt.partial {
			t.Errorf("%d. partial value mismatch: got=%t exp=%t", i, partial, tt.partial)
		}
	}
}

func equalErrs(e1, e2 error) bool {
	if e1 == nil {
		return e2 == nil
	}
	if e2 == nil {
		return e1 == nil
	}
	return e1.Error() == e2.Error()
}
