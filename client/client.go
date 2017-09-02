// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package client

import "errors"

// Reader must be implemented by any client to get data.
type Getter interface {
	Get(key string) interface{}
	Lookup(key string) (interface{}, bool)
}

// Writer must be implemented by any client to set data.
type Setter interface {
	Set(key string, value interface{}) error
}

// ReadWriter must be implemented by any client to get and set data.
type GetSetter interface {
	Getter
	Setter
}

// Error messages.
var (
	ErrFailure = errors.New("request has failed")
	ErrKind    = errors.New("invalid data type")
)
