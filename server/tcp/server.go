// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package main

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"strconv"

	cache "github.com/rvflash/eve/rpc"
)

// Server represents the default server's configuration.
type Server struct {
	Host string
	Port int
	log  *log.Logger
	rpc  *cache.Cache
}

// NewServer returns an instance of Server.
func NewServer(listenIP string, httpPort int) *Server {
	return &Server{
		Host: listenIP,
		Port: httpPort,
		log:  log.New(os.Stdout, "server> ", log.Ltime|log.Lshortfile),
		rpc:  cache.New(),
	}
}

// Serve starts the server.
func (s *Server) Serve() {
	// Prepares the launching.
	addr := s.Host + ":" + strconv.Itoa(s.Port)
	s.log.Println("Serving " + addr)

	// Launches it.
	rpc.Register(s.rpc)
	rpc.HandleHTTP()
	l, err := net.Listen("tcp", addr)
	if err != nil {
		s.log.Fatal("Listen error:", err)
	}
	go http.Serve(l, nil)
}
