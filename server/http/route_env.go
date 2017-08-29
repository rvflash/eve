package main

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/rvflash/eve/db"
)

// EnvsHandler listens post data to create a environment and go the project page.
func (s *Server) EnvsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.jsonHandler(w, "invalid method", http.StatusBadRequest)
		return
	}
	r.ParseForm()

	// We uses comma to split environment's values.
	f := func(r rune) bool {
		return r == ','
	}
	v := strings.FieldsFunc(r.Form.Get("vals"), f)

	// Tries to create a new environment.
	scp := db.NewEnv(r.Form.Get("name"), v)
	if err := s.db.AddEnv(scp); err != nil {
		s.jsonHandler(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Attaches this environment to a project.
	pid := r.Form.Get("pid")
	if pid == "" {
		// Redirects to the environment page (@todo).
		loc := fmt.Sprintf("/envs/%s/", scp.Key())
		s.jsonHandler(w, loc, http.StatusOK)
		return
	}
	if err := s.db.BindEnvInProject(scp, pid); err != nil {
		s.jsonHandler(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Redirects to the project page.
	loc := fmt.Sprintf("/projects/%s/", pid)
	s.jsonHandler(w, loc, http.StatusOK)
}
