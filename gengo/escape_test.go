// Copyright © 2020 The Pea Authors under an MIT-style license.

package gengo

import (
	"strings"
	"testing"
)

func TestEscape(t *testing.T) {
	tests := []struct {
		str  string
		want string
	}{
		{"", ""},
		{"a", "a"},
		{"noNeedToEscape", "noNeedToEscape"},
		{" ", "_20"},
		{"_", "_5F"},
		{"abc_xyz", "abc_5Fxyz"},
		{"abc/xyz", "abc_2Fxyz"},
		{"☺", "_E2_98_BA"},
		{"こんにちは", "_E3_81_93_E3_82_93_E3_81_AB_E3_81_A1_E3_81_AF"},
	}
	for _, test := range tests {
		got := escape(test.str, new(strings.Builder)).String()
		if got != test.want {
			t.Errorf("escape(%q)=%q, want %q", test.str, got, test.want)
			continue
		}
		got1, err := unescape(got)
		if err != nil {
			t.Errorf("unescape(%q)=_,%v, want %q,nil", got, err, test.str)
			continue
		}
		if got1 != test.str {
			t.Errorf("unescape(%q)=%q, want %q", got, got1, test.str)
			continue
		}
	}
}

func TestUnescapeBadInput(t *testing.T) {
	tests := []string{
		"_",
		"_F",
		"_ff",
		"_GA",
		"_0G",
		"_☺",
	}
	for _, test := range tests {
		if _, err := unescape(test); err == nil {
			t.Errorf("unescape(%s) had no error", test)
		}
	}
}
