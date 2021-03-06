// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package main

import (
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"github.com/rvflash/eve/db"
)

// Server represents the default server's configuration.
type Server struct {
	Host string
	Port int
	db   *db.Data
	log  *log.Logger
	r    *mux.Router
}

// NewServer returns an instance of Server.
func NewServer(listenIP string, httpPort int) *Server {
	return &Server{
		Host: listenIP,
		Port: httpPort,
		log:  log.New(os.Stdout, "server> ", log.Ltime|log.Lshortfile),
		r:    mux.NewRouter(),
	}
}

// Route defines all routes to listen and serve.
func (s *Server) Route() {
	switch s.db {
	case nil:
		// No connection to the database to share.
		s.r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			s.OopsHandler(w, r, errors.WithMessage(db.ErrMissing, "database"))
		})
	default:
		s.r.HandleFunc("/", s.HomeHandler)
		s.r.HandleFunc("/node", s.NodesHandler)
		s.r.HandleFunc("/node/{naddr}/delete", s.NodeHandler)
		s.r.HandleFunc("/env/{eid:[0-9]+}/", s.EnvHandler)
		s.r.HandleFunc("/project", s.ProjectsHandler)
		s.r.HandleFunc("/project/{pid:[a-z-]+}/", s.ProjectHandler)
		s.r.HandleFunc("/project/{pid:[a-z-]+}/env", s.EnvsHandler)
		s.r.HandleFunc("/project/{pid:[a-z-]+}/env/{eid:[0-9]+}/{do}", s.EnvHandler)
		s.r.HandleFunc("/project/{pid:[a-z-]+}/var", s.VarsHandler)
		s.r.HandleFunc("/project/{pid:[a-z-]+}/var/{vid:[0-9]+}", s.VarHandler)
		s.r.HandleFunc("/project/{pid:[a-z-]+}/var/{vid:[0-9]+}/delete", s.VarHandler)
		s.r.HandleFunc("/project/{pid:[a-z-]+}/deploy", s.DeployHandler)
		s.r.HandleFunc("/vars", s.CacheHandler)
		s.r.HandleFunc("/favicon.ico", s.StaticHandler)
	}
}

// Serve starts the server.
func (s *Server) Serve() {
	defer func() {
		// Close the connection to the database.
		if s.db != nil {
			if err := s.db.Close(); err != nil {
				s.log.Printf("fails to close the database: %s\n", err)
			}
		}
	}()
	addr := s.Host + ":" + strconv.Itoa(s.Port)
	s.log.Println("Serving " + addr)
	s.log.Fatal(http.ListenAndServe(addr, s.r))
}
