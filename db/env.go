// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package db

import (
	"strings"
	"time"
)

// DefaultEnv is the default Environment used to build the variables's values.
var DefaultEnv = &Env{Values: []string{""}}

// Environment represents a env of execution.
type Env struct {
	ID           uint64    `json:"id"`
	Name         string    `json:"name"`
	Values       []string  `json:"vals"`
	LastUpdateTs time.Time `json:"upd_ts"`
}

// NewEnv returns a new instance of Environment.
func NewEnv(name string, values []string) *Env {
	return &Env{Name: name, Values: values}
}

// AutoIncrementing return true in order to have auo-increment primary key.
func (s *Env) AutoIncrementing() bool {
	return true
}

// Default returns true if the environment is a default env for Eve.
func (s *Env) Default() bool {
	if len(s.Values) == 1 && s.Values[0] == "" {
		return true
	}
	return false
}

// Key returns the key of the env.
func (s *Env) Key() []byte {
	if s.ID == 0 {
		return nil
	}
	return itob(s.ID)
}

// SetKey returns if error if the change of the key failed.
func (s *Env) SetKey(k []byte) error {
	s.ID = btoi(k)
	return nil
}

// Updated changes the last update date of the environment.
func (s *Env) Updated() {
	s.LastUpdateTs = time.Now()
}

// Valid checks if all required data as well formed.
func (s *Env) Valid(insert bool) error {
	s.Name = strings.TrimSpace(s.Name)
	if !check(s.Name) {
		return ErrInvalid
	}
	s.Values = unique(s.Values)
	if !checklist(s.Values) {
		return ErrMissing
	}
	if !insert && s.ID == 0 {
		return ErrOutOfBounds
	}
	return nil
}
