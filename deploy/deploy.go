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
type Source interface {
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
		for _, s := range src {
			for _, r := range ref {
				if s != r {
					return false
				}
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

// Push uploads via RPC to the cache servers all the required data in one bulk.
// It returns on error if the process fails.
func (d *Release) Push() error {
	if _ = d.merge(); len(d.src) == 0 {
		return ErrMissing
	}
	var g errgroup.Group
	for _, server := range d.to {
		c := server
		g.Go(func() error {
			return c.Bulk(d.src)
		})
	}
	return g.Wait()
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
