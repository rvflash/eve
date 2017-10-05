// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package client

import "errors"

// List of value's kind.
const (
	BoolVal int = 1 << iota
	FloatVal
	IntVal
	StringVal
)

// Caller must be implemented by any client to call a service,
// waits for it to complete, and returns its error status.
type Caller interface {
	Call(service string, args, reply interface{}) error
}

// Asserter must be implemented by any client
// that needs to assert it values.
type Asserter interface {
	Assert(value interface{}, kind int) (interface{}, bool)
}

// Reader must be implemented by any client to get data.
type Getter interface {
	Lookup(key string) (interface{}, bool)
	NeedAssert() bool
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
