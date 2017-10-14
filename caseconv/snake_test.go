// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package caseconv_test

import (
	"testing"

	"github.com/rvflash/eve/caseconv"
)

func TestSnakeCase(t *testing.T) {
	var dt = []struct {
		in, out string
	}{
		{"camelCase", "camel_case"},
		{"rv72", "rv_72"},
		{"RV72", "rv72"},
		{"EVEDuration", "eve_duration"},
	}
	for i, tt := range dt {
		if out := caseconv.SnakeCase(tt.in); out != tt.out {
			t.Errorf("%d. mismatch content: got=%q, exp=%q", i, out, tt.out)
		}
	}
}
