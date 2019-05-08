package pea

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func TestImportError(t *testing.T) {
	tests := []checkTest{
		{
			name: "import not found",
			src:  `import "nothing"`,
			err:  "error importing nothing: not found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestRedefError(t *testing.T) {
	tests := []checkTest{
		{
			name: "empty",
			src:  "",
			err:  "",
		},
		{
			name: "no redefinition",
			src:  "Abc{} Xyz{}",
			err:  "",
		},
		{
			name: "type redefined",
			src:  "Xyz{} Xyz{}",
			err:  "Xyz is redefined(.|\n)*previous definition",
		},
		{
			name: "fun redefined",
			src:  "[foo: _ Int32 |] [foo: _ Int32 |]",
			err:  "foo: is redefined(.|\n)*previous definition",
		},
		{
			name: "fun redefined with different param types",
			src:  "[foo: _ Int32 |] [foo: _ String |]",
			err:  "foo: is redefined(.|\n)*previous definition",
		},
		{
			name: "fun with different arity is ok",
			src:  "[foo: _ Int32 |] [foo |] [foo: _ Int32 bar: _ String |]",
			err:  "",
		},
		{
			name: "same-name fun and method are not redefined",
			src:  "[foo: _ Int32 |] String [foo: _ Int32 |]",
			err:  "",
		},
		{
			name: "same-name methods are not redefined",
			src:  "Int [foo: _ Int32 |] String [foo: _ String |]",
			err:  "",
		},
		{
			name: "var redefined",
			src:  "Xyz := [5] Xyz := [6]",
			err:  "Xyz is redefined",
		},
		{
			name: "fun redefines a type",
			src:  "Xyz{} [Xyz |]",
			err:  "Xyz is redefined",
		},
		{
			name: "var redefines a type",
			src:  "Xyz{} Xyz := [5]",
			err:  "Xyz is redefined",
		},
		{
			name: "type redefineds a fun",
			src:  "[Xyz |] Xyz{}",
			err:  "Xyz is redefined",
		},
		{
			name: "type redefineds a var",
			src:  "Xyz := [5] Xyz {}",
			err:  "Xyz is redefined",
		},
		{
			name: "fun redefines a var",
			src:  "Xyz := [5] [Xyz |]",
			err:  "Xyz is redefined",
		},
		{
			name: "var redefines a fun",
			src:  "[Xyz |] Xyz := [5]",
			err:  "Xyz is redefined",
		},
		{
			name: "built-in overridden",
			src:  "Int32 {}",
			err:  "",
		},
		{
			name: "import overridden",
			src:  `import "xyz" Xyz{}`,
			mods: [][2]string{{
				"xyz",
				"Xyz{}",
			}},
			err: "",
		},
		{
			name: "redefined by import",
			src:  `Xyz{} import "xyz"`,
			mods: [][2]string{{
				"xyz",
				"Xyz{}",
			}},
			err: "imported definition xyz Xyz is redefined",
		},
		{
			name: "import redefined by import",
			src:  `import "xyz" import "xyz"`,
			mods: [][2]string{{
				"xyz",
				"Xyz{}",
			}},
			err: "imported definition xyz Xyz is redefined(.|\n)*previous definition imported",
		},
		{
			name: "same def in different submods is ok",
			src: `
				#one Xyz{}
				#two Xyz{}
			`,
			mods: [][2]string{{
				"xyz",
				"Xyz{}",
			}},
			err: "",
		},
		{
			name: "same import in different submods is ok",
			src: `
				#one import "xyz"
				#two import "xyz"
			`,
			mods: [][2]string{{
				"xyz",
				"Xyz{}",
			}},
			err: "",
		},
		{
			name: "redef in a submod",
			src: `
				#one Xyz{}
				#one Xyz{}
			`,
			mods: [][2]string{{
				"xyz",
				"Xyz{}",
			}},
			err: "Xyz is redefined",
		},
		{
			name: "multiple redefs",
			src: `
				Xyz{} Abc{} Xyz{} Abc{}
				Cde{}
				[Cde |]
			`,
			err: "Xyz is redefined(.|\n)*Abc is redefined(.|\n)*Cde is redefined",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCheckFun(t *testing.T) {
	tests := []checkTest{
		{
			name: "redefined param",
			src:  "[foo: x Int bar: x Int |]",
			err:  "x is redefined",
		},
		{
			name: "_ is not redefined",
			src:  "[foo: _ Int bar: _ Int |]",
			err:  "",
		},
		{
			name: "bad parameter type",
			src:  "[foo: _ Undef |]",
			err:  "Undef is undefined",
		},
		{
			name: "bad return type",
			src:  "[foo ^Undef |]",
			err:  "Undef is undefined",
		},
		{
			name: "undef recv type",
			src:  "Undef [foo |]",
			err:  "Undef is undefined",
		},
		{
			name: "non-type receiver",
			src: `
				Var [foo |]
				Var := [5]
			`,
			err: "got variable, expected a type",
		},
		{
			name: "undef recv type constraint",
			src:  "(T Undef) Array [foo |]",
			err:  "Undef is undefined",
		},
		{
			name: "non-type recv type constraint",
			src: `
				(X Var) Array [foo |]
				Var := [5]
			`,
			err: "got variable, expected a type",
		},
		{
			name: "bad recv param count",
			src:  "(T, U) Array [foo |]",
			err:  "parameter count mismatch",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCheckTypeSig(t *testing.T) {
	tests := []checkTest{
		{
			name: "undef constraint",
			src:  "(T Undef) Foo { x: T }",
			err:  "Undef is undefined",
		},
		{
			name: "non-type constraint",
			src: `
				(T Var) Foo { x: T }
				Var := [5]
			`,
			err: "got variable, expected a type",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCheckTypeAlias(t *testing.T) {
	tests := []checkTest{
		{
			name: "ok",
			src: `
				Abc := Int.
				Def := Abc.
				Ghi := Def.
			`,
			err: "",
		},
		{
			name: "undefined",
			src: `
				Abc := Undef.
			`,
			err: "Undef is undefined",
		},
		{
			name: "alias of bad type",
			src: `
				Abc := Def.
				Def := Undef.
			`,
			err: "Undef is undefined",
		},
		{
			name: "2-cycle",
			src: `
				Abc := Def.
				Def := Abc.
			`,
			err: "type alias cycle",
		},
		{
			name: "larger cycle",
			src: `
				Abc := Def.
				Def := Ghi.
				Ghi := Jkl.
				Jkl := Mno.
				Mno := Abc.
			`,
			err: "type alias cycle",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCheckFields(t *testing.T) {
	tests := []checkTest{
		{
			name: "ok",
			src:  "Point { x: Float y: Float }",
			err:  "",
		},
		{
			name: "undefined",
			src:  "Point { x: Undef y: Float }",
			err:  "Undef is undefined",
		},
		{
			name: "field redefined",
			src:  "Point { x: Float x: Float }",
			err:  "field x is redefined",
		},
		{
			name: "non-type",
			src: `
				[ someFunc | ]
				Point { x: someFunc }
			`,
			err: "got function, expected a type",
		},
		/* // TODO: test a built-in non-type error if there is ever a built-in that is not a type.
		{
			name: "built-in non-type",
			src: `
				Point { x: someBuiltin }
			`,
			err: "got function, expected a type",
		},
		*/
		{
			name: "imported non-type",
			src: `
				import "other"
				Point { x: someFunc }
			`,
			mods: [][2]string{
				{"other", "[ someFunc | ]"},
			},
			err: "got function, expected a type",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCheckCases(t *testing.T) {
	tests := []checkTest{
		{
			name: "ok",
			src:  "IntOpt { none, some: Int }",
			err:  "",
		},
		{
			name: "undefined",
			src:  "UndefOpt { none, some: Undef }",
			err:  "Undef is undefined",
		},
		{
			name: "typeless case redefined",
			src:  "Opt { none, none }",
			err:  "case none is redefined",
		},
		{
			name: "typed case redefined",
			src:  "Opt { some: Int, some: Int }",
			err:  "case some is redefined",
		},
		{
			name: "typed and typeless case redefined",
			src:  "Opt { none, none: Int }",
			err:  "case none is redefined",
		},
		{
			name: "case capitalization redefined",
			src:  "Opt { none, None }",
			err:  "case none is redefined",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestCheckVirts(t *testing.T) {
	tests := []checkTest{
		{
			name: "ok",
			src:  "Reader { [readInto: Byte Array ^Int64] }",
			err:  "",
		},
		{
			name: "undefined parameter type",
			src:  "Reader { [readInto: Undef ^Int64] }",
			err:  "Undef is undefined",
		},
		{
			name: "undefined return type",
			src:  "Reader { [readInto: Byte Array ^Undef] }",
			err:  "Undef is undefined",
		},
		{
			name: "method signature redefined",
			src:  "IntSeq { [at: Int ^Int] [at: Int ^Int] }",
			err:  "virtual method at: is redefined",
		},
		{
			name: "not redefined",
			src: `
				IntSeq {
					[at ^Int]
					[at: Int ^Int]
					[at: Int at: Int ^Int]
					[at: Int put: Int]
				}
			`,
			err: "",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

type checkTest struct {
	name  string
	src   string
	mods  [][2]string
	err   string // regexp, "" means no error
	trace bool
}

func (test checkTest) run(t *testing.T) {
	mod, err := parseString(test.src)
	if err != nil {
		t.Fatalf("failed to parse %q: %v", test.src, err)
	}
	opts := []Opt{testImporter(test.mods)}
	if test.trace {
		opts = append(opts, Trace)
	}
	switch errs := Check(mod, opts...); {
	case test.err == "" && len(errs) == 0:
		break // good
	case test.err == "" && len(errs) > 0:
		t.Errorf("got\n%v\nexpected nil", errs)
	case test.err != "" && len(errs) == 0:
		t.Errorf("got nil, expected matching %q", test.err)
	case !regexp.MustCompile(test.err).MatchString(fmt.Sprintf("%v", errs)):
		t.Errorf("got\n%v,\nexpected matching %q", errs, test.err)
	}
}

func testImporter(mods [][2]string) Opt {
	return func(x *state) {
		x.importer = func(name string) (*Mod, error) {
			for _, m := range mods {
				if m[0] != name {
					continue
				}
				p := NewParser(m[0])
				err := p.Parse("", strings.NewReader(m[1]))
				return p.Mod(), err
			}
			return nil, errors.New("not found")
		}
	}
}
