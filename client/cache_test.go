// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package client_test

import (
	"testing"
	"time"

	"github.com/rvflash/eve/client"
)

func TestCacheWorkflow(t *testing.T) {
	d := time.Millisecond * 250
	c := client.NewCache(d)
	defer func() {
		if err := c.Close(); err != nil {
			t.Fatalf("expected no error on closing: got=%q", err)
		}
	}()
	k, v := "RV", true
	// Looks for an unknown variable.
	if _, ok := c.Lookup(k); ok {
		t.Fatal("expected key not found")
	}
	// Gets its value.
	if data := c.Get(k); data != nil {
		t.Fatalf("expected value not found: got=%v", data)
	}
	// Sets a variable.
	if err := c.Set(k, v); err != nil {
		t.Fatalf("expected no error on setting: got=%q", err)
	}
	// Looks for a known variable.
	if data, ok := c.Lookup(k); !ok {
		t.Fatal("expected key found")
	} else if b, ok := data.(bool); !ok {
		t.Fatalf("expected boolean data: got=%q", data)
	} else if b != v {
		t.Fatalf("content mismatch: exp=%q got=%q", v, b)
	}
	// Gets its value.
	if data := c.Get(k); data == nil {
		t.Fatal("expected data")
	} else if b, ok := data.(bool); !ok {
		t.Fatalf("expected boolean data: got=%q", data)
	} else if b != v {
		t.Fatalf("content mismatch: exp=%q got=%q", v, b)
	}
	// Break to stand by the cache's expiration.
	time.Sleep(d)
	// Gets the value of the deleted variable.
	if data := c.Get(k); data != nil {
		t.Fatalf("expected value not found: got=%v", data)
	}
	// Looks for the deleted variable.
	if _, ok := c.Lookup(k); ok {
		t.Fatal("expected key not found")
	}
	// Sets a variable.
	if err := c.Set(k, v); err != nil {
		t.Fatalf("expected no error on setting: got=%q", err)
	}
	// Break to stand by the limit of the cache expiration.
	time.Sleep(d)
	// Looks for the deleted variable.
	if _, ok := c.Lookup(k); ok {
		t.Fatal("expected key not found")
	}
	// Sets a variable.
	if err := c.Set(k, v); err != nil {
		t.Fatalf("expected no error on setting: got=%q", err)
	}
	// Break to stand by the cache's cleaning.
	time.Sleep(d * 2)
	// Looks for the deleted variable.
	if _, ok := c.Lookup(k); ok {
		t.Fatal("expected key not found")
	}
	// Sets a variable.
	if err := c.Set(k, v); err != nil {
		t.Fatalf("expected no error on setting: got=%q", err)
	}
	// Deletes the variable.
	if err := c.Delete(k); err != nil {
		t.Fatalf("expected no error on deletion: got=%q", err)
	}
	// Looks for the deleted variable.
	if _, ok := c.Lookup(k); ok {
		t.Fatal("expected key not found")
	}
	// Disables the purge.
	c.WithoutExpire()
	// Sets a variable.
	if err := c.Set(k, v); err != nil {
		t.Fatalf("expected no error on setting: got=%q", err)
	}
	// Break to stand by the limit of the cache expiration.
	time.Sleep(d)
	// Looks for the deleted variable.
	if _, ok := c.Lookup(k); !ok {
		t.Fatal("expected key found")
	}
	// Reactivates the item's expiration.
	c.WithExpire()
	if _, ok := c.Lookup(k); ok {
		t.Fatal("expected key not found")
	}
}
