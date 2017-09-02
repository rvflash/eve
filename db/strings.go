// Copyright (c) 2017 Hervé Gouchet. All rights reserved.
// Use of this source code is governed by the MIT License
// that can be found in the LICENSE file.

package	 db

import (
	"regexp"
	"strings"
)

func check(s string) bool {
	ok, _ := regexp.MatchString("^[a-zA-Z0-9_-]+$", s)
	return ok
}

// Returns a lower case version of the given string with only a-z, dash and underscore).
func clean(s string) string {
	// Replaces the spaces by dashes after removing all redundant spaces
	s = strings.Join(strings.Fields(s), " ")
	s = strings.Replace(strings.TrimSpace(s), " ", "-", -1)
	// Removes all the chars not allowed.
	reg, err := regexp.Compile("[^a-z0-9_-]+")
	if err != nil {
		return ""
	}
	return reg.ReplaceAllString(strings.ToLower(s), "")
}

// Removes the duplicates in a slice of string.
func unique(s []string) []string {
	exists := map[string]bool{}
	r := []string{}
	for v := range s {
		if !exists[s[v]] {
			// Record this element as an encountered element.
			exists[s[v]] = true
			r = append(r, s[v])
		}
	}
	return r
}

// Creates a map using string value in the slice as key.
// Useful to check after if a value exists in the slice.
func toMap(s []string) (m map[string]struct{}) {
	m = make(map[string]struct{}, len(s))
	for _, v := range s {
		m[v] = struct{}{}
	}
	return
}
