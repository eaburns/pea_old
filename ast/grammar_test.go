// Copyright © 2020 The Pea Authors under an MIT-style license.

package ast

import (
	"strings"
	"testing"
)

func TestCallAssocPrec(t *testing.T) {
	tests := []struct {
		src string
		// want is the paranthesized representation of the 1st statement.
		want string
	}{
		{
			src:  "func [f | a + b + c + d]",
			want: "(((a) + (b)) + (c)) + (d)",
		},
		{
			src:  "func [f | a b c d]",
			want: "(((a) b) c) d",
		},
		{
			src:  "func [f | a b + c d]",
			want: "((a) b) + ((c) d)",
		},
		{
			src:  "func [f | foo: a b + c bar: d]",
			want: "foo: (((a) b) + (c)) bar: (d)",
		},
	}
	for _, test := range tests {
		p := NewParser("")
		if err := p.Parse("", strings.NewReader(test.src)); err != nil {
			t.Errorf("failed to parse [%s]: %s,", test.src, err.Error())
			continue
		}
		stmt := p.Mod().Files[0].Defs[0].(*Fun).Stmts[0]
		var s strings.Builder
		stmt.buildString("", &s)
		if s.String() != test.want {
			t.Errorf("got:\n	%s\nexpected:\n	%s", s.String(), test.want)
			continue
		}
	}
}
