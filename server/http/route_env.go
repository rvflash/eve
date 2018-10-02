// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	"github.com/rvflash/eve/db"
)

// EnvHandler catches the routes to get, update or delete an env.
// With the project ID, it also can unbind an env on it.
func (s *Server) EnvHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// Tries to retrieve this environment.
	eid, _ := strconv.ParseUint(vars["eid"], 10, 64)
	d, err := s.db.GetEnv(eid)
	if err != nil {
		s.jsonHandler(w, "environment not found", http.StatusNotFound)
		return
	}
	// Routes the request.
	env := d.(*db.Env)
	pid, ok := vars["pid"]
	if !ok {
		if r.Method == http.MethodPost {
			// Updates this environment.
			scp := parseEnv(r)
			env.Name, env.Values = scp.Name, scp.Values
			if err := s.db.UpsertEnv(env); err != nil {
				s.jsonHandler(w, err.Error(), http.StatusBadRequest)
			} else {
				s.jsonHandler(w, "", http.StatusOK)
			}
		} else {
			// Gets its properties.
			buf, err := json.Marshal(env)
			if err != nil {
				s.jsonHandler(w, err.Error(), http.StatusBadRequest)
			} else {
				// Displays a JSON representation of this env.
				s.jsonAppHandler(w, buf)
			}
		}
	} else {
		loc := fmt.Sprintf("/project/%s/", pid)
		switch do := vars["do"]; do {
		case "bind":
			if err := s.db.BindEnvInProject(env, pid); err != nil {
				s.jsonHandler(w, err.Error(), http.StatusBadRequest)
			} else {
				s.jsonHandler(w, loc, http.StatusOK)
			}
		case "unbind":
			_ = s.db.UnbindEnvInProject(env, pid)
			http.Redirect(w, r, loc, http.StatusFound)
		default:
			s.NotFoundHandler(w, r)
		}
	}
}

// EnvsHandler listens post data to create a environment and go the project page.
func (s *Server) EnvsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		s.jsonHandler(w, "invalid method", http.StatusBadRequest)
		return
	}
	// Tries to create a new environment.
	env := parseEnv(r)
	if err := s.db.AddEnv(env); err != nil {
		s.jsonHandler(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Attaches this environment to a project.
	pid := mux.Vars(r)["pid"]
	if err := s.db.BindEnvInProject(env, pid); err != nil {
		s.jsonHandler(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Redirects to the project page.
	loc := fmt.Sprintf("/project/%s/", pid)
	s.jsonHandler(w, loc, http.StatusOK)
}

func parseEnv(r *http.Request) *db.Env {
	_ = r.ParseForm()

	// We uses comma to split environment's values.
	f := func(r rune) bool {
		return r == ','
	}
	v := strings.FieldsFunc(r.Form.Get("vals"), f)

	return db.NewEnv(r.Form.Get("name"), v)
}
