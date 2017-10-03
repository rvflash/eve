// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package eve_test

import (
	"github.com/rvflash/eve"
)

// cli is the test client to fake RPC client.
type cli struct{}

// Get implements the client.Getter interface.
func (c cli) Get(key string) interface{} {
	return nil
}

// Lookup implements the client.Getter interface.
func (c cli) Lookup(key string) (interface{}, bool) {
	return nil, false
}

func ExampleNew() {
	env := eve.New("test")
	env.Lookup("coucou")
}
