package db

import (
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

const (
	// Unknown values
	Unknown Kind = iota

	// Numeric values
	Int
	Float

	// Non-numeric values
	Bool
	String
)

// Kind specifies the kind of value.
type Kind int

// Kinds returns the list of available kinds.
var Kinds = []Kind{Int, Float, Bool, String}

// NewKind returns an instance a Kind.
func NewKind(kind int) Kind {
	for _, k := range Kinds {
		if k.Int() == kind {
			return k
		}
	}
	return Unknown
}

// Assert returns true if the value holds its kind.
func (k Kind) Assert(value interface{}) (ok bool) {
	switch k {
	case Int:
		_, ok = value.(int)
	case Float:
		_, ok = value.(float64)
	case Bool:
		_, ok = value.(bool)
	case String:
		_, ok = value.(string)
	}
	return ok
}

// Int gives the value of the kind.
func (k Kind) Int() int {
	return int(k)
}

// ParseValue convert the string to expected value.
func (k Kind) Parse(s string) (interface{}, error) {
	switch k {
	case Int:
		return strconv.Atoi(s)
	case Float:
		return strconv.ParseFloat(s, 10)
	case Bool:
		return strconv.ParseBool(s)
	case String:
		return s, nil
	}
	return nil, ErrOutOfBounds
}

// Pattern returns the regex pattern to use to validate input.
func (k Kind) Pattern() string {
	switch k {
	case Bool:
		return `\b(true|false)\b`
	case Int:
		return "[0-9]+"
	case Float:
		return "[0-9.]+"
	}
	return ""
}

// String gives the name of the kind.
func (k Kind) String() string {
	switch k {
	case Bool:
		return "Bool"
	case String:
		return "String"
	case Int:
		return "Int"
	case Float:
		return "Float"
	}
	return "Unknown"
}

// ZeroString returns the zero value of the kind as string.
func (k Kind) ZeroString() string {
	switch k {
	case Int, Float:
		return "0"
	case Bool:
		return "false"
	}
	return ""
}

// ZeroValue returns the zero value of the kind.
func (k Kind) ZeroValue() interface{} {
	switch k {
	case Int, Float:
		return 0
	case Bool:
		return false
	case String:
		return ""
	}
	return nil
}

// Var represents a variable.
type Var struct {
	ID           uint64    `json:"id"`
	Name         string    `json:"name"`
	Kind         Kind      `json:"kind"`
	Values       EnvsValue `json:"vals,omitempty"`
	LastUpdateTs time.Time `json:"upd_ts"`
	DeletionTs   time.Time `json:"del_ts,omitempty"`
	Partial      bool
}

// NewVar returns an instance a Var.
func NewVar(name string, kind int) *Var {
	return &Var{
		Name: name,
		Kind: NewKind(kind),
	}
}

// AutoIncrementing return true in order to have auo-increment primary key.
func (v *Var) AutoIncrementing() bool {
	return true
}

// Deleted returns true if the variable has been deleted.
func (v *Var) Deleted() bool {
	return !v.DeletionTs.IsZero()
}

// Key returns the key of the variable.
func (v *Var) Key() []byte {
	if v.ID == 0 {
		return nil
	}
	return itob(v.ID)
}

// SetKey returns if error if the change of the key failed.
func (v *Var) SetKey(k []byte) error {
	v.ID = btoi(k)
	return nil
}

// Updated changes the last update date of the variable.
func (v *Var) Updated() {
	v.LastUpdateTs = time.Now()
}

// Valid checks if the variable has required properties.
func (v *Var) Valid(insert bool) error {
	if !check(v.Name) {
		return ErrInvalid
	}
	// Validates the enum range.
	v.Kind = NewKind(v.Kind.Int())
	if v.Kind == Unknown {
		return ErrMissing
	}
	if !insert && v.ID == 0 {
		return ErrOutOfBounds
	}
	return nil
}

// EnvsValue contains all variable's values by given envs.
// Key combines environment names with a colon.
type EnvsValue map[string]interface{}

// NewValues returns a new map of values for the given environments.
// Each value use the default value of the kind of the variable.
func (v *Var) NewValues(env1, env2 *Environment) EnvsValue {
	m := make(EnvsValue, len(env1.Values)*len(env2.Values))
	for _, mv := range env1.Values {
		for _, sv := range env2.Values {
			k := VarID{mv, sv}
			m[k.String()] = v.Kind.ZeroValue()
		}
	}
	return m
}

// SetValues sets the values of the variable without any check on the environments behind.
func (v *Var) SetValues(m map[string]string) (err error) {
	ev := make(EnvsValue, len(m))
	for k, d := range m {
		if ev[k], err = v.Kind.Parse(d); err != nil {
			return errors.WithMessage(err, k)
		}
	}
	v.Values = ev

	return nil
}

// CleanValues ensures that all values use the kind of the variable.
// It also checks that only the current environments values are used.
// A partial result is returned if one the environment does not exist.
// It return on error if the kind of value does not match.
func (v *Var) CleanValues(env1, env2 *Environment) error {
	v.Partial = false
	// Transforms the values of envs as map to perform search on it.
	ev1 := toMap(env1.Values)
	ev2 := toMap(env2.Values)
	// Creates a temporary map to keep only the ok values.
	val := v.NewValues(env1, env2)
	// Checks each value for all envs.
	for k, d := range v.Values {
		vid := NewVarID(k)
		if _, ok := ev1[vid.EnvValue1]; !ok {
			// Unknown value in the main environment.
			v.Partial = true
			continue
		}
		if _, ok := ev2[vid.EnvValue2]; !ok {
			// Unknown value in the second environment.
			v.Partial = true
			continue
		}
		if !v.Kind.Assert(d) {
			return errors.WithMessage(ErrInvalid, k)
		}
		val[vid.String()] = d
	}
	v.Values = val

	return nil
}

// envTie is used to join two environment'values.
const (
	VarIDPrefix = "_"
	VarIDTie    = "."
)

type VarID struct {
	EnvValue1,
	EnvValue2 string
}

func NewVarID(s string) *VarID {
	v := &VarID{}
	if !strings.HasPrefix(s, VarIDPrefix) {
		return v
	}
	d := strings.SplitN(strings.TrimPrefix(s, VarIDPrefix), VarIDTie, 2)
	switch len(d) {
	case 2:
		v.EnvValue1, v.EnvValue2 = d[0], d[1]
	case 1:
		v.EnvValue1 = d[0]
	}
	return v
}

func (v *VarID) String() string {
	return VarIDPrefix + v.EnvValue1 + VarIDTie + v.EnvValue2
}
