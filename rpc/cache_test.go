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
	var i *rpc.Item
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
	i = &rpc.Item{Key: k, Value: v}
	if err := c.Put(i, &ok); err != nil {
		t.Fatalf("error mismatch: exp=nil got=%q", err)
	} else if !ok {
		t.Fatalf("ack mismatch: exp=true got=%v", ok)
	}
	// Applies various changes in one bulk.
	batch := make([]*rpc.Item, 4)
	batch[0] = &rpc.Item{Key: "r1", Value: "hi"}
	batch[1] = &rpc.Item{Key: "r2", Value: 3.14}
	batch[2] = &rpc.Item{Key: "r3", Value: true}
	batch[3] = &rpc.Item{Key: k}
	if err := c.Bulk(batch, &ok); err != nil {
		t.Fatalf("error mismatch: exp=nil got=%q", err)
	} else if !ok {
		t.Fatalf("ack mismatch: exp=true got=%v", ok)
	}
	for p := 0; p < 4; p++ {
		i = &rpc.Item{}
		if _ = c.Get(batch[p].Key, i); i.Value != batch[p].Value {
			t.Fatalf(
				"content mismatch for %q: exp=%v got=%v",
				batch[p].Key, batch[p].Value, i.Value,
			)
		}
	}
	// Adds a variable.
	i = &rpc.Item{Key: k, Value: v}
	if err := c.Put(i, &ok); err != nil {
		t.Fatalf("error mismatch: exp=nil got=%q", err)
	} else if !ok {
		t.Fatalf("ack mismatch: exp=true got=%v", ok)
	}
	// Retrieves its content.
	i = &rpc.Item{}
	if err := c.Get(k, i); err != nil {
		t.Fatalf("expected key found: got=%q", err)
	} else if k != i.Key {
		t.Fatalf("key mismatch: exp=%q got=%q", k, i.Key)
	} else if !i.Value.(bool) {
		t.Fatalf("value mismatch: exp=%v got=%v", v, i.Value)
	}
	// Deletes this variable.
	if err := c.Delete(k, &ok); err != nil {
		t.Fatalf("expected no error with deletion: got=%q", err)
	}
	// Adds a variable.
	i = &rpc.Item{Key: k, Value: v}
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
	req := &rpc.Metrics{}
	exp := rpc.Requests{Bulk: 1, Clear: 1, Delete: 1, Get: 4, Put: 3}
	if err := c.Stats(true, req); err != nil {
		t.Fatalf("error mismatch: exp=nil got=%q", err)
	} else if !reflect.DeepEqual(req.Requests, exp) {
		t.Fatalf("stats mismatch: exp=%v got=%v", exp, req.Requests)
	}
}
