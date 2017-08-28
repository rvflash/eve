package db

import (
	"reflect"
	"testing"
)

// TestProject_Envs tests the methods Envs.
func TestProject_Envs(t *testing.T) {
	var dt = []struct {
		im, is, om, os *Environment
	}{
		{om: DefaultEnv, os: DefaultEnv},
		{im: &Environment{ID: 1}, om: &Environment{ID: 1}, os: DefaultEnv},
		{im: &Environment{ID: 1}, is: &Environment{ID: 2}, om: &Environment{ID: 2}, os: &Environment{ID: 1}},
		{im: &Environment{ID: 11}, is: &Environment{ID: 2}, om: &Environment{ID: 11}, os: &Environment{ID: 2}},
	}
	for i, tt := range dt {
		// Creates the project.
		p := NewProject("test", "")
		p.envs = make([]Keyer, 0)

		// And add if necessary the environments.
		if tt.im != nil {
			p.envs = append(p.envs, tt.im)
		}
		if tt.is != nil {
			p.envs = append(p.envs, tt.is)
		}

		// Checks if the listing is ordered as required.
		e := p.Envs()
		if !reflect.DeepEqual(tt.om, e[0]) {
			t.Errorf("%d. content mismatch: exp=%v got=%v", i, tt.om, e[0])
		}
		if !reflect.DeepEqual(tt.os, e[1]) {
			t.Errorf("%d. content mismatch: exp=%v got=%v", i, tt.os, e[1])
		}
	}
}
