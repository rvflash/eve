// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package db

import (
	"net"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// Server represents one node with the status of the deployment on it.
type Server struct {
	TCPAddr   string `json:"naddr"`
	Succeeded bool   `json:"ok,omitempty"`
}

// Deploy represents one deployment.
type Deploy struct {
	ID           uint64                 `json:"id"`
	ProjectID    string                 `json:"project_id"`
	ServerList   []*Server              `json:"servers"`
	ItemList     map[string]interface{} `json:"items"`
	LastUpdateTs time.Time              `json:"upd_ts"`
}

// NewDeploy returns a new instance of Deploy.
func NewDeploy(projectID string, servers ...string) *Deploy {
	naddr := make([]*Server, len(servers))
	for i, server := range servers {
		naddr[i] = &Server{TCPAddr: server}
	}
	return &Deploy{ProjectID: projectID, ServerList: naddr}
}

// AutoIncrementing return true in order to have auo-increment primary key.
func (d *Deploy) AutoIncrementing() bool {
	return true
}

// Key returns the key of the env.
func (d *Deploy) Key() []byte {
	if d.ID == 0 {
		return nil
	}
	return itob(d.ID)
}

// SetKey returns if error if the change of the key failed.
func (d *Deploy) SetKey(k []byte) error {
	d.ID = btoi(k)
	return nil
}

// Updated changes the last update date of the deployment.
func (d *Deploy) Updated() {
	d.LastUpdateTs = time.Now()
}

// Valid checks if all required data as well formed.
func (d *Deploy) Valid(insert bool) error {
	if d.ProjectID = strings.TrimSpace(d.ProjectID); d.ProjectID == "" {
		return ErrInvalid
	}
	if len(d.ItemList) == 0 {
		return ErrMissing
	}
	if len(d.ServerList) == 0 {
		return ErrNotFound
	}
	for _, naddr := range d.ServerList {
		// Parses addr as a TCP address of the form "host:port".
		if _, err := net.ResolveTCPAddr("tcp", naddr.TCPAddr); err != nil {
			// Fails to resolve it.
			return errors.WithMessage(err, naddr.TCPAddr)
		}
	}
	if !insert && d.ID == 0 {
		return ErrOutOfBounds
	}
	return nil
}
