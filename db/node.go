// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package db

import (
	"net"
	"strings"
	"time"
)

// Node represents a server used as cache by EVE.
type Node struct {
	Addr  string    `json:"naddr"`
	AddTs time.Time `json:"upd_ts"`
}

// NewNode creates a new instance of a server node.
func NewNode(server string) *Node {
	return &Node{Addr: server}
}

// AutoIncrementing return true in order to have auo-increment primary key.
func (c *Node) AutoIncrementing() bool {
	return false
}

// ID returns the key of the server node as string
func (c *Node) ID() string {
	return string(c.Key())
}

// Key returns the key of the cache.
func (c *Node) Key() []byte {
	return []byte(c.Addr)
}

// SetKey returns if error if the change of the key failed.
func (c *Node) SetKey(k []byte) error {
	c.Addr = string(k)
	return nil
}

// Updated must be implemented for the Valuable interface.
func (c *Node) Updated() {
	c.AddTs = time.Now()
}

// Valid checks if all required data as well formed.
func (c *Node) Valid(insert bool) error {
	c.Addr = strings.TrimSpace(c.Addr)
	if c.Addr == "" {
		return ErrInvalid
	}
	// Tries to resolve the server names as TCP address.
	_, err := net.ResolveTCPAddr("tcp", c.Addr)
	return err
}
