// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package client

import (
	"testing"
	"time"
)

func TestCacheItemExpired(t *testing.T) {
	var dt = []struct {
		in  *cacheItem
		out bool
	}{
		{&cacheItem{}, false},
		{&cacheItem{expires: time.Now().Add(-time.Minute)}, true},
	}
	for i, tt := range dt {
		if out := tt.in.expired(); tt.out != out {
			t.Errorf("%d. result mismatch: exp=%q got=%q", i, tt.out, out)
		}
	}
}
