package db_test

import (
	"reflect"
	"testing"

	"github.com/pkg/errors"
	"github.com/rvflash/eve/db"
)

func TestNewKind(t *testing.T) {
	var dt = []struct {
		i, id             int
		out               db.Kind
		str, zstr, regexp string
		zval              interface{}
	}{
		{i: 0, out: db.Unknown, id: 0, str: "Unknown", zstr: "", regexp: "", zval: nil},
		{i: 1, out: db.Int, id: 1, str: "Int", zstr: "0", regexp: "[0-9]+", zval: 0},
		{i: 2, out: db.Float, id: 2, str: "Float", zstr: "0", regexp: "[0-9.]+", zval: 0},
		{i: 3, out: db.Bool, id: 3, str: "Bool", zstr: "false", regexp: `\b(true|false)\b`, zval: false},
		{i: 4, out: db.String, id: 4, str: "String", zstr: "", regexp: "", zval: ""},
	}
	for i, tt := range dt {
		k := db.NewKind(tt.i)
		if !reflect.DeepEqual(k, tt.out) {
			t.Errorf("%d. %q kind mismatch: exp=%q got=%q", i, tt.i, tt.out, k)
		}
		if v := k.Int(); tt.id != v {
			t.Errorf("%d. %q kind ID mismatch: exp=%q got=%q", i, tt.i, tt.id, v)
		}
		if v := k.String(); tt.str != v {
			t.Errorf("%d. %q kind name mismatch: exp=%q got=%q", i, tt.i, tt.str, v)
		}
		if v := k.ZeroString(); tt.zstr != v {
			t.Errorf("%d. %q kind name mismatch: exp=%q got=%q", i, tt.i, tt.zstr, v)
		}
		if v := k.ZeroValue(); tt.zval != v {
			t.Errorf("%d. %q kind name mismatch: exp=%q got=%q", i, tt.i, tt.zval, v)
		}
		if v := k.Pattern(); tt.regexp != v {
			t.Errorf("%d. %q kind name mismatch: exp=%q got=%q", i, tt.i, tt.regexp, v)
		}
	}
}

func TestKindAssert(t *testing.T) {
	var dt = []struct {
		on      db.Kind
		in, out interface{}
		ok      bool
	}{
		// ko
		{db.Unknown, nil, nil, false},
		{db.Bool, "", false, false},
		// ok
		{db.String, "rv", "rv", true},
		{db.Int, 1, 1, true},
		{db.Float, 3.14, 3.14, true},
		{db.Bool, true, true, true},
	}
	for i, tt := range dt {
		if out, ok := tt.on.Assert(tt.in); ok != tt.ok {
			t.Errorf("%d. kind mismatch: exp=%t got=%t", i, tt.ok, ok)
		} else if out != tt.out {
			t.Errorf("%d. result mismatch: exp=%v got=%v", i, tt.out, out)
		}
	}
}

func TestKindParse(t *testing.T) {
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
		if !reflect.DeepEqual(tt.err, err) {
			t.Errorf("%d. error mismatch: exp=%q got=%q", i, tt.err, err)
		}
		if !reflect.DeepEqual(tt.out, out) {
			t.Errorf("%d. content mismatch: exp=%v got=%v", i, tt.out, out)
		}
	}
}

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

func TestVarSetValues(t *testing.T) {
	var dt = []struct {
		on  *db.Var
		in  map[string]string
		out db.EnvsValue
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
			on:  db.NewVar("b", db.Bool.Int()),
			in:  map[string]string{"name": "true"},
			out: db.EnvsValue{"name": true},
		},
		{
			on:  db.NewVar("i", db.Int.Int()),
			in:  map[string]string{"name": "10", "name1": "11", "name2": "12"},
			out: db.EnvsValue{"name": 10, "name1": 11, "name2": 12},
		},
		{
			on:  db.NewVar("f", db.Float.Int()),
			in:  map[string]string{"name": "3.14"},
			out: db.EnvsValue{"name": 3.14},
		},
		{
			on:  db.NewVar("s", db.String.Int()),
			in:  map[string]string{"name": "true"},
			out: db.EnvsValue{"name": "true"},
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
		} else if !reflect.DeepEqual(tt.out, tt.on.Values) {
			t.Errorf("%d. content mismatch: exp=%q got=%q", i, tt.out, tt.on.Values)
		}
	}
}

func TestVarCleanValues(t *testing.T) {
	var dt = []struct {
		on      *db.Var
		with    [2]*db.Env
		out     db.EnvsValue
		partial bool
		err     error
	}{
		// errors
		{
			on:   &db.Var{Name: "b", Kind: db.Bool, Values: db.EnvsValue{"_.": ""}},
			with: [2]*db.Env{db.DefaultEnv, db.DefaultEnv},
			out:  db.EnvsValue{},
			err:  errors.WithMessage(db.ErrInvalid, "_."),
		},
		// valid
		{
			on:      &db.Var{Name: "b", Kind: db.Bool, Values: db.EnvsValue{"_r.": false}},
			with:    [2]*db.Env{db.DefaultEnv, db.DefaultEnv},
			out:     db.EnvsValue{"_.": false},
			partial: true,
		},
		{
			on:      &db.Var{Name: "b", Kind: db.Bool, Values: db.EnvsValue{"_.v": false}},
			with:    [2]*db.Env{db.DefaultEnv, db.DefaultEnv},
			out:     db.EnvsValue{"_.": false},
			partial: true,
		},
		{
			on:   &db.Var{Name: "i", Kind: db.Int, Values: db.EnvsValue{"_.": 666}},
			with: [2]*db.Env{db.DefaultEnv, db.DefaultEnv},
			out:  db.EnvsValue{"_.": 666},
		},
		{
			on:   &db.Var{Name: "f", Kind: db.Float, Values: db.EnvsValue{"_.": 3.14}},
			with: [2]*db.Env{db.DefaultEnv, db.DefaultEnv},
			out:  db.EnvsValue{"_.": 3.14},
		},
		{
			on:   &db.Var{Name: "b", Kind: db.Bool, Values: db.EnvsValue{"_.": true}},
			with: [2]*db.Env{db.DefaultEnv, db.DefaultEnv},
			out:  db.EnvsValue{"_.": true},
		},
		{
			on:   &db.Var{Name: "s", Kind: db.String, Values: db.EnvsValue{"_.": ""}},
			with: [2]*db.Env{db.DefaultEnv, db.DefaultEnv},
			out:  db.EnvsValue{"_.": ""},
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
				t.Errorf("%d. partial content mismatch: exp=%t got=%t", i, tt.partial, tt.on.Partial)
			}
		}
	}
}
