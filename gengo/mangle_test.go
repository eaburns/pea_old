// Copyright © 2020 The Pea Authors under an MIT-style license.

package gengo

import (
	"regexp"
	"strings"
	"testing"

	"github.com/eaburns/pea/types"
)

func TestMangleFun(t *testing.T) {
	tests := []struct {
		name string
		src  string
		// regexp of the mangle of all fun insts in src
		want []string
	}{
		{
			name: "0-ary func",
			src:  "Func [foo]",
			want: []string{"F0_main__foo__"},
		},
		{
			name: "n-ary func ",
			src:  "Func [foo: _ Int bar: _ Int]",
			want: []string{"F0_main__foo_3Abar_3A__"},
		},
		{
			name: "type param func",
			src: `
				Func T [foo: _ T bar: _ T]
				val _ := [
					foo: 1 bar: 3.
					foo: "s" bar: "t".
					foo: 3.14 bar: 3.4.
				]
			`,
			want: []string{
				"F1___0_Int__main__foo_3Abar_3A__",
				"F1___0_String__main__foo_3Abar_3A__",
				"F1___0_Float__main__foo_3Abar_3A__",
			},
		},
		{
			name: "unary meth",
			src:  "Meth Int [foo]",
			want: []string{"M__0_Int__0_main__foo__"},
		},
		{
			name: "binary op",
			src:  "Meth Int [+ _ Int ^Int]",
			want: []string{"M__0_Int__0_main___2B__"},
		},
		{
			name: "long binary op",
			src:  "Meth Int [--> _ Int ^Int]",
			want: []string{"M__0_Int__0_main___2D_2D_3E__"},
		},
		{
			name: "n-ary meth ",
			src:  "Meth Int [foo: _ Int bar: _ Int]",
			want: []string{"M__0_Int__0_main__foo_3Abar_3A__"},
		},
		{
			name: "param receiver type",
			src: `
				Meth _ Array [foo: _ Int bar: _ Int]
				val _ := [
					x Int Array := {}.
					x foo: 1 bar: 2.
				]
			`,
			want: []string{"M__1_Array____0_Int__0_main__foo_3Abar_3A__"},
		},
		{
			name: "type param method",
			src: `
				Meth Int T [foo: _ T bar: _ T]
				val _ := [
					1 foo: 1 bar: 2.
					1 foo: "s" bar: "t".
					1 foo: 1.3 bar: 2.
				]
			`,
			want: []string{
				"M__0_Int__1___0_Int__main__foo_3Abar_3A__",
				"M__0_Int__1___0_String__main__foo_3Abar_3A__",
				"M__0_Int__1___0_Float__main__foo_3Abar_3A__",
			},
		},
		{
			name: "parm receiver and type param method",
			src: `
				Meth _ Array T [foo: _ T bar: _ T]
				val _ := [
					x String Array := {}.
					x foo: 1 bar: 2.
					y Float Array := {}.
					y foo: "s" bar: "t".
					z Int Array := {}.
					z foo: 1.3 bar: 2.
				]
			`,
			want: []string{
				"M__1_Array____0_String__1___0_Int__main__foo_3Abar_3A__",
				"M__1_Array____0_Float__1___0_String__main__foo_3Abar_3A__",
				"M__1_Array____0_Int__1___0_Float__main__foo_3Abar_3A__",
			},
		},
		{
			name: "block",
			src: `
				Func [foo ^Int Fun  | ^[3]]
			`,
			want: []string{
				"F0_main__foo__",
				"main_____24Block[0-9]__",
			},
		},
		{
			name: "test",
			src: `
				test [foo | 1]
			`,
			want: []string{
				"T0_main__foo",
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if strings.HasPrefix(test.name, "SKIP") {
				t.Skip()
			}
			mod, errs := compile("main", test.src)
			if len(errs) > 0 {
				t.Fatalf("failed to compile: %v", errs)
			}
			var got []string
			for _, f := range mod.Funs {
				m := mangleFun(f, new(strings.Builder)).String()
				got = append(got, m)
			}
			t.Log(got)
			for _, want := range test.want {
				re := regexp.MustCompile(want)
				var ok bool
				for i := range got {
					if re.MatchString(got[i]) {
						got[i] = ""
						ok = true
						break
					}
				}
				if !ok {
					t.Errorf("%s did not match", want)
				}
			}
		})
	}
}

func TestMangleTypesFun(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{
			src: `
				val test := [true]
			`,
			want: "Func true",
		},
		{
			src: `
				val test := [foo]
				func [foo]
			`,
			want: "Func /test/test foo",
		},
		{
			src: `
				val test := [foo: 5 bar: 6]
				func [foo: _ Int bar: _ Int]
			`,
			want: "Func /test/test foo:bar:",
		},
		{
			src: `
				val test := [5 foo]
				meth Int [foo]
			`,
			want: "Meth Int /test/test foo",
		},
		{
			src: `
				val test := [5 -- 6]
				meth Int [-- _ Int]
			`,
			want: "Meth Int /test/test --",
		},
		{
			src: `
				val test := [4 foo: 5 bar: 6]
				meth Int [foo: _ Int bar: _ Int]
			`,
			want: "Meth Int /test/test foo:bar:",
		},
		{
			src: `
				val test := [mytype foo]
				meth MyType [foo]
				type MyType {}
				val mytype MyType := [{}]
			`,
			want: "Meth /test/test MyType /test/test foo",
		},
		{
			src: `
				val test := [mytype -- mytype]
				meth MyType [-- _ MyType]
				type MyType {}
				val mytype MyType := [{}]
			`,
			want: "Meth /test/test MyType /test/test --",
		},
		{
			src: `
				val test := [mytype foo: 5 bar: 6]
				meth MyType [foo: _ Int bar: _ Int]
				type MyType {}
				val mytype MyType := [{}]
			`,
			want: "Meth /test/test MyType /test/test foo:bar:",
		},
		{
			src: `
				val test := [foo: 5 bar: 6]
				func T [foo: _ T bar: _ T]
			`,
			want: "Func Int /test/test foo:bar:",
		},
		{
			src: `
				val test := [foo: 5 bar: 3.14]
				func (T, U) [foo: _ T bar: _ U]
			`,
			want: "Func (Int, Float) /test/test foo:bar:",
		},
		{
			src: `
				val test := [intArray foo: 5 bar: 6]
				meth T Array [foo: _ T bar: _ Int]
				val intArray Int Array := [{}]
			`,
			want: "Meth Int Array /test/test foo:bar:",
		},
		{
			src: `
				val test := [intArray foo: 5 bar: 6]
				meth T Array U [foo: _ T bar: _ U]
				val intArray Int Array := [{}]
			`,
			want: "Meth Int Array Int /test/test foo:bar:",
		},
		{
			src: `
				val test String := [intArray foo: 5 bar: 6]
				meth T Array (U, V) [foo: _ T bar: _ U ^V]
				val intArray Int Array := [{}]
			`,
			want: "Meth Int Array (Int, String) /test/test foo:bar:",
		},
		{
			src: `
				val test := [intArray foo]
				meth (_ Fooer) Array [foo]
				type Fooer {[foo]}
				meth Int [foo]
				val intArray Int Array := [{}]
			`,
			want: "Meth Int Array /test/test foo 0 /test/test",
		},
		{
			src: `
				val test := [foo: 3]
				func (T Fooer) [foo: _ T]
				type Fooer {[foo]}
				meth Int [foo]
			`,
			want: "Func Int /test/test foo: 0 /test/test",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.want, func(t *testing.T) {
			t.Parallel()
			mod, errs := check("/test/test", test.src)
			if len(errs) > 0 {
				t.Fatalf("failed to check: %v", errs)
			}
			v := findTestVal(mod, "test")
			if v == nil {
				t.Fatalf("no Val named test")
			}
			f := v.Init[0].(*types.Call).Msgs[0].Fun

			var s strings.Builder
			mangleTypesFun("/test/test", f, &s)
			m := s.String()
			if strings.IndexFunc(m, badRune) >= 0 {
				t.Errorf("mangle %q has non-[_a-zA-Z0-9] character", m)
			}

			u, err := demangleFun(strings.NewReader(m))
			if err != nil {
				t.Fatalf("demangleType(%q)=_,%v, want no error", m, err)
			}
			if u != test.want {
				t.Errorf("%s demangled to %s, want %s", f, u, test.want)
			}
		})
	}
}

func TestMangleType(t *testing.T) {
	tests := []struct {
		src  string
		want string
	}{
		{
			src:  "val test Int := [5]",
			want: "Int",
		},
		{
			src:  "val test Int Array := [{}]",
			want: "Int Array",
		},
		{
			src: `
				val test Point := [{}]
				type Point {}
			`,
			want: "/test/test Point",
		},
		{
			src: `
				val test Int ? := [{}]
				type _ ? {}
			`,
			want: "Int /test/test ?",
		},
		{
			src: `
				val test Int ? ? := [{}]
				type _ ? {}
			`,
			want: "Int /test/test ? /test/test ?",
		},
		{
			src: `
				val test (Int, String) Pair := [{}]
				type (_, _) Pair {}
			`,
			want: "(Int, String) /test/test Pair",
		},
		{
			src: `
				val test ((Int, String) Pair, String) Pair := [{}]
				type (_, _) Pair {}
			`,
			want: "((Int, String) /test/test Pair, String) /test/test Pair",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.want, func(t *testing.T) {
			t.Parallel()
			mod, errs := check("/test/test", test.src)
			if len(errs) > 0 {
				t.Fatalf("failed to check: %v", errs)
			}
			v := findTestVal(mod, "test")
			if v == nil {
				t.Fatalf("no Val named test")
			}
			typ := v.Var.Type()

			m := mangleType(typ, new(strings.Builder)).String()
			if strings.IndexFunc(m, badRune) >= 0 {
				t.Errorf("mangle %q has non-[_a-zA-Z0-9] character", m)
			}

			u, err := demangleType(strings.NewReader(m))
			if err != nil {
				t.Fatalf("demangleType(%q)=_,%v, want no error", m, err)
			}
			if u != test.want {
				t.Errorf("%s demangled to %s, want %s", typ, u, test.want)
			}
		})
	}
}

func badRune(r rune) bool {
	return r != '_' && !azAZ09(r)
}

func findTestVal(mod *types.Mod, name string) *types.Val {
	for _, def := range mod.Defs {
		if v, ok := def.(*types.Val); ok && v.Var.Name == name {
			return v
		}
	}
	return nil
}

func TestMangleMod(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"", "__"},
		{"abc", "abc__"},
		{"/test/test", "_2Ftest_2Ftest__"},
	}
	for _, test := range tests {
		got0 := mangleMod(test.path, new(strings.Builder)).String()
		if got0 != test.want {
			t.Errorf("mangleMod(%q)=%q, want %q", test.path, got0, test.want)
			continue
		}

		got1, err := demangleMod(strings.NewReader(got0))
		if err != nil {
			t.Errorf("demangleMod(%q)=_,%v, want no error", got0, err)
			continue
		}
		if got1 != test.path {
			t.Errorf("demangleMod(%q)=%q, want %q", got0, got1, test.path)
			continue
		}
	}
}

func TestDeangleTestName(t *testing.T) {
	tests := []struct {
		mangle string
		mod    string
		name   string
		err    string
	}{
		{mangle: "T0_main__foo__", mod: "main", name: "foo"},
		{mangle: "T0_main__fooBar__", mod: "main", name: "fooBar"},
		{mangle: "T0_baz__fooBar__", mod: "baz", name: "fooBar"},
		{mangle: "F0_main__fooBar__", err: "expected 'T'"},
		{mangle: "T1_main__fooBar__", err: "expected 0 args"},
	}
	for _, test := range tests {
		mod, name, err := demangleTestName(test.mangle)
		switch {
		case test.err == "" && err == nil && (name != test.name || mod != test.mod):
			t.Errorf("demangleTestName(%q)=%q, %q, want %q, %q", test.mangle, mod, name, test.mod, test.name)
		case test.err == "" && err != nil:
			t.Errorf("demangleTestName(%q)=_,%v, want nil", test.mangle, err)
		case test.err != "" && err.Error() != test.err:
			t.Errorf("demangleTestName(%q)=_,%v, want _,%s", test.mangle, err, test.err)
		}
	}
}
