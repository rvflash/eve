// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package main

import (
	"encoding/json"
	"errors"
	"html/template"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/rvflash/eve/client"
	"github.com/rvflash/eve/db"
	"github.com/rvflash/eve/deploy"
)

type deployTmplVars struct {
	projectTmplVars
	Step    int
	Release *deploy.Release
	Err     error
}

// CacheHandler prints a JSON string with all vars to expose.
func (s *Server) CacheHandler(w http.ResponseWriter, r *http.Request) {
	// Parses the dedicated directory to retrieve all projects vars.
	fs, err := ioutil.ReadDir(varsPath)
	if err != nil {
		s.jsonHandler(w, err.Error(), http.StatusBadRequest)
	}
	// Reads the JSON file to get map of vars as key/value.
	readJSON := func(filePath string) map[string]interface{} {
		raw, err := ioutil.ReadFile(filePath)
		if err != nil {
			return nil
		}
		var d map[string]interface{}
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil
		}
		return d
	}
	var d map[string]interface{}
	for _, f := range fs {
		for k, v := range readJSON(f.Name()) {
			d[k] = v
		}
	}
	// Prints in one JSON string all of them.
	if len(d) == 0 {
		s.jsonAppHandler(w, []byte("{}"))
		return
	}
	var raw []byte
	if raw, err = json.Marshal(d); err != nil {
		s.jsonHandler(w, err.Error(), http.StatusBadRequest)
	}
	s.jsonAppHandler(w, raw)
}

// NodeHandler deletes a server node.
func (s *Server) NodeHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	// Tries to delete this server's address.
	n := db.NewNode(vars["naddr"])
	if err := s.db.DeleteNode(n); err != nil {
		s.jsonHandler(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.jsonHandler(w, "", http.StatusOK)
}

// NodesHandler enables to create a node.
func (s *Server) NodesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.jsonHandler(w, "invalid method", http.StatusBadRequest)
		return
	}
	r.ParseForm()

	// Tries to add this server's address.
	n := db.NewNode(r.Form.Get("naddr"))
	if err := s.db.AddNode(n); err != nil {
		s.jsonHandler(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.jsonHandler(w, "", http.StatusOK)
}

// DeployHandler allows to deploy a project.
func (s *Server) DeployHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieves data to deploy.
	vars := mux.Vars(r)
	p, err := s.db.GetProject(vars["pid"])
	if err != nil {
		s.NotFoundHandler(w, r)
		return
	}

	// Builds the page.
	var t *template.Template
	t, err = template.New("deploy.html").Funcs(tmplFuncMap).ParseFiles(
		tmplPath+"/deploy.html",
		tmplPath+"/common/form.html",
		tmplPath+"/common/node.html",
		tmplPath+"/common/header.html",
		tmplPath+"/common/head.html",
		tmplPath+"/common/foot.html",
		tmplPath+"/common/footer.html",
	)
	if err != nil {
		s.OopsHandler(w, r, err)
		return
	}

	// Assigns vars to the templates.
	tv := deployTmplVars{}
	tv.Title = p.(*db.Project).Name
	tv.Href = "/project/" + vars["pid"] + "/"
	tv.Project = p
	if tv.Servers, tv.Err = s.nodes(); tv.Err == nil {
		tv.Step, tv.Release, tv.Err = s.deploy(p, tv.Servers, r)
	}
	// Displays the page.
	if err = t.Execute(w, tv); err != nil {
		s.OopsHandler(w, r, err)
	}
}

func (s *Server) deploy(p db.Keyer, w []db.Keyer, r *http.Request) (
	step int, out *deploy.Release, err error,
) {
	toMap := func(s []string) map[string]struct{} {
		m := make(map[string]struct{}, len(s))
		for _, v := range s {
			m[v] = struct{}{}
		}
		return m
	}
	r.ParseForm()

	// Checks the project envs to bypass the checkout page.
	project := p.(*db.Project)
	if project.FirstEnv().Default() {
		r.Form["ev1"] = []string{""}
	}
	if project.SecondEnv().Default() {
		r.Form["ev2"] = []string{""}
	}
	// Nothing to checkout
	if len(r.Form["ev1"]) == 0 || len(r.Form["ev2"]) == 0 {
		return
	}
	// Gets all env values of the project.
	eev1, eev2 := project.EnvsValues()
	cev1 := toMap(eev1)
	for _, v := range r.Form["ev1"] {
		if _, ok := cev1[v]; !ok {
			return
		}
	}
	cev2 := toMap(eev2)
	for _, v := range r.Form["ev2"] {
		if _, ok := cev2[v]; !ok {
			return
		}
	}
	step = 1

	// Checkout the project and initialize the release.
	nodes := make([]deploy.Dest, len(w))
	for k, v := range w {
		nodes[k], err = client.OpenRPC(v.(*db.Node).Addr, 500*time.Millisecond)
		if err != nil {
			step = 0
			return
		}
	}
	force, _ := strconv.Atoi(r.Form.Get("force"))
	if force == 1 {
		// A force push is required.
		// Adds a fake destination as main server to do that.
		nodes = append([]deploy.Dest{deploy.ServerLess}, nodes...)
	}
	out = deploy.New(project, nodes[0], nodes[1:]...)
	if err = out.Checkout(r.Form["ev1"], r.Form["ev2"]); err != nil {
		return
	}
	if len(r.Form["vars"]) == 0 && force == 0 {
		// Force push does not required
		return
	}
	step = 2
	err = out.Push(r.Form["vars"]...)

	return
}

func (s *Server) nodes() ([]db.Keyer, error) {
	nodes, err := s.db.Nodes()
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, errors.New("expected at least one server")
	}
	return nodes, nil
}
