package db_test

import (
	"os"
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/rvflash/eve/db"
)

const (
	dbTest = "test.db"

	// Actions
	doGet = "get"
	doAdd = "add"
	doUpd = "upd"
	doDel = "del"
)

type dbt struct {
	r    *db.Data
	vs   []uint64
	v, s uint64
}

// openDb returns an instance of database dedicated to tests.
func openDb() (*dbt, error) {
	// Cleans workspace.
	_ = os.Remove(dbTest)
	// Opens the database.
	r, err := db.Open(dbTest)
	if err != nil {
		return nil, err
	}
	return &dbt{r: r}, nil
}

func (d *dbt) createNode() error {
	return d.r.AddNode(db.NewNode(":9090"))
}

func (d *dbt) createProject() error {
	p := db.NewProject("test", "")
	return d.r.AddProject(p)
}

func (d *dbt) createProjectWithVar() error {
	if err := d.createProject(); err != nil {
		return err
	}
	c := db.NewVar("test", db.Bool.Int())
	if err := d.r.AddVarInProject(c, "test"); err != nil {
		return err
	}
	d.v = c.ID

	return nil
}

func (d *dbt) createProjectWithVars() error {
	if err := d.createProject(); err != nil {
		return err
	}

	// Adds the boolean var.
	v := db.NewVar("bool", db.Bool.Int())
	if err := d.r.AddVarInProject(v, "test"); err != nil {
		return err
	}
	d.vs = append(d.vs, v.ID)

	// Adds the integer var.
	v = db.NewVar("int", db.Int.Int())
	if err := d.r.AddVarInProject(v, "test"); err != nil {
		return err
	}
	d.vs = append(d.vs, v.ID)

	// Adds the float var.
	v = db.NewVar("float", db.Float.Int())
	if err := d.r.AddVarInProject(v, "test"); err != nil {
		return err
	}
	d.vs = append(d.vs, v.ID)

	// Adds the string var.
	v = db.NewVar("string", db.String.Int())
	if err := d.r.AddVarInProject(v, "test"); err != nil {
		return err
	}
	d.vs = append(d.vs, v.ID)

	return nil
}

func (d *dbt) createProjectWithEnv() error {
	if err := d.createProject(); err != nil {
		return err
	}
	return d.createEnv([]string{"test"})
}

func (d *dbt) createProjectWithEnvVar(ev ...string) error {
	if err := d.createProject(); err != nil {
		return err
	}
	c := db.NewVar("test", db.Bool.Int())
	if err := d.r.AddVarInProject(c, "test"); err != nil {
		return err
	}
	d.v = c.ID

	if err := d.createEnv(ev); err != nil {
		return err
	}
	s, err := d.r.GetEnv(d.s)
	if err != nil {
		return err
	}
	return d.r.BindEnvInProject(s.(*db.Env), "test")
}

func (d *dbt) createEnv(v []string) error {
	s := db.NewEnv("test", v)
	if err := d.r.AddEnv(s); err != nil {
		return err
	}
	d.s = s.ID

	return nil
}

func (d *dbt) stop() error {
	// Try to close the connection to the test's database.
	if err := d.r.Close(); err != nil {
		return err
	}
	// Cleans workspace.
	return os.Remove(dbTest)
}

func TestOpen(t *testing.T) {
	var dt = []struct {
		in  string
		err error
	}{
		{in: "", err: db.ErrMissing},
		{in: "new.db"},
	}
	for i, tt := range dt {
		// Opens the database.
		r, err := db.Open(tt.in)
		// Checks the result.
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("%d. %q error mismatch: exp=%q got=%q", i, tt.in, tt.err, err)
		}
		// Closes it.
		if err == nil {
			if err := r.Close(); err != nil {
				t.Fatalf("close %s: %s", tt.in, err)
			}
			if err := os.Remove(tt.in); err != nil {
				t.Fatalf("remove %s: %s", tt.in, err)
			}
		}
	}
}

func TestDataNodes(t *testing.T) {
	// Opens the database.
	dbt, err := openDb()
	if err != nil {
		t.Fatalf("open %s: %s", dbTest, err)
	}
	defer func() { _ = dbt.stop() }()

	if err := dbt.createNode(); err != nil {
		t.Fatalf("unable to create the test's node: %s", err)
	}
	var l []db.Keyer
	if l, err = dbt.r.Nodes(); err != nil {
		t.Errorf("error on nodes's listing: %q", err)
	}
	if s := len(l); s != 1 {
		t.Errorf("nodes size mismatch: exp=%q got=%q", 1, s)
	}
}

func TestDataProjects(t *testing.T) {
	// Opens the database.
	dbt, err := openDb()
	if err != nil {
		t.Fatalf("open %s: %s", dbTest, err)
	}
	defer func() { _ = dbt.stop() }()

	if err := dbt.createProject(); err != nil {
		t.Fatalf("unable to create the test's database: %s", err)
	}
	var l []db.Keyer
	if l, err = dbt.r.Projects(); err != nil {
		t.Errorf("error on projects's listing: %q", err)
	}
	if s := len(l); s != 1 {
		t.Errorf("projects size mismatch: exp=%q got=%q", 1, s)
	}
}

func TestDataEnvs(t *testing.T) {
	// Opens the database.
	dbt, err := openDb()
	if err != nil {
		t.Fatalf("open %s: %s", dbTest, err)
	}
	defer func() { _ = dbt.stop() }()

	if err := dbt.createEnv([]string{"test"}); err != nil {
		t.Fatalf("unable to create an env on the test's database: %s", err)
	}
	var l []db.Keyer
	if l, err = dbt.r.Envs(); err != nil {
		t.Errorf("error on environment's listing: %q", err)
	}
	if s := len(l); s != 1 {
		t.Errorf("envs size mismatch: exp=%q got=%q", 1, s)
	}
}

func TestDataProject(t *testing.T) {
	// Opens the database.
	dbt, err := openDb()
	if err != nil {
		t.Fatalf("open %s: %s", dbTest, err)
	}
	defer func() { _ = dbt.stop() }()

	if err := dbt.createProjectWithEnvVar("test"); err != nil {
		t.Fatalf("unable to create the test's database (full options): %s", err)
	}

	var dt = []struct {
		do, pn string
		p      *db.Project
		err    error
	}{
		// errors
		{
			do:  doGet,
			pn:  "",
			err: db.ErrNotFound,
		},
		{
			do:  doAdd,
			p:   db.NewProject(" ", ""),
			err: db.ErrInvalid,
		},
		{
			do:  doUpd,
			p:   db.NewProject("", ""),
			err: db.ErrInvalid,
		},
		// valid
		{
			do: doGet,
			pn: "test",
			p:  db.NewProject("test", ""),
		},
		{
			do: doDel,
			p:  db.NewProject("", ""),
		},
		{
			do: doDel,
			p:  db.NewProject("test", ""),
		},
	}

	// Launches tests.
	for i, tt := range dt {
		// Creates the project.
		var err error
		if tt.do == doAdd {
			// Adds it.
			err = dbt.r.AddProject(tt.p)
		} else if tt.do == doGet {
			// Gets it.
			var (
				p  db.Keyer
				ok bool
			)
			p, err = dbt.r.GetProject(tt.pn)
			tt.p, ok = p.(*db.Project)
			if !ok {
				t.Fatalf("invalid project: %q", tt.pn)
			}
		} else if tt.do == doUpd {
			// Updates it.
			err = dbt.r.UpsertProject(tt.p)
		} else if tt.do == doDel {
			// Deletes it.
			err = dbt.r.DeleteProject(tt.p)
		} else {
			t.Fatalf("unkwown action to do: %q", tt.do)
		}
		// Checks the result.
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("%d. error mismatch: exp=%q got=%q", i, tt.err, err)
		}
	}
}

func TestDataVar(t *testing.T) {
	// Opens the database.
	dbt, err := openDb()
	if err != nil {
		t.Fatalf("open %s: %s", dbTest, err)
	}
	defer func() { _ = dbt.stop() }()

	// Create one project as container.
	if err := dbt.createProjectWithVar(); err != nil {
		t.Fatalf("unable to create the test's project: %v", err)
	}

	var dt = []struct {
		do, to  string
		in, out *db.Var
		err     error
	}{
		// errors
		{
			do:  doAdd,
			in:  db.NewVar("", 1),
			to:  "test",
			err: errors.WithMessage(db.ErrInvalid, "var"),
		},
		{
			do:  doAdd,
			in:  &db.Var{Name: "r v", Kind: db.Bool},
			to:  "test",
			err: errors.WithMessage(db.ErrInvalid, "var"),
		},
		{
			do:  doAdd,
			in:  &db.Var{Name: "rv", Kind: 0},
			to:  "test",
			err: errors.WithMessage(db.ErrMissing, "var"),
		},
		{
			do:  doAdd,
			in:  &db.Var{Name: "rv", Kind: 10},
			to:  "test",
			err: errors.WithMessage(db.ErrMissing, "var"),
		},
		{
			do:  doAdd,
			in:  db.NewVar(" ", 1),
			to:  "test",
			err: errors.WithMessage(db.ErrInvalid, "var"),
		},
		{
			do:  doUpd,
			in:  db.NewVar("", 1),
			to:  "test",
			err: errors.WithMessage(db.ErrNotFound, "var"),
		},
		{
			do:  doUpd,
			in:  &db.Var{ID: 340, Name: "r v", Kind: db.Bool},
			to:  "test",
			err: errors.WithMessage(db.ErrNotFound, "var"),
		},
		{
			do:  doUpd,
			in:  &db.Var{ID: 340, Name: "rv", Kind: 0},
			to:  "test",
			err: errors.WithMessage(db.ErrNotFound, "var"),
		},
		{
			do:  doUpd,
			in:  &db.Var{ID: 340, Name: "rv", Kind: 10},
			to:  "test",
			err: errors.WithMessage(db.ErrNotFound, "var"),
		},
		{
			do:  doUpd,
			in:  &db.Var{Name: "rv", Kind: 1},
			to:  "test",
			err: errors.WithMessage(db.ErrNotFound, "var"),
		},
		{
			do:  doUpd,
			in:  &db.Var{ID: 340, Name: "rv", Kind: db.Int},
			to:  "test",
			err: errors.WithMessage(db.ErrNotFound, "var"),
		},
		{
			do:  doGet,
			in:  &db.Var{ID: 340, Name: "rv", Kind: db.Int},
			to:  "test",
			err: errors.WithMessage(db.ErrNotFound, "var"),
		},
		{
			do:  doGet,
			in:  &db.Var{ID: 340, Name: "rv", Kind: db.Int},
			err: errors.WithMessage(db.ErrNotFound, "project"),
		},
		{
			do:  doDel,
			in:  &db.Var{ID: 666, Name: "rv", Kind: db.Int},
			to:  "test",
			err: errors.WithMessage(db.ErrNotFound, "var"),
		},
		{
			do:  doDel,
			in:  &db.Var{ID: 666, Name: "rv", Kind: db.Int},
			err: errors.WithMessage(db.ErrNotFound, "project"),
		},
		// Valid
		{
			do:  doAdd,
			in:  db.NewVar("rv", db.Bool.Int()),
			to:  "test",
			out: &db.Var{Name: "rv", Kind: db.Bool},
		},
		{
			do:  doGet,
			in:  &db.Var{ID: dbt.v, Name: "test", Kind: db.Bool},
			to:  "test",
			out: &db.Var{ID: dbt.v, Name: "test", Kind: db.Bool},
		},
		{
			do:  doDel,
			in:  &db.Var{ID: dbt.v, Name: "test", Kind: db.Bool},
			to:  "test",
			out: &db.Var{ID: dbt.v, Name: "test", Kind: db.Bool},
		},
	}

	// Launches tests.
	for i, tt := range dt {
		// Creates the variable.
		var err error
		if tt.do == doGet {
			// Gets it.
			var d db.Keyer
			d, err = dbt.r.GetVarInProject(tt.in.ID, tt.to)
			if err == nil {
				var ok bool
				if tt.in, ok = d.(*db.Var); !ok {
					t.Fatalf("invalid var: %v", tt.in)
				}
			}
		} else if tt.do == doAdd {
			// Adds it.
			err = dbt.r.AddVarInProject(tt.in, tt.to)
		} else if tt.do == doUpd {
			// Updates it.
			err = dbt.r.UpdateVarInProject(tt.in, tt.to)
		} else if tt.do == doDel {
			// Deletes it.
			err = dbt.r.DeleteVarInProject(tt.in, tt.to)
		} else {
			t.Fatalf("unkwown action to do: %q", tt.do)
		}
		// Checks the result.
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("%d. error mismatch: exp=%q got=%q", i, tt.err, err)
		}
		if err == nil {
			if tt.in.ID == 0 {
				t.Errorf("%d. id mismatch: exp= >0 got=%d", i, tt.in.ID)
			}
			if tt.out.Name != tt.in.Name {
				t.Errorf("%d. name mismatch: exp=%q got=%q", i, tt.out.Name, tt.in.Name)
			}
			if tt.out.Kind.Int() != tt.in.Kind.Int() {
				t.Errorf("%d. kind mismatch: exp=%q got=%q", i, tt.out.Kind.Int(), tt.in.Kind.Int())
			}
			if tt.in.LastUpdateTs.IsZero() {
				t.Errorf("%d. update date mismatch: exp= >0 got=%q", i, tt.in.LastUpdateTs)
			}
			if tt.do == doDel && !tt.in.Deleted() {
				t.Errorf("%d. deletion date mismatch: exp= >0 got=%q", i, tt.in.DeletionTs)
			}
		}
	}
}

func TestDataVarLifeCycle(t *testing.T) {
	// Opens the database.
	dbt, err := openDb()
	if err != nil {
		t.Fatalf("open %s: %s", dbTest, err)
	}
	defer func() { _ = dbt.stop() }()

	// Create one project as container with all type of variables.
	if err := dbt.createProjectWithVars(); err != nil {
		t.Fatalf("unable to create the scoped test's project: %v", err)
	}

	// New data by kind to update variables.
	var dt = map[db.Kind]map[string]string{
		db.Bool:   {"_.": "true"},
		db.Int:    {"_.": "666"},
		db.Float:  {"_.": "3.14"},
		db.String: {"_.": "rv"},
	}

	for _, i := range dbt.vs {
		// Tries to get each var.
		d, err := dbt.r.GetVarInProject(i, "test")
		if err != nil {
			t.Fatalf("unable to get var %d: got=%q", i, err)
		}
		v := d.(*db.Var)

		// Tries to update it.
		if err = v.SetValues(dt[v.Kind]); err != nil {
			t.Fatalf("unable to change var's values of %s: got=%q", v.Name, err)
		}
		if err = dbt.r.UpdateVarInProject(v, "test"); err != nil {
			t.Fatalf("unable to update var %s: got=%q", v.Name, err)
		}

		// Checks if the updates has failed.
		if d, err = dbt.r.GetVarInProject(v.ID, "test"); err != nil {
			t.Fatalf("unable to get var %d: got=%q", v.ID, err)
		}
		nv := d.(*db.Var)
		if !reflect.DeepEqual(v.Values, nv.Values) {
			t.Fatalf("content mismatch for var %v: exp:%#v got=%#v", v.Name, v.Values, nv.Values)
		}

		// Tries to delete it.
		if err = dbt.r.DeleteVarInProject(nv, "test"); err != nil {
			t.Fatalf("unable to delete var %s: got=%q", nv.Name, err)
		}
	}
}

func TestDataEnv(t *testing.T) {
	// Opens the database.
	dbt, err := openDb()
	if err != nil {
		t.Fatalf("open %s: %s", dbTest, err)
	}
	defer func() { _ = dbt.stop() }()

	// Create one project as container.
	if err := dbt.createProjectWithEnv(); err != nil {
		t.Fatalf("unable to create the scoped test's project: %v", err)
	}

	var dt = []struct {
		do, into string
		in, out  *db.Env
		err      error
	}{
		// errors
		{
			do:  doAdd,
			in:  db.NewEnv("", []string{"dev"}),
			err: db.ErrInvalid,
		},
		{
			do:  doAdd,
			in:  db.NewEnv("r v", []string{"dev"}),
			err: db.ErrInvalid,
		},
		{
			do:  doAdd,
			in:  db.NewEnv("rv", nil),
			err: db.ErrMissing,
		},
		{
			do:  doAdd,
			in:  db.NewEnv("rv", nil),
			err: db.ErrMissing,
		},
		{
			do:  doUpd,
			in:  db.NewEnv("rv", []string{"missing", "id"}),
			err: db.ErrOutOfBounds,
		},
		{
			do:  doGet,
			in:  &db.Env{ID: 666},
			err: db.ErrNotFound,
		},
		{
			do:   "bind",
			in:   &db.Env{Name: "Environment", Values: []string{"dev", "qa"}},
			into: "test",
			err:  errors.WithMessage(db.ErrMissing, "project: env"),
		},
		{
			do:   "unbind",
			in:   &db.Env{ID: 666, Name: "Environment", Values: []string{"dev", "qa"}},
			into: "test",
			err:  errors.WithMessage(db.ErrNotFound, "env"),
		},
		{
			do:  "unbind",
			in:  &db.Env{ID: dbt.s, Name: "test", Values: []string{"test"}},
			err: errors.WithMessage(db.ErrNotFound, "project"),
		},
		// valid
		{
			do:  doAdd,
			in:  db.NewEnv(" Env", []string{"dev", "qa", "prod", "prod"}),
			out: &db.Env{Name: "Env", Values: []string{"dev", "qa", "prod"}},
		},
		{
			do:  doUpd,
			in:  &db.Env{ID: 666, Name: "Environment", Values: []string{"dev"}},
			out: &db.Env{ID: 666, Name: "Environment", Values: []string{"dev"}},
		},
		{
			do:  doUpd,
			in:  &db.Env{ID: 666, Name: " Environment", Values: []string{"dev", "qa", "qa"}},
			out: &db.Env{ID: 666, Name: "Environment", Values: []string{"dev", "qa"}},
		},
		{
			do:  doGet,
			in:  &db.Env{ID: 666},
			out: &db.Env{ID: 666, Name: "Environment", Values: []string{"dev", "qa"}},
		},
		{
			do:   "bind",
			in:   &db.Env{ID: dbt.s, Name: "test", Values: []string{"test"}},
			out:  &db.Env{ID: dbt.s, Name: "test", Values: []string{"test"}},
			into: "test",
		},
		{
			do:   "unbind",
			in:   &db.Env{ID: dbt.s, Name: "test", Values: []string{"test"}},
			out:  &db.Env{ID: dbt.s, Name: "test", Values: []string{"test"}},
			into: "test",
		},
		{
			do:  doGet,
			in:  &db.Env{ID: dbt.s, Name: "test", Values: []string{"test"}},
			out: &db.Env{ID: dbt.s, Name: "test", Values: []string{"test"}},
		},
		// Errors
		{
			do:  doAdd,
			in:  &db.Env{ID: 666, Name: "Environment", Values: []string{"qa"}},
			err: db.ErrAlreadyExists,
		},
	}

	for i, tt := range dt {
		// Creates the environment.
		var err error
		if tt.do == doGet {
			// Updates it.
			var d db.Keyer
			d, err = dbt.r.GetEnv(tt.in.ID)
			if err == nil {
				var ok bool
				if tt.in, ok = d.(*db.Env); !ok {
					t.Fatalf("invalid env: %v", tt.in)
				}
			}
		} else if tt.do == doUpd {
			// Updates it.
			err = dbt.r.UpsertEnv(tt.in)
		} else if tt.do == doAdd {
			// Adds it.
			err = dbt.r.AddEnv(tt.in)
		} else if tt.do == "bind" {
			// Binds it to the test's project.
			err = dbt.r.BindEnvInProject(tt.in, tt.into)
		} else if tt.do == "unbind" {
			// Unbinds it to the test's project.
			err = dbt.r.UnbindEnvInProject(tt.in, tt.into)
		} else {
			t.Fatalf("unkwown action to do: %q", tt.do)
		}
		// Checks the result.
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("%d. error mismatch: exp=%q got=%q", i, tt.err, err)
		}
		if err == nil {
			if tt.in.ID == 0 {
				t.Errorf("%d. id mismatch: exp= >0 got=%d", i, tt.in.ID)
			}
			if tt.out.Name != tt.in.Name {
				t.Errorf("%d. name mismatch: exp=%q got=%q", i, tt.out.Name, tt.in.Name)
			}
			if !reflect.DeepEqual(tt.out.Values, tt.in.Values) {
				t.Errorf("%d. values mismatch: exp=%q got=%q", i, tt.out.Values, tt.in.Values)
			}
			if tt.in.LastUpdateTs.IsZero() && tt.do != "bind" && tt.do != "unbind" {
				t.Errorf("%d. update date mismatch: exp= >0 got=%q", i, tt.in.LastUpdateTs)
			}
		}
	}
}
