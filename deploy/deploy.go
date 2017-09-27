// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package deploy

import (
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/rvflash/eve/client"

	"golang.org/x/sync/errgroup"
)

// Error message.
var (
	ErrInvalid = errors.New("invalid data")
	ErrMissing = errors.New("nothing to deploy")
)

// Key returns the name of the variable used as key in the cache.
func Key(parts ...string) (k string) {
	for _, v := range parts {
		if v = strings.TrimSpace(v); v == "" {
			continue
		}
		if k != "" {
			k += "_"
		}
		k += v
	}
	return strings.ToUpper(k)
}

// Task maintains counter of change.
type Task struct {
	Add, Del, Upd, NoOp uint64
}

// All returns the sum of tasks to perform.
func (t *Task) All() uint64 {
	return t.Add + t.Del + t.Upd + t.NoOp
}

// PctOfAdd returns the percentage of addition.
func (t *Task) PctOfAdd() uint64 {
	return t.percentOf(t.Add)
}

// PctOfDel returns the percentage of deletion.
func (t *Task) PctOfDel() uint64 {
	return t.percentOf(t.Del)
}

// PctOfUpd returns the percentage of update.
func (t *Task) PctOfUpd() uint64 {
	return t.percentOf(t.Upd)
}

// PctOfNoOp returns the percentage of no operation.
func (t *Task) PctOfNoOp() uint64 {
	return t.percentOf(t.NoOp)
}

func (t *Task) percentOf(i uint64) uint64 {
	if s := t.All(); s > 0 {
		return i * 100 / s
	}
	return 0
}

// Source must be implemented by any source want to be deployed.
// Key returns the identifier of the project.
// EnvsValues returns the values of each environments behind the project.
// ToDeploy gives the list of variable to deploy, with their deploy's name as key.
type Source interface {
	Key() []byte
	EnvsValues() (firstEnvValues, secondEnvValues []string)
	ToDeploy(firstEnvValues, secondEnvValues []string) map[string]interface{}
}

// Release represents a new deployment.
type Release struct {
	ref           Source
	to            []*client.RPC
	env1, env2    []string
	src, dst, dep map[string]interface{}
	task          *Task
	err           error
}

// New returns a new Release.
func New(src Source, server *client.RPC, more ...*client.RPC) *Release {
	servers := make([]*client.RPC, 1)
	servers[0] = server
	for i := 0; i < len(more); i++ {
		if more[i] != nil {
			servers = append(servers, more[i])
		}
	}
	return &Release{ref: src, to: servers, task: &Task{}}
}

// Replicate returns the number of servers used to save the data.
func (d *Release) Replicate() int {
	return len(d.to)
}

// Checkout allows to define until 2 environments.
// The adding's order is important, the first must be
// the first environment defined in the EVE's project.
// It returns an error if the number of environment is unexpected.
func (d *Release) Checkout(envs ...[]string) error {
	// Checks if the values of the source slice matched with those in the reference.
	in := func(src, ref []string) bool {
		m := make(map[string]struct{}, len(ref))
		for _, v := range ref {
			m[v] = struct{}{}
		}
		for _, s := range src {
			if _, ok := m[s]; !ok {
				return false
			}
		}
		return true
	}
	// Gets the environments values of the project.
	env1, env2 := d.ref.EnvsValues()
	size := func(envs [][]string) int {
		var l int
		for _, env := range envs {
			if len(env) > 0 {
				l++
			}
		}
		return l
	}(envs)

	// Calculates the number of environment in entry.
	sizes := func(first, second []string) int {
		firstSize, secondSize := len(first), len(second)
		switch {
		case firstSize == 0, secondSize == 0:
			return 0
		case first[0] == "" && second[0] == "":
			return 0
		case first[0] != "" && second[0] == "":
			return 1
		}
		return 2
	}(env1, env2)

	switch {
	case size < sizes || size > 2:
		return ErrInvalid
	case size == 2:
		if !in(envs[1], env2) {
			return ErrInvalid
		}
		d.env2 = envs[1]
		fallthrough
	case size == 1:
		if !in(envs[0], env1) {
			return ErrInvalid
		}
		d.env1 = envs[0]
		if size == 1 {
			d.env2 = []string{""}
		}
	case size == 0:
		d.env1, d.env2 = []string{""}, []string{""}
	}
	return nil
}

// Diff fetches from and merge data to return only the differences
// according to their values in one of the cache instance.
// For a given key, if it not exists in cache, a nil value is returned.
// If it exists with an other value, its value is returned.
// If it exists with the same value, nothing is returned.
func (d *Release) Diff() map[string]interface{} {
	return d.merge()
}

// Log shows the push's logs.
// For each key, it returns the value before and after the push.
func (d *Release) Log() map[string][2]interface{} {
	if d.err != nil {
		// An error occurred on pushing.
		return nil
	}
	log := make(map[string][2]interface{})
	for k := range d.dep {
		nv, ok := d.src[k]
		if !ok {
			// Data ignored when the push.
			continue
		}
		// In the first position, we have the value before the push,
		// then, in the second, the value after the push.
		v, _ := d.dst[k]
		log[k] = [2]interface{}{v, nv}
	}
	if len(log) == 0 {
		return nil
	}
	return log
}

// FirstEnvValues returns the values of the first environment
// used to checkout the release.
func (d *Release) FirstEnvValues() []string {
	return d.env1
}

// SecondEnvValues returns the values of the second environment
// used to checkout the release.
func (d *Release) SecondEnvValues() []string {
	return d.env2
}

// Push uploads via RPC to the cache servers all the required data
// in one bulk.
// It can take as parameter the exclusive list of variable's names to push.
// This list do not have the project ID or envs names as components.
// It returns on error if the process fails.
func (d *Release) Push(only ...string) error {
	if _ = d.merge(); len(d.src) == 0 {
		return ErrMissing
	}
	if d.rebase(only); len(d.src) == 0 {
		return ErrMissing
	}
	var g errgroup.Group
	for _, server := range d.to {
		c := server
		g.Go(func() error {
			return c.Bulk(d.src)
		})
	}
	d.err = g.Wait()
	return d.err
}

// Status gives various counters about the tasks to do to deploy it.
func (d *Release) Status() *Task {
	_ = d.merge()
	return d.task
}

// Gets from cache the data with same key that the data to deploy.
// Unknown keys in cache are ignored.
func (d *Release) fetch() map[string]interface{} {
	var gap = struct {
		data map[string]interface{}
		mu   sync.Mutex
	}{
		data: make(map[string]interface{}),
	}
	var wg sync.WaitGroup
	for k, v := range d.src {
		wg.Add(1)
		go func(k string, v interface{}) {
			defer wg.Done()
			// we arbitrary choose the first server as data reference.
			cv, found := d.to[0].Lookup(k)
			if !found {
				return
			}
			gap.mu.Lock()
			gap.data[k] = cv
			gap.mu.Unlock()
		}(k, v)
	}
	wg.Wait()

	return gap.data
}

// Merges local with cached data to keep only differences.
func (d *Release) merge() map[string]interface{} {
	if len(d.dep) > 0 {
		return d.dep
	}
	if d.src = d.ref.ToDeploy(d.env1, d.env2); len(d.src) == 0 {
		// No variable in this project for these environments
		return nil
	}
	d.dst = d.fetch()
	d.dep = make(map[string]interface{})
	for k, sv := range d.src {
		dv, ok := d.dst[k]
		switch {
		case !ok:
			d.dep[k] = sv
			d.task.Add++
		case sv == nil:
			d.dep[k] = sv
			d.task.Del++
		case sv != dv:
			d.dep[k] = sv
			d.task.Upd++
		default:
			d.task.NoOp++
		}
	}
	return d.dep
}

// Limits the scope of the push to these variable's name.
func (d *Release) rebase(with []string) {
	if len(with) == 0 {
		return
	}
	// Converts variable names in map of deploy keys.
	only := make(map[string]struct{}, 0)
	for _, name := range with {
		for _, ev1 := range d.env1 {
			for _, ev2 := range d.env2 {
				pid := string(d.ref.Key())
				only[Key(pid, ev1, ev2, name)] = struct{}{}
			}
		}
	}
	// Cleans the source by removing all the data to ignore.
	for k := range d.src {
		if _, ok := only[k]; !ok {
			delete(d.src, k)
		}
	}
}
