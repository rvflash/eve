// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package client

import "os"

// OS is the client to get environment variable from operating system.
type OS struct{}

// Get retrieves the value of the environment variable named by the key.
func (o *OS) Get(key string) interface{} {
	return os.Getenv(key)
}

// Lookup gets the value of the environment variable named by the key.
// If the variable is present in the environment, the value (which may be empty)
// is returned and the boolean is true.
// Otherwise the returned value will be empty and the boolean will be false.
func (o *OS) Lookup(key string) (interface{}, bool) {
	return os.LookupEnv(key)
}

// Set sets the value of the environment variable named by the key.
func (o *OS) Set(key string, value interface{}) error {
	s, ok := value.(string)
	if !ok {
		return ErrKind
	}
	return os.Setenv(key, s)
}
