// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package caseconv

import (
	"regexp"
	"strings"
)

var camelCase = regexp.MustCompile("(^[^A-Z0-9]*|[A-Z0-9]*)([A-Z0-9][^A-Z]+|$)")

func SnakeCase(s string) string {
	var a []string
	for _, p := range camelCase.FindAllStringSubmatch(s, -1) {
		if p[1] != "" {
			a = append(a, p[1])
		}
		if p[2] != "" {
			a = append(a, p[2])
		}
	}
	return strings.ToLower(strings.Join(a, "_"))
}
