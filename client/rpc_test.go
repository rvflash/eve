// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package client_test

import (
	"errors"
	"testing"
	"time"

	"reflect"

	"github.com/rvflash/eve/client"
	cache "github.com/rvflash/eve/rpc"
)

// rpc is the test's RPC client.
type rpc struct{}

// Call implements the client.Caller interface
func (c *rpc) Call(service string, args, reply interface{}) error {
	switch service {
	case "Cache.Bulk":
		items := args.([]*cache.Item)
		if items[0].Key == "bool" {
			*reply.(*bool) = true
		}
		return nil
	case "Cache.Clear":
		*reply.(*bool) = true
		return nil
	case "Cache.Delete":
		switch args {
		case "err":
			return cache.ErrNotFound
		case "bool":
			*reply.(*bool) = true
		}
		return nil
	case "Cache.Get":
		if args == "bool" {
			reply.(*cache.Item).Value = true
			return nil
		}
		return cache.ErrNotFound
	case "Cache.Put":
		item := args.(*cache.Item)
		switch item.Key {
		case "err":
			return cache.ErrNotFound
		case "bool":
			*reply.(*bool) = true
		}
		return nil
	case "Cache.Stats":
		return nil
	}
	return errors.New("unknown service")
}

var c = client.NewRPC(&rpc{})

// TestOpenRPC is a basic test for OpenRPC.
func TestOpenRPC(t *testing.T) {
	if _, err := client.OpenRPC(":0007", time.Second); err == nil {
		t.Fatal("expected error with open RPC")
	}
}

// TestRPC_Get tests all getter.
func TestRPC_Get(t *testing.T) {
	var dt = []struct {
		in     string
		out    interface{}
		err    error
		exists bool
	}{
		{in: "bool", out: true, exists: true},
		{in: "nil", err: cache.ErrNotFound},
		{in: "err", err: cache.ErrNotFound},
	}
	var exists bool
	for i, tt := range dt {
		out, err := c.Raw(tt.in)
		if !reflect.DeepEqual(err, tt.err) {
			t.Fatalf("%d. error mismatch for %q: got=%q exp=%q", i, tt.in, err, tt.err)
		}
		if out != tt.out {
			t.Errorf("%d. raw content mismatch for %q: got=%q exp=%q", i, tt.in, out, tt.out)
		}
		if out = c.Get(tt.in); out != tt.out {
			t.Errorf("%d. content mismatch for %q: got=%q exp=%q", i, tt.in, out, tt.out)
		}
		out, exists = c.Lookup(tt.in)
		if exists != tt.exists {
			t.Errorf("%d. exists mismatch for %q: got=%q exp=%q", i, tt.in, exists, tt.exists)
		}
		if out != tt.out {
			t.Errorf("%d. lookup content mismatch for %q: got=%q exp=%q", i, tt.in, out, tt.out)
		}
	}
}

// TestRPC_Get tests the delete method.
func TestRPC_Delete(t *testing.T) {
	var dt = []struct {
		in  string
		err error
	}{
		{in: "bool"},
		{in: "nil", err: client.ErrFailure},
		{in: "err", err: cache.ErrNotFound},
	}
	for i, tt := range dt {
		if err := c.Delete(tt.in); !reflect.DeepEqual(err, tt.err) {
			t.Fatalf("%d. error mismatch for %q: got=%q exp=%q", i, tt.in, err, tt.err)
		}
	}
}

// TestRPC_Clear tests the clear method.
func TestRPC_Clear(t *testing.T) {
	if err := c.Clear(); err != nil {
		t.Fatalf("unexpected error: got=%q", err)
	}
}

// TestRPC_Set tests the setter.
func TestRPC_Set(t *testing.T) {
	var dt = []struct {
		key   string
		value interface{}
		err   error
	}{
		{key: "bool", value: true},
		{key: "nil", err: client.ErrFailure},
		{key: "err", err: cache.ErrNotFound},
	}
	for i, tt := range dt {
		if err := c.Set(tt.key, tt.value); !reflect.DeepEqual(err, tt.err) {
			t.Fatalf("%d. error mismatch for %q: got=%q exp=%q", i, tt.key, err, tt.err)
		}
	}
}

// TestRPC_Stats tests the stats method.
func TestRPC_Stats(t *testing.T) {
	if _, err := c.Stats(); err != nil {
		t.Fatalf("unexpected error: got=%q", err)
	}
}

// TestRPC_Bulk tests the bulk method.
func TestRPC_Bulk(t *testing.T) {
	var dt = []struct {
		batch map[string]interface{}
		err   error
	}{
		{},
		{batch: map[string]interface{}{"bool": true}},
		{batch: map[string]interface{}{"err": nil}, err: client.ErrFailure},
	}
	for i, tt := range dt {
		if err := c.Bulk(tt.batch); !reflect.DeepEqual(err, tt.err) {
			t.Fatalf("%d. error mismatch: got=%q exp=%q", i, err, tt.err)
		}
	}
}