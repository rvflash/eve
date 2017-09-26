// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package main

import (
	"fmt"
	"html/template"
	"net/http"

	"strings"

	"github.com/rvflash/elapsed"
	"github.com/rvflash/eve/db"
)

var (
	tmplPath    = "./html/template"
	tmplFuncMap = template.FuncMap{
		// date
		"elapsed": elapsed.Time,
		// arithmetic
		"inc": func(i int) int { return i + 1 },
		"mod": func(i, j int) bool { return i%j == 0 },
		"mul": func(i, j int) int { return i * j },
		// strings
		"join": func(s []string) string { return strings.Join(s, ", ") },
	}
)

type tmplVars struct {
	Title, Href string
}

// HomeHandler listens and server the homepage.
func (s *Server) HomeHandler(w http.ResponseWriter, r *http.Request) {
	// Adds the function to display time elapsed instead of the datetime.
	t, err := template.New("home.html").Funcs(tmplFuncMap).ParseFiles(
		tmplPath+"/home.html",
		tmplPath+"/home/top.html",
		tmplPath+"/home/bottom.html",
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
	type homeTmplVars struct {
		tmplVars
		Projects,
		Servers []db.Keyer
		Err error
	}
	hv := homeTmplVars{}
	hv.Projects, hv.Err = s.db.Projects()
	hv.Servers, _ = s.db.Nodes()

	// Displays the page.
	if err = t.Execute(w, hv); err != nil {
		s.OopsHandler(w, r, err)
	}
}

// ProjectsHandler display the error status code 404 page.
func (s *Server) NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	http.NotFound(w, r)
}

// OopsHandler responds with an HTTP status code 500.
func (s *Server) OopsHandler(w http.ResponseWriter, r *http.Request, err error) {
	s.log.Println(err)
	http.Error(w, "Oops I did it again", http.StatusInternalServerError)
}

// StaticHandler responds with the content of file in static directory.
func (s *Server) StaticHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "./static/"+r.URL.Path[1:])
}

// jsonHandler prints a JSON response with the appropriate headers.
// {
//   "code": 400,
//   "response": "invalid method"
// }
func (s *Server) jsonHandler(w http.ResponseWriter, res string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	fmt.Fprintf(w, `{"code": %d, "response": %q}`, code, res)
}
