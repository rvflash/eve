// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package rpc_test

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/rvflash/eve/rpc"
)

var errNoTransport = errors.New("no transport")

type fakeHttpClient struct{}

func (c *fakeHttpClient) Get(url string) (*http.Response, error) {
	if !strings.HasPrefix(url, "http") {
		return nil, errNoTransport
	}
	// Mocks responses base on the URL.
	urlHandler := func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/vars":
			_, _ = io.WriteString(w, `{"ALPHA_BOOL":true,"ALPHA_STR":"2ojE41"}`)
		case "/oops":
			_, _ = io.WriteString(w, `{"ALPHA_BOOL"`)
		default:
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{}`)
		}
	}
	req := httptest.NewRequest("GET", url, nil)
	w := httptest.NewRecorder()
	urlHandler(w, req)
	return w.Result(), nil
}

func TestNewFrom(t *testing.T) {
	var ct = &fakeHttpClient{}
	var dt = []struct {
		from   string
		client rpc.Getter
		stats  rpc.Requests
		onErr  bool
	}{
		{from: "/", client: ct, onErr: true},
		{from: "http://localhost:8080/oops", client: ct, onErr: true},
		{from: "http://localhost:8080", client: ct, onErr: true},
		{from: "http://localhost:8080/vars", client: ct, stats: rpc.Requests{Put: 2}},
	}
	for i, tt := range dt {
		c, err := rpc.NewFrom(tt.from, tt.client)
		if tt.onErr != (err != nil) {
			t.Fatalf("%d. error mismatch: error expexted=%q got=%q", i, tt.onErr, err)
		}
		if err == nil {
			req := &rpc.Metrics{}
			_ = c.Stats(true, req)
			if !reflect.DeepEqual(req.Requests, tt.stats) {
				t.Errorf("%d. stats mismatch: exp=%v got=%v", i, tt.stats, req.Requests)
			}
		}
	}
}

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
