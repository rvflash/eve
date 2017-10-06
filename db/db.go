// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package db

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
)

// Error messages.
var (
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalid       = errors.New("invalid data")
	ErrMissing       = errors.New("missing data")
	ErrNotFound      = errors.New("not found")
	ErrOutOfBounds   = errors.New("out of bounds")
	ErrUnknown       = errors.New("unknown data")
)

// List of available buckets.
var (
	// tables
	projects = []byte("projects")
	envs     = []byte("envs")
	vars     = []byte("vars")
	nodes    = []byte("nodes")

	// unique indexes
	idxEnvs = []byte("ix_envs")
	idxVars = []byte("ix_vars")
)

// Data manages the collection of buckets.
type Data struct {
	db *bolt.DB
}

// Open opens a new connection to the database.
func Open(db string) (*Data, error) {
	if db == "" {
		return nil, ErrMissing
	}
	r, err := bolt.Open(db, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	// Initializes the database by creating the default buckets.
	err = r.Update(func(tx *bolt.Tx) error {
		for _, b := range [][]byte{projects, envs, vars, nodes, idxEnvs, idxVars} {
			if _, err := tx.CreateBucketIfNotExists(b); err != nil {
				return err
			}
		}
		return nil
	})
	return &Data{db: r}, err
}

// Close closes the connection to database.
func (m *Data) Close() error {
	return m.db.Close()
}

// Nodes returns the list of server used as RPC cache.
func (m *Data) Nodes() ([]Keyer, error) {
	return m.all(nodes)
}

// AddNode adds a server as cache or returns on error if already exists.
func (m *Data) AddNode(n *Node) error {
	return m.db.Update(func(tx *bolt.Tx) error {
		return m.put(tx, n, nodes, true)
	})
}

// DeleteNode removes a server.
func (m *Data) DeleteNode(n *Node) error {
	return m.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(nodes).Delete(n.Key())
	})
}

// GetNode returns an error or the node if it exists.
func (m *Data) GetNode(key string) (Keyer, error) {
	d, err := m.get([]byte(key), nodes)
	if err != nil {
		return nil, err
	}
	n, ok := d.(*Node)
	if !ok {
		return nil, ErrInvalid
	}
	return n, err
}

// Projects returns the list of projects.
func (m *Data) Projects() ([]Keyer, error) {
	return m.all(projects)
}

// AddProject creates a project or returns on error if already exists.
func (m *Data) AddProject(p *Project) error {
	return m.db.Update(func(tx *bolt.Tx) error {
		return m.put(tx, p, projects, true)
	})
}

// DeleteProject removes a project.
func (m *Data) DeleteProject(p *Project) error {
	return m.db.Update(func(tx *bolt.Tx) error {
		// @todo Removes env's unique indexes.
		// @todo Removes its vars.
		// @todo Removes var's unique indexes.
		return tx.Bucket(projects).Delete(p.Key())
	})
}

// GetProject returns an error or the project if it exists.
// It contains all its properties, as the list of vars or envs.
func (m *Data) GetProject(key string) (Keyer, error) {
	p, err := m.project(key)
	if err != nil {
		return p, err
	}
	if len(p.EnvList) > 0 {
		keys := sitosb(p.EnvList)
		if p.envs, err = m.list(keys, envs); err != nil {
			return p, err
		}
	}
	if len(p.VarList) > 0 {
		keys := sitosb(p.VarList)
		if p.vars, err = m.list(keys, vars); err != nil {
			return p, err
		}
	}
	return p, err
}

// UpsertProject updates or creates if not exists a project.
func (m *Data) UpsertProject(p *Project) error {
	return m.db.Update(func(tx *bolt.Tx) error {
		return m.put(tx, p, projects, false)
	})
}

// Envs returns the list of available envs and skip those
// to ignore as asked.
func (m *Data) Envs(ignores ...uint64) ([]Keyer, error) {
	all, err := m.all(envs)
	if err != nil {
		return nil, err
	}
	var size int
	if size = len(ignores); size == 0 {
		return all, nil
	}
	skip := make(map[uint64]struct{}, size)
	for _, i := range ignores {
		skip[i] = struct{}{}
	}
	envs := make([]Keyer, 0)
	for _, env := range all {
		if _, ok := skip[env.(*Env).ID]; !ok {
			envs = append(envs, env)
		}
	}
	return envs, nil
}

// GetEnv returns an error or the env if it exists.
func (m *Data) GetEnv(key uint64) (Keyer, error) {
	return m.env(key)
}

// AddEnv creates a env or returns on error if already exists.
func (m *Data) AddEnv(s *Env) error {
	return m.db.Update(func(tx *bolt.Tx) error {
		return m.put(tx, s, envs, true)
	})
}

// BindEnvInProject binds a env to a project
func (m *Data) BindEnvInProject(s *Env, project string) error {
	p, err := m.project(project)
	if err != nil {
		return errors.WithMessage(err, "project")
	}
	// Checks if the env exists in this project.
	ck := p.Hash(s.Name)
	if ok, err := m.exists(ck, idxEnvs); ok {
		return errors.WithMessage(ErrAlreadyExists, "env")
	} else if err != ErrNotFound {
		return errors.WithMessage(err, "env")
	}
	if err = p.AddEnv(s); err != nil {
		return errors.WithMessage(err, "project: env")
	}
	return m.db.Update(func(tx *bolt.Tx) error {
		// Save the project with the new env.
		if err := m.put(tx, p, projects, false); err != nil {
			return errors.WithMessage(err, "project")
		}
		// Updates the unique index.
		return tx.Bucket(idxEnvs).Put(ck, s.Key())
	})
}

// UnbindEnvInProject unbinds a env to a project.
func (m *Data) UnbindEnvInProject(s *Env, project string) error {
	p, err := m.project(project)
	if err != nil {
		return errors.WithMessage(err, "project")
	}
	// Checks if the env name exists for this project.
	ck := p.Hash(s.Name)
	if ok, err := m.exists(ck, idxEnvs); !ok {
		return errors.WithMessage(err, "env")
	}
	if err = p.DeleteEnv(s); err != nil {
		return errors.WithMessage(err, "project: env")
	}
	return m.db.Update(func(tx *bolt.Tx) error {
		// Save the project with the new env.
		if err := m.put(tx, p, projects, false); err != nil {
			return errors.WithMessage(err, "project")
		}
		// Removes the unique index.
		return tx.Bucket(idxEnvs).Delete(ck)
	})
}

// UpsertEnv updates or creates a env if not exists.
func (m *Data) UpsertEnv(s *Env) error {
	return m.db.Update(func(tx *bolt.Tx) error {
		return m.put(tx, s, envs, false)
	})
}

// GetVarInProject returns an error or the var if it exists.
func (m *Data) GetVarInProject(key uint64, project string) (Keyer, error) {
	dp, err := m.GetProject(project)
	if err != nil {
		return nil, errors.WithMessage(err, "project")
	}
	var d *Var
	if d, err = m.variable(key); err != nil {
		return d, errors.WithMessage(err, "var")
	}
	// Ensure to manipulate ok values for current environments.
	p := dp.(*Project)
	if err = d.CleanValues(p.FirstEnv(), p.SecondEnv()); err != nil {
		return nil, errors.WithMessage(err, "var")
	}
	return d, err
}

// AddVarToProject adds a var on a project with its name.
func (m *Data) AddVarInProject(d *Var, project string) error {
	dp, err := m.GetProject(project)
	if err != nil {
		return errors.WithMessage(err, "project")
	}
	// Checks if the var name is already used for this project.
	p := dp.(*Project)
	ck := p.Hash(d.Name)
	if ok, err := m.exists(ck, idxVars); ok {
		return errors.WithMessage(ErrAlreadyExists, "var")
	} else if err != ErrNotFound {
		return errors.WithMessage(err, "var")
	}
	// Ensure to manipulate ok values for current environments.
	if err = d.CleanValues(p.FirstEnv(), p.SecondEnv()); err != nil {
		return errors.WithMessage(err, "var")
	}
	return m.db.Update(func(tx *bolt.Tx) error {
		// Creates the new var.
		if err := m.put(tx, d, vars, true); err != nil {
			return errors.WithMessage(err, "var")
		}
		// Adds this var to the project.
		if err := p.AddVar(d); err != nil {
			return errors.WithMessage(err, "project: var")
		}
		// Saves the project with the new var.
		if err := m.put(tx, p, projects, false); err != nil {
			return errors.WithMessage(err, "project")
		}
		// Updates the unique index.
		return tx.Bucket(idxVars).Put(ck, d.Key())
	})
}

// DeleteVar removes a var.
func (m *Data) DeleteVarInProject(d *Var, project string) error {
	p, err := m.project(project)
	if err != nil {
		return errors.WithMessage(err, "project")
	}
	// Checks if the env name exists for this project.
	ck := p.Hash(d.Name)
	if ok, err := m.exists(ck, idxVars); !ok {
		return errors.WithMessage(err, "var")
	}
	return m.db.Update(func(tx *bolt.Tx) error {
		// Mark as deleted this var.
		d.DeletionTs = time.Now()
		if err := m.put(tx, d, vars, false); err != nil {
			return errors.WithMessage(err, "var")
		}
		// Deletes this var from the project.
		if err := p.DeleteVar(d); err != nil {
			return errors.WithMessage(err, "project: var")
		}
		// Saves the project without this var.
		if err := m.put(tx, p, projects, false); err != nil {
			return errors.WithMessage(err, "project")
		}
		// Removes the unique index.
		return tx.Bucket(idxVars).Delete(ck)
	})
}

// UpdateVarInProject updates a project's var or returns in error if not exists.
func (m *Data) UpdateVarInProject(d *Var, project string) error {
	dp, err := m.GetProject(project)
	if err != nil {
		return errors.WithMessage(err, "project")
	}
	// Checks if the env name exists for this project.
	p := dp.(*Project)
	ck := p.Hash(d.Name)
	if ok, err := m.exists(ck, idxVars); !ok {
		return errors.WithMessage(err, "var")
	}
	// Ensure to manipulate ok values for current environments.
	if err = d.CleanValues(p.FirstEnv(), p.SecondEnv()); err != nil {
		return errors.WithMessage(err, "var")
	}
	return m.db.Update(func(tx *bolt.Tx) error {
		return m.put(tx, d, vars, false)
	})
}

// AutoIncrementer must be implement to manage the kind of primary key.
type AutoIncrementer interface {
	AutoIncrementing() bool
}

// Keyer is implemented by any value that has a Key method,
// which returns the “key” identifier for that value.
type Keyer interface {
	Key() []byte
	SetKey([]byte) error
}

// Validator is implemented to check if the data is correct.
type Validator interface {
	Valid(insert bool) error
}

// Updater is implemented to mark as updated the data.
type Updater interface {
	Updated()
}

// Valuable must be implemented by any value to store.
type Valuable interface {
	AutoIncrementer
	Keyer
	Updater
	Validator
}

// all returns a slice with all data in the given table.
func (m *Data) all(table []byte) (res []Keyer, err error) {
	err = m.db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys.
		b := tx.Bucket(table)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			d, err := newFor(table)
			if err != nil {
				return errors.WithMessage(err, string(table))
			}
			if err := json.Unmarshal(v, d); err != nil {
				return err
			}
			res = append(res, d)
		}
		return nil
	})
	return
}

// exists returns an error if the key does not exist.
// It returns its value otherwise.
func (m *Data) exists(key, table []byte) (bool, error) {
	err := m.db.View(func(tx *bolt.Tx) error {
		if key == nil {
			return ErrMissing
		}
		b := tx.Bucket(table)
		if len(b.Get(key)) > 0 {
			return nil
		}
		return ErrNotFound
	})
	return err == nil, err
}

// get returns from the database the data behind the key in the given table.
func (m *Data) get(key, table []byte) (d Keyer, err error) {
	err = m.db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys
		b := tx.Bucket(table)
		v := b.Get(key)
		if len(v) == 0 {
			return ErrNotFound
		}
		if d, err = newFor(table); err != nil {
			return errors.WithMessage(err, string(table))
		}
		if err := json.Unmarshal(v, d); err != nil {
			return err
		}
		return nil
	})
	return
}

// list returns a slice with all required keys of the given table.
func (m *Data) list(keys [][]byte, table []byte) (res []Keyer, err error) {
	err = m.db.View(func(tx *bolt.Tx) error {
		// Assume bucket exists and has keys.
		b := tx.Bucket(table)
		for _, key := range keys {
			v := b.Get(key)
			d, err := newFor(table)
			if err != nil {
				return errors.WithMessage(err, string(table))
			}
			if err := json.Unmarshal(v, d); err != nil {
				return err
			}
			res = append(res, d)
		}
		return nil
	})
	return
}

// newFor returns a new instance of the required data for a bucket.
func newFor(table []byte) (Keyer, error) {
	if bytes.Equal(table, projects) {
		return &Project{}, nil
	}
	if bytes.Equal(table, envs) {
		return &Env{}, nil
	}
	if bytes.Equal(table, vars) {
		return &Var{}, nil
	}
	if bytes.Equal(table, nodes) {
		return &Node{}, nil
	}
	return nil, errors.WithMessage(ErrUnknown, "new")
}

// put adds or updates the data in the given table by using the given transaction.
// If free is set to true (add data for example),
// it returns in error if the key already exists.
func (m *Data) put(tx *bolt.Tx, d Valuable, table []byte, free bool) error {
	// Checks if all mandatory fields are set.
	if err := d.Valid(free); err != nil {
		return err
	}
	// Retrieve the dedicated bucket.
	b := tx.Bucket(table)
	if free {
		// Sets a key for the data.
		if d.Key() == nil && d.AutoIncrementing() {
			k, err := b.NextSequence()
			if err != nil {
				return err
			}
			d.SetKey(itob(k))
		}
		// Checks if the elements doesn't exist.
		if len(b.Get(d.Key())) > 0 {
			return ErrAlreadyExists
		}
	}
	// Mark the data as updated.
	d.Updated()
	// Encodes to JSON to save it.
	buf, err := json.Marshal(d)
	if err != nil {
		return err
	}
	if err := b.Put(d.Key(), buf); err != nil {
		return err
	}
	return nil
}

// var returns a reference to a var or an error.
func (m *Data) variable(key uint64) (*Var, error) {
	d, err := m.get(itob(key), vars)
	if err != nil {
		return nil, err
	}
	s, ok := d.(*Var)
	if !ok {
		return nil, ErrInvalid
	}
	return s, err
}

// project returns a reference to a project or an error.
func (m *Data) project(key string) (*Project, error) {
	d, err := m.get([]byte(key), projects)
	if err != nil {
		return nil, err
	}
	p, ok := d.(*Project)
	if !ok {
		return nil, ErrInvalid
	}
	return p, err
}

// env returns a reference to a env or an error.
func (m *Data) env(key uint64) (*Env, error) {
	d, err := m.get(itob(key), envs)
	if err != nil {
		return nil, err
	}
	s, ok := d.(*Env)
	if !ok {
		return nil, ErrInvalid
	}
	return s, err
}
