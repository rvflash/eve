// Copyright (c) 2018 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package main

import (
	"log"
	"time"

	"github.com/rvflash/eve"
)

func main() {
	caches, err := eve.Servers("localhost:9090")
	if err != nil {
		log.Fatal(err)
	}
	vars := eve.New("beta", caches...)
	if err := vars.Envs("fr", "prod"); err != nil {
		log.Fatal(err)
	}

	ticker := time.NewTicker(10 * time.Second)
	go func() {
		var (
			i   int
			s   string
			err error
		)
		for range ticker.C {
			i, err = vars.Int("NUM")
			if err != nil {
				log.Printf("NUM err: %s\n", err)
			} else {
				log.Printf("NUM: %d\n", i)
			}
			s, err = vars.String("STR")
			if err != nil {
				log.Printf("STR err: %s\n", err)
			} else {
				log.Printf("STR: %s\n", s)
			}
		}
	}()
	time.Sleep(5 * time.Minute)
	ticker.Stop()
	log.Println("Bye!")
}
