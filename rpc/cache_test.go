// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package rpc_test

import (
	"reflect"
	"testing"

	"github.com/rvflash/eve/rpc"
)

func TestWorkflow(t *testing.T) {
	// Creates the workspace.
	k, v := "RV", true
	// Creates a new instance of the cache.
	c := rpc.New()
	// Gets an unknown variable.
	var i *rpc.CacheItem
	if err := c.Get(k, i); err != rpc.ErrNotFound {
		t.Fatalf("expected key not found: got=%q", err)
	}
	// Deletes an unknown variable.
	var ok bool
	if err := c.Delete(k, &ok); err == nil {
		t.Fatalf("expected key not found: got=%q", err)
	} else if ok {
		t.Fatalf("ack mismatch: exp=false got=%v", ok)
	}
	// Adds a variable.
	i = &rpc.CacheItem{Key: k, Value: v}
	if err := c.Put(i, &ok); err != nil {
		t.Fatalf("error mismatch: exp=nil got=%q", err)
	} else if !ok {
		t.Fatalf("ack mismatch: exp=true got=%v", ok)
	}
	// Retrieves its content.
	i = &rpc.CacheItem{}
	if err := c.Get(k, i); err != nil {
		t.Fatalf("expected key found: got=%q", err)
	} else if k != i.Key {
		t.Fatalf("key mismatch: exp=%q got=%q", k, i.Key)
	} else if !i.Value.(bool) {
		t.Fatalf("value mismatch: exp=%v got=%v", v, i.Value)
	}
	// Deletes this variable.
	ok = false
	if err := c.Delete(k, &ok); err != nil {
		t.Fatalf("expected no error with deletion: got=%q", err)
	}
	// Adds a variable.
	i = &rpc.CacheItem{Key: k, Value: v}
	if err := c.Put(i, &ok); err != nil {
		t.Fatalf("error mismatch: exp=nil got=%q", err)
	} else if !ok {
		t.Fatalf("ack mismatch: exp=true got=%v", ok)
	}
	// Resets the cache.
	if err := c.Clear(true, &ok); err != nil {
		t.Fatalf("error mismatch: exp=nil got=%q", err)
	}
	// Retrieves the statistics.
	req := &rpc.Requests{}
	exp := &rpc.Requests{Get: 1, Put: 2, Delete: 1, Clear: 1}
	if err := c.Stats(true, req); err != nil {
		t.Fatalf("error mismatch: exp=nil got=%q", err)
	} else if !reflect.DeepEqual(exp, req) {
		t.Fatalf("stats mismatch: exp=%v got=%v", exp, req)
	}
}
