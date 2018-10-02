// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package main

import (
	"encoding/binary"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"strings"

	"github.com/gorilla/mux"
	"github.com/rvflash/eve/db"
)

type varHandler struct {
	s    *Server
	p, v db.Keyer
	rv   map[string]string
}

// VarHandler manages all actions to perform on variables.
func (s *Server) VarHandler(w http.ResponseWriter, r *http.Request) {
	// Tries to handle the request.
	var err error
	h := &varHandler{s: s, rv: mux.Vars(r)}
	if h.p, err = s.db.GetProject(h.rv["pid"]); err != nil {
		s.log.Println(err)
		s.NotFoundHandler(w, r)
		return
	}
	var vid uint64
	if vid, err = strconv.ParseUint(h.rv["vid"], 10, 64); err != nil {
		s.NotFoundHandler(w, r)
		return
	}
	if h.v, err = s.db.GetVarInProject(vid, h.rv["pid"]); err != nil {
		s.log.Println(err)
		s.NotFoundHandler(w, r)
		return
	}
	switch r.Method {
	case "POST":
		h.putHandler(w, r)
	case "GET":
		if strings.HasSuffix(r.URL.Path, "/delete") {
			h.deleteHandler(w, r)
		} else {
			h.getHandler(w, r)
		}
	}
}

func (h *varHandler) deleteHandler(w http.ResponseWriter, r *http.Request) {
	// Deletes this variable on the given project.
	if err := h.s.db.DeleteVarInProject(h.v.(*db.Var), h.rv["pid"]); err != nil {
		h.s.jsonHandler(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Redirects to the variable's page.
	loc := fmt.Sprintf("/project/%s/", h.rv["pid"])
	h.s.jsonHandler(w, loc, http.StatusOK)
}

func (h *varHandler) getHandler(w http.ResponseWriter, r *http.Request) {
	// Builds the page.
	t, err := template.New("var.html").Funcs(tmplFuncMap).ParseFiles(
		tmplPath+"/var.html",
		tmplPath+"/var/table.html",
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
		h.s.OopsHandler(w, r, err)
		return
	}

	// Assigns vars to the templates.
	type varTmplVars struct {
		projectTmplVars
		VarIDPrefix,
		VarIDTie string
	}
	tv := varTmplVars{
		VarIDPrefix: db.VarIDPrefix,
		VarIDTie:    db.VarIDTie,
	}
	tv.Title = h.p.(*db.Project).Name
	tv.Href = "/project/" + h.rv["pid"] + "/"
	tv.Var = h.v
	tv.Project = h.p
	tv.Kinds = db.Kinds
	tv.Servers, _ = h.s.db.Nodes()

	// Displays the page.
	if err = t.Execute(w, tv); err != nil {
		h.s.OopsHandler(w, r, err)
	}
}

func (h *varHandler) putHandler(w http.ResponseWriter, r *http.Request) {
	// Try to update the given variable.
	if err := r.ParseForm(); err != nil {
		h.s.jsonHandler(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Builds the map of new values for the response.
	v := h.v.(*db.Var)
	m := make(map[string]string)
	// Parses all the url values and gets as string each value.
	var s string
	for k := range r.PostForm {
		if s = r.PostForm.Get(k); s == "" {
			s = v.Kind.ZeroString()
		}
		m[k] = s
	}
	if err := v.SetValues(m); err != nil {
		h.s.jsonHandler(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := h.s.db.UpdateVarInProject(v, h.rv["pid"]); err != nil {
		h.s.jsonHandler(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Redirects to the variable's page.
	h.s.jsonHandler(w, r.URL.Path, http.StatusOK)
}

// VarsHandler manages the creation of a project's variable.
func (s *Server) VarsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		s.jsonHandler(w, "invalid method", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		s.jsonHandler(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Try to create one variable.
	vars := mux.Vars(r)
	kind, err := strconv.Atoi(r.Form.Get("kind"))
	if err != nil {
		s.jsonHandler(w, "missing kind", http.StatusBadRequest)
		return
	}
	c := db.NewVar(r.Form.Get("name"), kind)
	if err = s.db.AddVarInProject(c, vars["pid"]); err != nil {
		s.jsonHandler(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Redirects to the project page.
	vid := binary.BigEndian.Uint64(c.Key())
	loc := fmt.Sprintf("/project/%s/var/%d", vars["pid"], vid)
	s.jsonHandler(w, loc, http.StatusOK)
}
