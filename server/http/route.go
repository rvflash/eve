package main

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/rvflash/eve/db"
)

var (
	tmplPath = "./html/template"
)

type tmplVars struct {
	Title string
}

// HomeHandler listens and server the homepage.
func (s *Server) HomeHandler(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles(
		tmplPath+"/home.html",
		tmplPath+"/home/top.html",
		tmplPath+"/home/bottom.html",
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
		Projects []db.Keyer
		Err      error
	}
	hv := homeTmplVars{}
	hv.Projects, hv.Err = s.db.Projects()

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
