package db_test

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/rvflash/eve/db"
)

// TestNewKind tests the Kind struct.
func TestNewKind(t *testing.T) {
	var dt = []struct {
		i, id int
		out   db.Kind
		str   string
	}{
		{i: 0, out: db.Unknown, id: 0, str: "Unknown"},
		{i: 1, out: db.Int, id: 1, str: "Int"},
		{i: 2, out: db.Float, id: 2, str: "Float"},
		{i: 3, out: db.Bool, id: 3, str: "Bool"},
		{i: 4, out: db.String, id: 4, str: "String"},
		{i: 5, out: db.Unknown, id: 0, str: "Unknown"},
	}
	for i, tt := range dt {
		k := db.NewKind(tt.i)
		if !reflect.DeepEqual(k, tt.out) {
			t.Errorf("%d. %q kind mismatch: exp=%q got=%q", i, tt.i, tt.out, k)
		}
		if tt.id != k.Int() {
			t.Errorf("%d. %q kind ID mismatch: exp=%q got=%q", i, tt.i, tt.id, k.Int())
		}
		if tt.str != k.String() {
			t.Errorf("%d. %q kind name mismatch: exp=%q got=%q", i, tt.i, tt.str, k.String())
		}
	}
}

// TestKind_Assert tests the method Assert on a Kind.
func TestKind_Assert(t *testing.T) {
	var dt = []struct {
		on  db.Kind
		in  interface{}
		out bool
	}{
		// ko
		{db.Unknown, nil, false},
		{db.Bool, "", false},
		// ok
		{db.String, "rv", true},
		{db.Int, 1, true},
		{db.Float, 3.14, true},
		{db.Bool, true, true},
	}
	for i, tt := range dt {
		if out := tt.on.Assert(tt.in); out != tt.out {
			t.Errorf("%d. kind mismatch: exp=%q got=%q", i, tt.out, out)
		}
	}
}

// TestNewVarID tests the method NewVarID.
func TestNewVarID(t *testing.T) {
	var dt = []struct {
		in  string
		out *db.VarID
	}{
		{"", db.NewVarID("")},
		{db.VarIDTie, db.NewVarID("")},
		{db.VarIDPrefix, db.NewVarID("")},
		{db.VarIDPrefix + db.VarIDTie, db.NewVarID("")},
		{db.VarIDPrefix + "" + db.VarIDTie + "", db.NewVarID("")},
		{db.VarIDPrefix + "r" + db.VarIDTie + "v", &db.VarID{"r", "v"}},
		{db.VarIDPrefix + "r" + db.VarIDTie + "v" + db.VarIDTie, &db.VarID{"r", "v" + db.VarIDTie}},
	}
	for i, tt := range dt {
		if out := db.NewVarID(tt.in); !reflect.DeepEqual(out, tt.out) {
			t.Errorf("%d. ID mismatch: exp=%q got=%q", i, tt.out, out)
		}
	}
}

// TestVar_ParseValue tests the method ParseValue on a Var.
func TestVar_ParseValue(t *testing.T) {
	var dt = []struct {
		on  *db.Var
		in  string
		out interface{}
		err error
	}{
		{
			on:  db.NewVar("b", db.Bool.Int()),
			in:  "false",
			out: false,
		},
		{
			on:  db.NewVar("i", db.Int.Int()),
			in:  "666",
			out: 666,
		},
		{
			on:  db.NewVar("f", db.Float.Int()),
			in:  "3.14",
			out: 3.14,
		},
		{
			on:  db.NewVar("s", db.String.Int()),
			in:  "false",
			out: "false",
		},
		{
			on:  db.NewVar("u", db.Unknown.Int()),
			in:  "false",
			out: nil,
			err: db.ErrOutOfBounds,
		},
	}
	for i, tt := range dt {
		out, err := tt.on.Kind.Parse(tt.in)
		if !reflect.DeepEqual(err, tt.err) {
			t.Errorf("%d. error mismatch: exp=%q got=%q", i, tt.err, err)
		}
		if !reflect.DeepEqual(out, tt.out) {
			t.Errorf("%d. content mismatch: exp=%q got=%q", i, tt.out, out)
		}
	}
}

// TestVar_SetValues tests the method SetValues on a Var.
func TestVar_SetValues(t *testing.T) {
	var dt = []struct {
		on  *db.Var
		in  map[string]string
		err error
	}{
		// errors
		{
			on:  db.NewVar("b", db.Bool.Int()),
			in:  map[string]string{"name": "value"},
			err: errors.New(`name: strconv.ParseBool: parsing "value": invalid syntax`),
		},
		{
			on:  db.NewVar("f", db.Float.Int()),
			in:  map[string]string{"name": "3.14", "name2": "3,14"},
			err: errors.New(`name2: strconv.ParseFloat: parsing "3,14": invalid syntax`),
		},
		// valid
		{
			on: db.NewVar("b", db.Bool.Int()),
			in: map[string]string{"name": "true"},
		},
		{
			on: db.NewVar("i", db.Int.Int()),
			in: map[string]string{"name": "10", "name1": "11", "name2": "12"},
		},
		{
			on: db.NewVar("f", db.Float.Int()),
			in: map[string]string{"name": "3.14"},
		},
		{
			on: db.NewVar("s", db.String.Int()),
			in: map[string]string{"name": "true"},
		},
	}
	for i, tt := range dt {
		if err := tt.on.SetValues(tt.in); err != nil {
			if tt.err == nil {
				t.Errorf("%d. expected no error: got=%q", i, err)
			} else if err.Error() != tt.err.Error() {
				t.Errorf("%d. error mismatch: exp=%q got=%q", i, tt.err, err)
			}
		} else if tt.err != nil {
			t.Errorf("%d. expected error: exp=%q got=%q", i, tt.err, err)
		}
	}
}

// TestVar_CleanValues tests the method CleanValues on a Var.
func TestVar_CleanValues(t *testing.T) {
	var dt = []struct {
		on      *db.Var
		with    [2]*db.Environment
		out     db.EnvsValue
		partial bool
		err     error
	}{
		// errors
		{
			on:   &db.Var{Name: "b", Kind: db.Bool, Values: db.EnvsValue{"_.": ""}},
			with: [2]*db.Environment{db.DefaultEnv, db.DefaultEnv},
			out:  db.EnvsValue{},
			err:  errors.WithMessage(db.ErrInvalid, "_."),
		},
		// valid
		{
			on:      &db.Var{Name: "b", Kind: db.Bool, Values: db.EnvsValue{"_r.": false}},
			with:    [2]*db.Environment{db.DefaultEnv, db.DefaultEnv},
			out:     db.EnvsValue{"_.": false},
			partial: true,
		},
		{
			on:      &db.Var{Name: "b", Kind: db.Bool, Values: db.EnvsValue{"_.v": false}},
			with:    [2]*db.Environment{db.DefaultEnv, db.DefaultEnv},
			out:     db.EnvsValue{"_.": false},
			partial: true,
		},
		{
			on:   &db.Var{Name: "b", Kind: db.Bool, Values: db.EnvsValue{"_.": false}},
			with: [2]*db.Environment{db.DefaultEnv, db.DefaultEnv},
			out:  db.EnvsValue{"_.": false},
		},
	}
	for i, tt := range dt {
		err := tt.on.CleanValues(tt.with[0], tt.with[1])
		if !reflect.DeepEqual(tt.err, err) {
			t.Errorf("%d. error mismatch: exp=%q got=%q", i, tt.err, err)
		} else if err == nil {
			if !reflect.DeepEqual(tt.out, tt.on.Values) {
				t.Errorf("%d. content mismatch: exp=%q got=%q", i, tt.out, tt.on.Values)
			}
			if tt.partial != tt.on.Partial {
				t.Errorf("%d. partial content mismatch: exp=%q got=%q", i, tt.partial, tt.on.Partial)
			}
		}
	}
}
