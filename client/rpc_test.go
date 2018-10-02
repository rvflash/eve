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

var (
	dataBool = "bool"
	dataErr  = "err"
	dataNil  = "nil"
)

// rpc is the test's RPC client.
type rpc struct{}

// Call implements the client.Caller interface
func (c *rpc) Call(service string, args, reply interface{}) error {
	switch service {
	case "Cache.Bulk":
		items := args.([]*cache.Item)
		if items[0].Key == dataBool {
			*reply.(*bool) = true
		}
		return nil
	case "Cache.Clear":
		*reply.(*bool) = true
		return nil
	case "Cache.Delete":
		switch args {
		case dataErr:
			return cache.ErrNotFound
		case dataBool:
			*reply.(*bool) = true
		}
		return nil
	case "Cache.Get":
		if args == dataBool {
			reply.(*cache.Item).Value = true
			return nil
		}
		return cache.ErrNotFound
	case "Cache.Put":
		item := args.(*cache.Item)
		switch item.Key {
		case dataErr:
			return cache.ErrNotFound
		case dataBool:
			*reply.(*bool) = true
		}
		return nil
	case "Cache.Stats":
		return nil
	}
	return errors.New("unknown service")
}

var c = client.NewRPC(&rpc{})

func TestOpenRPC(t *testing.T) {
	if _, err := client.OpenRPC(":0007", time.Second); err == nil {
		t.Fatal("expected error with open RPC")
	}
}

func TestRPCGet(t *testing.T) {
	var dt = []struct {
		in     string
		out    interface{}
		err    error
		exists bool
	}{
		{in: dataBool, out: true, exists: true},
		{in: dataNil, err: cache.ErrNotFound},
		{in: dataErr, err: cache.ErrNotFound},
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
			t.Errorf("%d. exists mismatch for %q: got=%t exp=%t", i, tt.in, exists, tt.exists)
		}
		if out != tt.out {
			t.Errorf("%d. lookup content mismatch for %q: got=%q exp=%q", i, tt.in, out, tt.out)
		}
	}
}

func TestRPCDelete(t *testing.T) {
	var dt = []struct {
		in  string
		err error
	}{
		{in: dataBool},
		{in: dataNil, err: client.ErrFailure},
		{in: dataErr, err: cache.ErrNotFound},
	}
	for i, tt := range dt {
		if err := c.Delete(tt.in); !reflect.DeepEqual(err, tt.err) {
			t.Fatalf("%d. error mismatch for %q: got=%q exp=%q", i, tt.in, err, tt.err)
		}
	}
}

func TestRPCClear(t *testing.T) {
	if err := c.Clear(); err != nil {
		t.Fatalf("unexpected error: got=%q", err)
	}
}

func TestRPCSet(t *testing.T) {
	var dt = []struct {
		key   string
		value interface{}
		err   error
	}{
		{key: dataBool, value: true},
		{key: dataNil, err: client.ErrFailure},
		{key: dataErr, err: cache.ErrNotFound},
	}
	for i, tt := range dt {
		if err := c.Set(tt.key, tt.value); !reflect.DeepEqual(err, tt.err) {
			t.Fatalf("%d. error mismatch for %q: got=%q exp=%q", i, tt.key, err, tt.err)
		}
	}
}

func TestRPCStats(t *testing.T) {
	if _, err := c.Stats(); err != nil {
		t.Fatalf("unexpected error: got=%q", err)
	}
}

func TestRPCAvailable(t *testing.T) {
	if ok := c.Available(); !ok {
		t.Fatal("expected the cache as available")
	}
}

func TestRPCBulk(t *testing.T) {
	var dt = []struct {
		batch map[string]interface{}
		err   error
	}{
		{},
		{batch: map[string]interface{}{dataBool: true}},
		{batch: map[string]interface{}{dataErr: nil}, err: client.ErrFailure},
	}
	for i, tt := range dt {
		if err := c.Bulk(tt.batch); !reflect.DeepEqual(err, tt.err) {
			t.Fatalf("%d. error mismatch: got=%q exp=%q", i, err, tt.err)
		}
	}
}
