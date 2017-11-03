// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package main

import (
	"log"
	"net"
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
func NewServer(listenIP string, port int) *Server {
	return &Server{
		Host: listenIP,
		Port: port,
		log:  log.New(os.Stdout, "server> ", log.Ltime|log.Lshortfile),
		rpc:  cache.New(),
	}
}

// Serve starts the server.
func (s *Server) Serve(fromUrl string) {
	// Uses this URL as JSON data source on loading.
	if fromUrl != "" {
		var err error
		if s.rpc, err = cache.NewFrom(fromUrl); err != nil {
			log.Fatal("Loader in error: ", err)
		}
	}
	// Initializes the RPC server.
	if err := rpc.Register(s.rpc); err != nil {
		log.Fatal("Register error: ", err)
	}
	// Launches it.
	addr := s.Host + ":" + strconv.Itoa(s.Port)
	s.log.Println("Serving " + addr)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		s.log.Fatal("Listen error: ", err)
	}
	rpc.Accept(l)
}
