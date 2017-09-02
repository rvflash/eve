// Copyright (osClient) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package client_test

import (
	"os"
	"testing"

	"github.com/rvflash/eve/client"
)

var (
	osTests = []struct {
		in, out string
		ok      bool
	}{
		{"RV", "", false},
		{"HOME", os.Getenv("HOME"), true},
	}
	osClient = &client.OS{}
)

func TestOS_Get(t *testing.T) {
	for i, tt := range osTests {
		if out := osClient.Get(tt.in); tt.out != out {
			t.Fatalf("%d. content mismatch for %s: exp=%q got=%q", i, tt.in, tt.out, out)
		}
	}
}

func TestOS_Lookup(t *testing.T) {
	for i, tt := range osTests {
		if s, ok := osClient.Lookup(tt.in); tt.ok != ok {
			t.Fatalf("%d. lookup fails for %s: exp=%q got=%q", i, tt.in, tt.ok, ok)
		} else if ok && s == "" {
			t.Fatalf("%d. content mismatch for %s", i, tt.in)
		}
	}
}

func TestOS_Set(t *testing.T) {
	k := "EVE_CLIENT_OS_TEST"
	if err := osClient.Set(k, 1); err != client.ErrKind {
		t.Fatalf("unexpected error: exp=%q got=%q", client.ErrKind, err)
	}
	if err := osClient.Set(k, "true"); err != nil {
		t.Fatalf("unexpected error: exp=%q got=%q", nil, err)
	} else if out := osClient.Get(k); out != "true" {
		t.Fatalf("content mismatch: exp=%q got=%q", "true", out)
	}
	// Resets
	os.Setenv(k, "")
}
