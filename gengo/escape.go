// © 2020 the Pea Authors under the MIT license. See AUTHORS for the list of authors.

package gengo

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// escape returns an escaped form of a string suitable for a Go identifier.
// The escape character is _.
// Any byte other than [a-zA-Z0-9] is escaped in the form _XX
// where XX are the hex digits of the byte.
func escape(s string, out *strings.Builder) *strings.Builder {
	for _, r := range s {
		if azAZ09(r) {
			out.WriteRune(r)
			continue
		}
		var bs [utf8.UTFMax]byte
		n := utf8.EncodeRune(bs[:], r)
		for _, b := range bs[:n] {
			out.WriteRune('_')
			out.WriteRune(hexDigit(b >> 4))
			out.WriteRune(hexDigit(b & 0xF))
		}
	}
	return out
}

// unescape returns the unescaped version of a strings escaped by escape().
func unescape(s string) (string, error) {
	var out strings.Builder
	for len(s) > 0 {
		r, w := utf8.DecodeRuneInString(s)
		s = s[w:]
		if r != '_' {
			out.WriteRune(r)
			continue
		}
		d0, w := utf8.DecodeRuneInString(s)
		s = s[w:]
		n0, err := hexVal(d0)
		if err != nil {
			return "", err
		}
		d1, w := utf8.DecodeRuneInString(s)
		s = s[w:]
		n1, err := hexVal(d1)
		if err != nil {
			return "", err
		}
		out.WriteByte(n0<<4 | n1)
	}
	return out.String(), nil
}

func hexVal(d rune) (byte, error) {
	b := byte(d & 0xFF)
	if 'A' <= b && b <= 'F' {
		return b - ('A' - 10), nil
	}
	if '0' <= b && b <= '9' {
		return b - '0', nil
	}
	return 0, fmt.Errorf("bad hex digit: %c", d)
}

func hexDigit(b byte) rune {
	r := rune(b & 0xF)
	if r < 10 {
		return r + '0'
	}
	return r + ('A' - 10)
}

func azAZ09(r rune) bool {
	return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9'
}
