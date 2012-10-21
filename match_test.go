// © 2012 Steve McCoy. Available under the MIT License.

package main

import (
	"testing"
)

func TestClean(t *testing.T) {
	tests := []struct {
		s, c string
	}{
		{"Bob Dylan", "Bob Dylan"},
		{"Bob Dylan & The Band", "Bob Dylan  The Band"},
		{"AC/DC", "ACDC"},
	}

	for _, test := range tests {
		c := clean(test.s)
		if c != test.c {
			t.Error("clean(", test.s, `) should be "`, test.c, `", but got`, c)
		}
	}
}

func TestMatch(t *testing.T) {
	tests := []struct {
		pattern, s string
		score      int
	}{
		{"The Who", "The Who", 0},
		{"acdc", "AC/DC", 0},
		{"bob dylan", "Bob Dylan", 0},
		{"bob dylan", "Bob Dylan & The Band", 10},
		{"the band", "Bob Dylan & The Band", 11},
	}

	for _, test := range tests {
		s := match(test.pattern, test.s)
		if s != test.score {
			t.Error("Score for match(", test.pattern, ",", test.s, ") should be", test.score, ", but got", s)
		}
	}
}
