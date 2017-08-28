package db

import (
	"strings"
	"time"
)

// DefaultEnv is the default Environment used to build the variables's values.
var DefaultEnv = &Environment{Values: []string{""}}

// Environment represents a env of execution.
type Environment struct {
	ID           uint64    `json:"id"`
	Name         string    `json:"name"`
	Values       []string  `json:"vals"`
	LastUpdateTs time.Time `json:"upd_ts"`
}

// NewEnv returns a new instance of Environment.
func NewEnv(name string, values []string) *Environment {
	return &Environment{Name: name, Values: values}
}

// AutoIncrementing return true in order to have auo-increment primary key.
func (s *Environment) AutoIncrementing() bool {
	return true
}

// Key returns the key of the env.
func (s *Environment) Key() []byte {
	if s.ID == 0 {
		return nil
	}
	return itob(s.ID)
}

// SetKey returns if error if the change of the key failed.
func (s *Environment) SetKey(k []byte) error {
	s.ID = btoi(k)
	return nil
}

// Updated changes the last update date of the var.
func (s *Environment) Updated() {
	s.LastUpdateTs = time.Now()
}

// Valid checks if all required data as well formed.
func (s *Environment) Valid(insert bool) error {
	s.Name = strings.TrimSpace(s.Name)
	if !check(s.Name) {
		return ErrInvalid
	}
	s.Values = unique(s.Values)
	if len(s.Values) == 0 {
		return ErrMissing
	}
	if !insert && s.ID == 0 {
		return ErrOutOfBounds
	}
	return nil
}
