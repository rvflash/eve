// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package main

import (
	"flag"
	"runtime"

	"github.com/rvflash/eve/rpc"
)

func init() {
	runtime.GOMAXPROCS(runtime.NumCPU())
}

func main() {
	host := flag.String("host", "", "host addr to listen on")
	port := flag.Int("port", rpc.DefaultPort, "service port")
	from := flag.String("from", "", "URL to fetch to get JSON data to use as default values")
	flag.Parse()

	// Try to connect to the local database.
	NewServer(*host, *port).Serve(*from)
}
