// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/rvflash/eve/db"
)

type projectTmplVars struct {
	tmplVars
	Project, Var  db.Keyer
	Envs, Servers []db.Keyer
	Kinds         []db.Kind
}

// ProjectHandler displays all the information about a project.
func (s *Server) ProjectHandler(w http.ResponseWriter, r *http.Request) {
	// Retrieves data to display.
	vars := mux.Vars(r)
	p, err := s.db.GetProject(vars["pid"])
	if err != nil {
		s.NotFoundHandler(w, r)
		return
	}
	// Builds the page.
	var t *template.Template
	t, err = template.New("project.html").Funcs(tmplFuncMap).ParseFiles(
		tmplPath+"/project.html",
		tmplPath+"/project/top.html",
		tmplPath+"/project/bottom.html",
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
	pr := p.(*db.Project)
	tv := projectTmplVars{}
	tv.Title = pr.Name
	tv.Project = pr
	tv.Envs, _ = s.db.Envs(pr.EnvList...)
	tv.Kinds = db.Kinds
	tv.Servers, _ = s.db.Nodes()

	// Displays the page.
	if err = t.Execute(w, tv); err != nil {
		s.OopsHandler(w, r, err)
	}
}

// ProjectsHandler listens post data to create a project and go its detail page.
func (s *Server) ProjectsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.jsonHandler(w, "invalid method", http.StatusBadRequest)
		return
	}
	r.ParseForm()

	// Try to create a new project.
	p := db.NewProject(r.Form.Get("name"), r.Form.Get("desc"))
	if err := s.db.AddProject(p); err != nil {
		s.jsonHandler(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Redirects to the new project page.
	loc := fmt.Sprintf("/project/%s/", p.Key())
	s.jsonHandler(w, loc, http.StatusOK)
}
