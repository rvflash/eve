// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package db

import (
	"strings"
	"time"
)

// Project represents the container of the vars.
type Project struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Description  string    `json:"desc,omitempty"`
	LastUpdateTs time.Time `json:"upd_ts"`
	LastDeployTs time.Time `json:"dep_ts,omitempty"`
	EnvList      []uint64  `json:"envs,omitempty"`
	VarList      []uint64  `json:"vars,omitempty"`
	envs, vars   []Keyer
}

// NewProject creates a new instance of Project.
func NewProject(name, desc string) *Project {
	return &Project{
		ID:          clean(name),
		Name:        name,
		Description: desc,
	}
}

// AutoIncrementing return false in order to manage primary by itself.
func (p *Project) AutoIncrementing() bool {
	return false
}

// Vars returns all the variables of the project.
func (p *Project) Vars() []Keyer {
	return p.vars
}

// AddVar adds a variable to the project.
func (p *Project) AddVar(v *Var) error {
	if v.ID == 0 {
		return ErrMissing
	}
	p.VarList = append(p.VarList, v.ID)

	return nil
}

// DeleteVar removes a project's variable.
func (p *Project) DeleteVar(v *Var) (err error) {
	if v.ID == 0 {
		return ErrMissing
	}
	if p.VarList, err = remove(p.VarList, v.ID); err != nil {
		return err
	}
	return nil
}

// Envs returns the envs of the project.
func (p *Project) Envs() (envs []Keyer) {
	envs = make([]Keyer, 2, 2)
	switch len(p.envs) {
	case 2:
		// A choice must be done to select a persistent "main" environment.
		// We arbitrary choose the env with the bigger identifier as leader.
		// Prevents flip flaps in case of changing of number of environment values.
		if p.envs[0].(*Environment).ID > p.envs[1].(*Environment).ID {
			envs[0], envs[1] = p.envs[0], p.envs[1]
		} else {
			envs[0], envs[1] = p.envs[1], p.envs[0]
		}
	case 1:
		envs[0], envs[1] = p.envs[0], DefaultEnv
	default:
		envs[0], envs[1] = DefaultEnv, DefaultEnv
	}
	return
}

// AddEnv adds a env to the project.
// Only two dimensions are managed by the system.
func (p *Project) AddEnv(e *Environment) error {
	if len(p.EnvList) > 1 {
		return ErrOutOfBounds
	}
	if e.ID == 0 {
		return ErrMissing
	}
	p.EnvList = append(p.EnvList, e.ID)

	return nil
}

// DeleteEnv removes a project's env.
func (p *Project) DeleteEnv(e *Environment) (err error) {
	if e.ID == 0 {
		return ErrMissing
	}
	if p.EnvList, err = remove(p.EnvList, e.ID); err != nil {
		return err
	}
	return nil
}

// FirstEnv is an alias to access the first environment.
func (p *Project) FirstEnv() *Environment {
	return p.Envs()[0].(*Environment)
}

// SecondEnv is an alias to access the second environment.
func (p *Project) SecondEnv() *Environment {
	return p.Envs()[1].(*Environment)
}

// Deployed returns true if the project if already deployed.
func (p *Project) Deployed() bool {
	return !p.LastDeployTs.IsZero()
}

// Updated changes the last update date of the variable.
func (p *Project) Updated() {
	p.LastUpdateTs = time.Now()
}

// Key returns the key used to store it.
func (p *Project) Key() []byte {
	return []byte(p.ID)
}

// SetKey returns if error if the change of the key failed.
func (p *Project) SetKey(k []byte) error {
	p.ID = string(k)
	return nil
}

// Hash returns a unique hash for the string in the project's context.
func (p *Project) Hash(s string) []byte {
	return []byte(p.ID + clean(s))
}

// Valid checks if all required data as well formed.
func (p *Project) Valid(insert bool) error {
	// Assumes that someone will not use the new func.
	if p.ID == "" {
		p.ID = clean(p.Name)
	}
	if p.ID == "" || !check(p.ID) {
		return ErrInvalid
	}
	p.Name = strings.TrimSpace(p.Name)
	p.Description = strings.TrimSpace(p.Description)

	return nil
}
