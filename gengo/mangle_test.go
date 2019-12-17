package gengo

import (
	"regexp"
	"testing"
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
			want: []string{"Func_foo"},
		},
		{
			name: "n-ary func ",
			src:  "Func [foo: _ Int bar: _ Int]",
			want: []string{"Func_foo_bar_"},
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
				"Func[0-9]?_1_Int_foo_bar_",
				"Func[0-9]?_1_String_foo_bar_",
				"Func[0-9]?_1_Float_foo_bar_",
			},
		},
		{
			name: "unary meth",
			src:  "Meth Int [foo]",
			want: []string{"Meth_Int_foo"},
		},
		{
			name: "binary op",
			src:  "Meth Int [+ _ Int ^Int]",
			want: []string{"Op_Int_plus"},
		},
		{
			name: "long binary op",
			src:  "Meth Int [--> _ Int ^Int]",
			want: []string{"Op_Int_minus_minus_greater"},
		},
		{
			name: "n-ary meth ",
			src:  "Meth Int [foo: _ Int bar: _ Int]",
			want: []string{"Meth_Int_foo_bar_"},
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
			want: []string{"Meth_1Array_Int_foo_bar_"},
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
				"Meth[0-9]?_Int_1_Int_foo_bar_",
				"Meth[0-9]?_Int_1_String_foo_bar_",
				"Meth[0-9]?_Int_1_Float_foo_bar_",
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
				"Meth[0-9]?_1Array_String_1_Int_foo_bar_",
				"Meth[0-9]?_1Array_Float_1_String_foo_bar_",
				"Meth[0-9]?_1Array_Int_1_Float_foo_bar_",
			},
		},
		{
			name: "op receiver type",
			src: `
				Type T? {none | some: T}
				Meth T? [ifSome: _ (T, Nil) Fun]
				Type T question {t: T}
				Meth T question [ifSome: _ (T, Nil) Fun]
				val _ := [
					x Int? := {none}.
					x ifSome: [:_|].
					y Int question := {t: 3}.
					y ifSome: [:_|].
				]
			`,
			want: []string{
				"Meth_1_question_Int_ifSome_", // op
				"Meth_1question_Int_ifSome_",  // non-op
			},
		},
		{
			name: "op param type",
			src: `
				Type T? {none | some: T}
				Meth T? [ifSome: _ (T, Nil) Fun]
				Func T [foo: _ T]
				val _ := [
					x Int? := {none}.
					foo: x.
				]
			`,
			want: []string{
				"Func_1_1_question_Int_foo_",
			},
		},
		{
			name: "block",
			src: `
				Func [foo ^Int Fun  | ^[3]]
			`,
			want: []string{
				"Func_foo",
				"block[0-9]",
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			mod, errs := compile(test.src)
			if len(errs) > 0 {
				t.Fatalf("failed to compile: %v", errs)
			}
			var got []string
			for _, f := range mod.Funs {
				got = append(got, mangleFun(f))
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
