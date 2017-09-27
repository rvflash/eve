// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package main

import (
	"errors"
	"html/template"
	"net/http"
	"strings"
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
	tv.Href = "/projects/" + vars["pid"] + "/"
	tv.Project = p
	if tv.Servers, tv.Err = s.nodes(); tv.Err == nil {
		tv.Step, tv.Release, tv.Err = s.deploy(p, tv.Servers, r)
	}
	// Displays the page.
	if err = t.Execute(w, tv); err != nil {
		s.OopsHandler(w, r, err)
	}
}

func (s *Server) deploy(p db.Keyer, w []db.Keyer, r *http.Request) (step int, out *deploy.Release, err error) {
	values := func(s string) []string {
		return strings.Split(s, ",")
	}
	toMap := func(s []string) map[string]struct{} {
		m := make(map[string]struct{}, len(s))
		for _, v := range s {
			m[v] = struct{}{}
		}
		return m
	}
	r.ParseForm()
	ev1 := values(r.Form.Get("ev1"))
	ev2 := values(r.Form.Get("ev2"))
	project := p.(*db.Project)

	// Nothing to checkout
	eev1, eev2 := project.EnvsValues()
	cev1 := toMap(eev1)
	for _, v := range ev1 {
		if _, ok := cev1[v]; !ok {
			return
		}
	}
	cev2 := toMap(eev2)
	for _, v := range ev2 {
		if _, ok := cev2[v]; !ok {
			return
		}
	}
	step = 1

	// Checkout the project and initialize the release.
	nodes := make([]*client.RPC, len(w))
	for k, v := range w {
		nodes[k], err = client.OpenRPC(v.(*db.Node).Addr, 500*time.Millisecond)
		if err != nil {
			step = 0
			return
		}
	}
	out = deploy.New(project, nodes[0], nodes[1:]...)
	if err = out.Checkout(ev1, ev2); err != nil {
		return
	}
	vars := values(r.Form.Get("vars"))
	if len(vars) == 0 {
		return
	}
	step = 2
	err = out.Push(vars...)
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
