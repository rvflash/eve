package main

import (
	"flag"

	"github.com/rvflash/eve/db"
)

func main() {
	host := flag.String("host", "", "host addr to listen on")
	port := flag.Int("port", 8080, "service port")
	dsn := flag.String("dsn", "eve.db", "database's filepath")
	flag.Parse()

	// Try to connect to the local database.
	server := NewServer(*host, *port)
	if db, err := db.Open(*dsn); err != nil {
		server.log.Printf("fails to open the database: %s\n", err)
	} else {
		server.db = db
	}
	server.Route()
	server.Serve()
}
