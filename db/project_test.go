package db_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/rvflash/eve/db"
)

// TestProject applies the basic tests on a project.
func TestNewProject(t *testing.T) {
	var tt = struct {
		id,
		key,
		name,
		desc string
	}{
		id:   "my-fist-project",
		key:  "my-first-project",
		name: "My Fist Project",
		desc: "Anything about the boxer",
	}
	p := db.NewProject(tt.name, tt.desc)
	if tt.id != p.ID {
		t.Errorf("id mismatch: exp=%q got=%q", tt.id, p.ID)
	}
	if tt.name != p.Name {
		t.Errorf("title mismatch: exp=%q got=%q", tt.name, p.Name)
	}
	if tt.desc != p.Description {
		t.Errorf("title mismatch: exp=%q got=%q", tt.desc, p.Description)
	}
	if p.AutoIncrementing() {
		t.Fatal("project's bucket is not auto incremented")
	}
	if p.Deployed() {
		t.Error("new project has never been deployed")
	}
	if n := len(p.Envs()); n != 2 {
		t.Errorf("new project has 2 env by default: got=%q", n)
	}
	if !reflect.DeepEqual(p.FirstEnv(), db.DefaultEnv) {
		t.Errorf("main env mismatch: exp=%q got=%q", p.FirstEnv(), db.DefaultEnv)
	}
	if !reflect.DeepEqual(p.SecondEnv(), db.DefaultEnv) {
		t.Errorf("second env mismatch: exp=%q got=%q", p.SecondEnv(), db.DefaultEnv)
	}
	if n := len(p.Vars()); n > 0 {
		t.Errorf("new project has no var: got=%q", n)
	}
	if !p.LastUpdateTs.IsZero() {
		t.Fatal("new project must not have update date")
	}
	// We change its last update date.
	p.Updated()
	if p.LastUpdateTs.IsZero() || !p.LastUpdateTs.Before(time.Now()) {
		t.Fatal("the last update date must be less than now")
	}
	if !reflect.DeepEqual(p.Key(), []byte(tt.id)) {
		t.Errorf("key mismatch: exp=%v got=%v", p.Key(), []byte(tt.id))
	}
	if err := p.SetKey([]byte(tt.key)); err != nil || p.ID != tt.key {
		t.Errorf("new key mismatch: exp=%v got=%v", tt.key, p.ID)
	}
}

// TestNewProject_Envs tests the management of the environments on a project.
func TestNewProject_Envs(t *testing.T) {
	var dt = []struct {
		in, in1         *db.Env
		err, err1, err2 error
	}{
		// error
		{
			in:   &db.Env{},
			in1:  &db.Env{},
			err:  db.ErrMissing,
			err1: db.ErrMissing,
			err2: db.ErrMissing,
		},
		// valid
		{
			in:   &db.Env{ID: 1},
			in1:  &db.Env{ID: 1},
			err2: db.ErrOutOfBounds,
		},
		{
			in:   &db.Env{ID: 1},
			in1:  &db.Env{ID: 2},
			err1: db.ErrNotFound,
			err2: db.ErrOutOfBounds,
		},
	}
	p := db.NewProject("test", "")
	for i, tt := range dt {
		if err := p.AddEnv(tt.in); tt.err != err {
			t.Errorf("%d. add error mismatch: exp=%q got=%q", i, tt.err, err)
		}
		if err := p.AddEnv(tt.in1); tt.err != err {
			t.Errorf("%d. add error mismatch: exp=%q got=%q", i, tt.err, err)
		}
		if err := p.AddEnv(tt.in1); tt.err2 != err {
			t.Errorf("%d. add error mismatch: exp=%q got=%q", i, tt.err2, err)
		}
		if err := p.DeleteEnv(tt.in); tt.err != err {
			t.Errorf("%d. del error mismatch: exp=%q got=%q", i, tt.err, err)
		}
		if err := p.DeleteEnv(tt.in); tt.err1 != err {
			t.Errorf("%d. del error mismatch: exp=%q got=%q", i, tt.err1, err)
		}
	}
}

// TestNewProject_Vars tests the management of the variables on a project.
func TestNewProject_Vars(t *testing.T) {
	var dt = []struct {
		in        *db.Var
		err, err1 error
	}{
		{in: &db.Var{}, err: db.ErrMissing, err1: db.ErrMissing},
		{in: &db.Var{ID: 1}, err1: db.ErrNotFound},
	}
	p := db.NewProject("test", "")
	for i, tt := range dt {
		if err := p.AddVar(tt.in); tt.err != err {
			t.Errorf("%d. add error mismatch: exp=%q got=%q", i, tt.err, err)
		}
		if err := p.DeleteVar(tt.in); tt.err != err {
			t.Errorf("%d. del error mismatch: exp=%q got=%q", i, tt.err, err)
		}
		if err := p.DeleteVar(tt.in); tt.err1 != err {
			t.Errorf("%d. del error mismatch: exp=%q got=%q", i, tt.err1, err)
		}
	}
}
