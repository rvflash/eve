// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package deploy_test

import (
	"fmt"
	"reflect"
	"testing"

	"strconv"

	"github.com/rvflash/eve/client"
	"github.com/rvflash/eve/deploy"
	cache "github.com/rvflash/eve/rpc"
)

// rpc is the test's RPC client.
type rpc int

// Call implements the client.Caller interface
func (c rpc) Call(service string, args, reply interface{}) error {
	switch service {
	case "Cache.Bulk":
		switch int(c) {
		case 0:
			*reply.(*bool) = false
		default:
			*reply.(*bool) = true
		}
		return nil
	case "Cache.Get":
		switch args {
		case "0_BOOL":
			reply.(*cache.Item).Value = false
			return nil
		case "0_INT":
			reply.(*cache.Item).Value = 12
			return nil
		case "0_STR":
			reply.(*cache.Item).Value = "rv"
			return nil
		}
		return cache.ErrNotFound
	}
	return nil
}

// Fake RPC client for tests.
var errClient = client.NewRPC(rpc(0))
var rpcClient = client.NewRPC(rpc(1))

// src is the test's source.
type src int

// Key implements the deploy.Source interface.
func (s src) Key() []byte {
	return []byte(strconv.Itoa(int(s)))
}

// EnvsValues implements the deploy.Source interface.
func (s src) EnvsValues() (firstEnvValues, secondEnvValues []string) {
	switch s {
	case twoEnv:
		return []string{"dev"}, []string{"fr", "en"}
	case oneEnv:
		return []string{"dev"}, []string{""}
	case errEnv:
		return nil, nil

	}
	return []string{""}, []string{""}
}

// ToDeploy implements the deploy.Source interface.
func (s src) ToDeploy(firstEnvValues, secondEnvValues []string) map[string]interface{} {
	switch s {
	case twoEnv:
		return map[string]interface{}{"2_DEV_FR_BOOL": true, "2_DEV_FR_FLOAT": 3.14}
	case oneEnv:
		return map[string]interface{}{"1_DEV_BOOL": true, "1_DEV_FLOAT": 3.14}
	case noEnv:
		if len(firstEnvValues) == 0 || len(secondEnvValues) == 0 {
			return nil
		}
		return map[string]interface{}{"0_BOOL": true, "0_FLOAT": 3.14, "0_STR": nil, "0_INT": 12}
	}
	return nil
}

// Sources without, or with one or two environments.
const (
	noEnv src = iota
	oneEnv
	twoEnv
	errEnv
)

func ExampleNew() {
	// Defines the project to deploy and which values of environments to push.
	r := deploy.New(noEnv, rpcClient)
	if err := r.Checkout(); err != nil {
		fmt.Println(err.Error())
		return
	}
	// Gets the list of variables to change.
	for key, value := range r.Diff() {
		fmt.Printf("%v: %s\n", key, value)
	}
	// Gets counters to show differences with cached data.
	task := r.Status()
	fmt.Printf(
		"Add: %d (%d%%), Delete: %d (%d%%), Update: %d (%d%%), No change: %d (%d%%)\n",
		task.Add, task.PctOfAdd(),
		task.Del, task.PctOfDel(),
		task.Upd, task.PctOfDel(),
		task.NoOp, task.PctOfNoOp(),
	)
	// Pushes these changes in production.
	if err := r.Push(); err != nil {
		fmt.Println(err.Error())
	}
}

// TestNew tests the New method to instantiate a new Release.
func TestNew(t *testing.T) {
	// Try with only one server.
	r := deploy.New(noEnv, rpcClient)
	if i := r.Replicate(); i != 1 {
		t.Errorf("server size mismatch: got=%d exp=1", i)
	}
	// Try with more.
	r = deploy.New(noEnv, rpcClient, rpcClient)
	if i := r.Replicate(); i != 2 {
		t.Errorf("server size mismatch: got=%d exp=2", i)
	}
}

// TestRelease_Checkout tests the method Checkout on Release.
func TestRelease_Checkout(t *testing.T) {
	var dt = []struct {
		src      deploy.Source
		dst      *client.RPC
		ev1, ev2 []string
		err      error
	}{
		{src: errEnv, dst: rpcClient},
		{src: noEnv, dst: rpcClient},
		{src: noEnv, dst: rpcClient, ev1: []string{""}, ev2: []string{""}},
		{src: oneEnv, dst: rpcClient, err: deploy.ErrInvalid},
		{src: oneEnv, dst: rpcClient, ev1: []string{"dev"}},
		{src: oneEnv, dst: rpcClient, ev1: []string{"qa"}, ev2: []string{""}, err: deploy.ErrInvalid},
		{src: oneEnv, dst: rpcClient, ev1: []string{"dev"}, ev2: []string{""}},
		{src: twoEnv, dst: rpcClient, err: deploy.ErrInvalid},
		{src: twoEnv, dst: rpcClient, ev1: []string{"dev"}, err: deploy.ErrInvalid},
		{src: twoEnv, dst: rpcClient, ev1: []string{"dev"}, ev2: []string{"dev"}, err: deploy.ErrInvalid},
	}
	for i, tt := range dt {
		r := deploy.New(tt.src, tt.dst)
		if err := r.Checkout(tt.ev1, tt.ev2); !reflect.DeepEqual(err, tt.err) {
			t.Errorf("%d. error mismatch: got=%q exp=%q", i, err, tt.err)
		}
	}
}

// TestRelease_Diff tests all Release methods associated to
// show the difference between local and cached data.
func TestRelease_Diff(t *testing.T) {
	var dt = []struct {
		src      deploy.Source
		dst      *client.RPC
		ev1, ev2 []string
		diff     map[string]interface{}
		task     *deploy.Task
	}{
		{src: errEnv, dst: rpcClient, task: &deploy.Task{}},
		{
			src: noEnv, dst: rpcClient,
			ev1: []string{""}, ev2: []string{""},
			task: &deploy.Task{Add: 1, Del: 1, NoOp: 1, Upd: 1},
			diff: map[string]interface{}{
				"0_BOOL": true, "0_FLOAT": 3.14, "0_STR": nil,
			},
		},
	}
	for i, tt := range dt {
		r := deploy.New(tt.src, tt.dst)
		if err := r.Checkout(tt.ev1, tt.ev2); err != nil {
			t.Fatalf("%d. unexpected error=%q", i, err)
		}
		if diff := r.Diff(); !reflect.DeepEqual(diff, tt.diff) {
			t.Errorf("%d. diff mismatch: got=%q exp=%q", i, diff, tt.diff)
		}
		if task := r.Status(); !reflect.DeepEqual(task, tt.task) {
			t.Errorf("%d. status mismatch: got=%q exp=%q", i, task, tt.task)
		}
	}
}

// TestRelease_Push tests the Push methods on Release.
func TestRelease_Push(t *testing.T) {
	var dt = []struct {
		src       deploy.Source
		dst, more *client.RPC
		ev1, ev2  []string
		skip      []string
		log       map[string][2]interface{}
		err       error
	}{
		{src: errEnv, dst: rpcClient, err: deploy.ErrMissing},
		{
			src: noEnv, dst: rpcClient, more: errClient,
			ev1: []string{""}, ev2: []string{""},
			err: client.ErrFailure,
		},
		{
			src: noEnv, dst: rpcClient,
			ev1: []string{""}, ev2: []string{""},
			log: map[string][2]interface{}{
				"0_STR":   {"rv", nil},
				"0_BOOL":  {false, true},
				"0_FLOAT": {nil, 3.14},
			},
		},
		{
			src: noEnv, dst: rpcClient,
			ev1: []string{""}, ev2: []string{""},
			log: map[string][2]interface{}{
				"0_STR":   {"rv", nil},
				"0_FLOAT": {nil, 3.14},
			},
			skip: []string{"bool"},
		},
		{
			src: noEnv, dst: rpcClient,
			ev1: []string{""}, ev2: []string{""},
			skip: []string{"bool", "str", "float", "int"},
			err:  deploy.ErrMissing,
		},
	}
	for i, tt := range dt {
		r := deploy.New(tt.src, tt.dst, tt.more)
		if err := r.Checkout(tt.ev1, tt.ev2); err != nil {
			t.Fatalf("%d. unexpected error=%q", i, err)
		}
		if err := r.Push(tt.skip...); !reflect.DeepEqual(err, tt.err) {
			t.Errorf("%d. error mismatch: got=%q exp=%q", i, err, tt.err)
		}
		if log := r.Log(); !reflect.DeepEqual(log, tt.log) {
			t.Errorf("%d. log mismatch: got=%v exp=%v", i, log, tt.log)
		}
	}
}

// TestKey tests the Key method.
func TestKey(t *testing.T) {
	var dt = []struct {
		in  []string
		out string
	}{
		{},
		{in: []string{"R", "", "V"}, out: "R_V"},
		{in: []string{"", " r ", " v", ""}, out: "R_V"},
	}
	for i, tt := range dt {
		if out := deploy.Key(tt.in...); out != tt.out {
			t.Errorf("%d. content mismatch: got=%d exp=%d", i, out, tt.out)
		}
	}
}

// TestKey tests the Task methods.
func TestTask(t *testing.T) {
	var dt = []struct {
		task                                 *deploy.Task
		all, pctAdd, pctDel, pctUpd, pctNoOp uint64
	}{
		{task: &deploy.Task{}},
		{
			task: &deploy.Task{Add: 9},
			all:  9, pctAdd: 100,
		},
		{
			task: &deploy.Task{Add: 9, Del: 1},
			all:  10, pctAdd: 90, pctDel: 10,
		},
	}
	for i, tt := range dt {
		if tt.task.All() != tt.all {
			t.Fatalf("%d. content mismatch: got=%d exp=%d", i, tt.task.All(), tt.all)
		}
		if tt.task.PctOfAdd() != tt.pctAdd {
			t.Fatalf("%d. add mismatch: got=%d exp=%d", i, tt.task.PctOfAdd(), tt.pctAdd)
		}
		if tt.task.PctOfDel() != tt.pctDel {
			t.Fatalf("%d. del mismatch: got=%d exp=%d", i, tt.task.PctOfDel(), tt.pctDel)
		}
		if tt.task.PctOfUpd() != tt.pctUpd {
			t.Fatalf("%d. upd mismatch: got=%d exp=%d", i, tt.task.PctOfUpd(), tt.pctUpd)
		}
		if tt.task.PctOfNoOp() != tt.pctNoOp {
			t.Fatalf("%d. noop mismatch: got=%d exp=%d", i, tt.task.PctOfNoOp(), tt.pctNoOp)
		}
	}
}
