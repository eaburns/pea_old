package types

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/pretty"
)

func TestImportError(t *testing.T) {
	tests := []errorTest{
		{
			name: "no import",
			src: `
				import "missing"
			`,
			err: "not found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestPrivate(t *testing.T) {
	tests := []errorTest{
		{
			name: "private type",
			src: `
				import "in"
				type Test := #in Private.
			`,
			imports: [][2]string{
				{"in", `
					type Private {}
				`},
			},
			err: "type #in Private not found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestRedefError(t *testing.T) {
	tests := []errorTest{
		{
			name: "val and val",
			src: `
				val Abc := [5]
				val Abc := [6]
			`,
			err: "Abc redefined",
		},
		{
			name: "val and type",
			src: `
				val Abc := [5]
				type Abc {}
			`,
			err: "Abc redefined",
		},
		{
			name: "val and unary func",
			src: `
				val Abc := [5]
				func [Abc ^Int |]
			`,
			err: "Abc redefined",
		},
		{
			name: "type and val",
			src: `
				type Abc {}
				val Abc := [6]
			`,
			err: "Abc redefined",
		},
		{
			name: "type and type",
			src: `
				type Abc {}
				type Abc {}
			`,
			err: "Abc redefined",
		},
		{
			name: "type and different arity type is OK",
			src: `
				type Abc {}
				type T Abc {}
				type (U, V) Abc {}
			`,
			err: "",
		},
		{
			name: "type and same arity type",
			src: `
				type Abc {}
				type T Abc {}
				type T Abc {}
			`,
			err: "\\(1\\)Abc redefined",
		},
		{
			name: "type and unary func",
			src: `
				type Abc {}
				func [Abc ^Int |]
			`,
			err: "Abc redefined",
		},
		{
			name: "unary func and val",
			src: `
				func [Abc ^Float |]
				val Abc := [6]
			`,
			err: "Abc redefined",
		},
		{
			name: "unary func and type",
			src: `
				func [Abc ^Float |]
				type Abc {}
			`,
			err: "Abc redefined",
		},
		{
			name: "unary func and unary func",
			src: `
				func [Abc ^Float |]
				func [Abc ^Int |]
			`,
			err: "Abc redefined",
		},
		{
			name: "nary func and nary func",
			src: `
				func [foo: _ Int bar: _ Float |]
				func [foo: _ Int bar: _ Float |]
			`,
			err: "foo:bar: redefined",
		},
		{
			name: "nary func and different nary func is OK",
			src: `
				func [foo: _ Int bar: _ Float |]
				func [foo: _ Int bar: _ Float baz: _ String |]
				func [bar: _ Int foo: _ Float |]
			`,
			err: "",
		},
		{
			name: "no redef with imported",
			src: `
				import "xyz"
				Val Abc := [5]
			`,
			imports: [][2]string{
				{
					"xyz",
					`
						Val Abc := [6]
					`,
				},
			},
			err: "",
		},
		{
			name: "method redefinition",
			src: `
				meth Int64 [foo |]
				meth Int64 [foo |]
			`,
			err: "method Int64 foo redefined",
		},
		{
			name: "method redefinition through an alias",
			src: `
				meth Int [foo |]
				meth Int64 [foo |]
			`,
			err: "method Int64 foo redefined",
		},
		{
			name: "method redefinition through multiple aliases",
			src: `
				meth Int [foo |]
				meth Abc [foo |]
				type Abc := Def.
				type Def := Ghi.
				type Ghi := Int64.
			`,
			err: "method Int64 foo redefined",
		},
		{
			name: "binary method redefinition",
			src: `
				meth Int [@@ _ Int |]
				meth Int [@@ _ Int |]
			`,
			err: "method Int64 @@ redefined",
		},
		{
			name: "nary method redefinition",
			src: `
				meth Int [foo: _ String bar: _ Int |]
				meth Int [foo: _ String bar: _ Int |]
			`,
			err: "method Int64 foo:bar: redefined",
		},
		{
			name: "method redefinition with different param types",
			src: `
				meth Int [foo: _ Int bar: _ Float |]
				meth Int [foo: _ String bar: _ Int |]
			`,
			err: "method Int64 foo:bar: redefined",
		},
		{
			name: "method redefinition with receiver type params",
			src: `
				meth T Array [foo: _ String bar: _ Int |]
				meth U Array [foo: _ String bar: _ Int |]
			`,
			err: "method \\(1\\)Array foo:bar: redefined",
		},
		{
			name: "method not redefined when differing receiver",
			src: `
				type V Map {}
				meth V Map [foo |]
				type (K, V) Map {}
				meth (K, V) Map [foo |]
			`,
			err: "",
		},
		{
			name: "method not redefined on same-name type from different imports",
			src: `
				import "foo"
				import "bar"
				meth #foo Abc [baz |]
				meth #bar Abc [baz |]
			`,
			imports: [][2]string{
				{"foo", "Type Abc {}"},
				{"bar", "Type Abc {}"},
			},
			err: "",
		},
		{
			name: "type field",
			src: `
				type Test { a: Int a: Float }
			`,
			err: "field a redefined",
		},
		{
			name: "type case",
			src: `
				type Test { a, a }
			`,
			err: "case a redefined",
		},
		{
			name: "type case:",
			src: `
				type Test { a: Int, a: Float }
			`,
			err: "case a: redefined",
		},
		{
			name: "type case not redefined with case:",
			src: `
				type Test { a, a: Float }
			`,
			err: "",
		},
		{
			name: "type virt unary selector",
			src: `
				type Test { [foo] [foo] }
			`,
			err: "virtual method foo redefined",
		},
		{
			name: "type virt binary selector",
			src: `
				type Test { [* Int] [* String] }
			`,
			err: "virtual method \\* redefined",
		},
		{
			name: "type virt n-ary selector",
			src: `
				type Test { [foo: Int bar: Float] [foo: String bar: Rune] }
			`,
			err: "virtual method foo:bar: redefined",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestAlias(t *testing.T) {
	tests := []errorTest{
		{
			name: "target not found",
			src: `
				type Abc := NotFound.
			`,
			err: "NotFound not found",
		},
		{
			name: "imported not found",
			src: `
				type Abc := #notFound Xyz.
			`,
			err: "#notFound not found",
		},
		{
			name: "imported type not found",
			src: `
				import "xyz"
				type Abc := #xyz NotFound.
			`,
			imports: [][2]string{
				{"xyz", ""},
			},
			err: "NotFound not found",
		},
		{
			name: "arg count mismatch",
			src: `
				type Abc := Int Int.
			`,
			err: "\\(1\\)Int not found",
		},
		{
			name: "no cycle",
			src: `
				type Abc := Int.
			`,
			err: "",
		},
		{
			name: "no cycle, import",
			src: `
				import "xyz"
				type Abc := #xyz Xyz.
			`,
			imports: [][2]string{
				{
					"xyz",
					`
						Type Xyz {}
					`,
				},
			},
			err: "",
		},
		{
			name: "1 cycle",
			src: `
				type Abc := Abc.
			`,
			err: "type alias cycle",
		},
		{
			name: "2 cycle",
			src: `
				type AbcXyz := Abc.
				type Abc := AbcXyz.
			`,
			err: "type alias cycle",
		},
		{
			name: "3 cycle",
			src: `
				type AbcXyz := AbcXyz123.
				type Abc := AbcXyz.
				type AbcXyz123 := Abc.
			`,
			err: "type alias cycle",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestRetError(t *testing.T) {
	tests := []errorTest{
		{
			name: "ok",
			src: `
				func [one ^Int | ^1]
			`,
			err: "",
		},
		{
			name: "return outside function",
			src: `
				val x := [^5]
			`,
			err: "return outside of a function or method",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

func TestAssignError(t *testing.T) {
	tests := []errorTest{
		{
			name: "ok",
			src: `
				Val x := [
					a Int := 1.
					b := 1.
				]
			`,
			err: "",
		},
		{
			name: "count mismatch: no call",
			src: `
				Val x := [
					a, b, c := 1.
				]
			`,
			err: "assignment count mismatch: got 1, want 3",
		},
		{
			name: "count mismatch: too few messages",
			src: `
				Val x := [
					a, b, c := 1 neg, neg
				]
			`,
			err: "assignment count mismatch: got 2, want 3",
		},
		{
			name: "count mismatch: too many messages",
			src: `
				Val x := [
					a, b, c := 1 neg, neg, neg, neg
				]
			`,
			err: "assignment count mismatch: got 4, want 3",
		},
		{
			name: "bad type name",
			src: `
				Val x := [
					a Unknown := 1
				]
			`,
			err: "type Unknown not found",
		},
		{
			name: "bad type name and argument count mismatch",
			src: `
				Val x := [
					a, b Unknown := 1
				]
			`,
			err: "assignment count mismatch: got 1, want 2(.|\n)*type Unknown not found",
		},
	}
	for _, test := range tests {
		t.Run(test.name, test.run)
	}
}

// TestTypeInstSub tests type instantiation substitution.
func TestTypeInstSub(t *testing.T) {
	// The test setup expects src to have an alias type named Test.
	// The expectation is that Test.Alias.Type!=nil
	// and Test.Alias.Type.String()==want.
	tests := []struct {
		name  string
		src   string
		want  string
		trace bool
	}{
		{
			name: "no type vars",
			src: `
				type Test := Int64.
			`,
			want: "type Int64 {}",
		},
		{
			name: "follow alias",
			src: `
				type Test := Abc.
				type Abc := Def.
				type Def := Ghi.
				type Ghi := Int32.
			`,
			want: "type Int32 {}",
		},
		{
			name: "sub type parm",
			src: `
				type Test := Int List.
				type T List { data: T, next: T List ? }
				type T ? { Nil, Some: T }
			`,
			want: "type Int List { data: Int, next: Int List? }",
		},
		{
			name: "sub type parms",
			src: `
				type Test := (Int, String) Pair.
				type (X, Y) Pair { x: X y: Y }
			`,
			want: "type (Int, String) Pair { x: Int y: String }",
		},
		{
			name: "sub only some type parms",
			src: `
				type T Test := (Int, T) Pair.
				type (X, Y) Pair { x: X y: Y }
			`,
			want: "type (Int, T) Pair { x: Int y: T }",
		},
		{
			name: "sub alias",
			src: `
				type Test := Int DifferentArray.
				type T DifferentArray := T Array.
			`,
			want: "type Int Array {}",
		},
		{
			name: "sub fields",
			src: `
				type Test := (Int, String) Pair.
				type (X, Y) Pair { x: X y: Y }
			`,
			want: "type (Int, String) Pair { x: Int y: String }",
		},
		{
			name: "sub cases",
			src: `
				type Test := Int?.
				type T? { none, some: T }
			`,
			want: "type Int? { none, some: Int }",
		},
		{
			name: "sub virts",
			src: `
				type Test := Int Eq.
				type T Eq { [= T& ^Bool] }
			`,
			want: "type Int Eq { [= Int& ^Bool] }",
		},
		{
			name: "recursive type",
			src: `
				type Test := Int List.
				type T List { data: T& next: T List? }
				type T? { none, some: T }
			`,
			want: "type Int List { data: Int& next: Int List? }",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			p := ast.NewParser("#test")
			if err := p.Parse("", strings.NewReader(test.src)); err != nil {
				t.Fatalf("failed to parse source: %s", err)
			}
			cfg := Config{
				Importer: testImporter(nil),
				Trace:    test.trace,
			}
			mod, errs := Check(p.Mod(), cfg)
			if len(errs) > 0 {
				t.Fatalf("failed to check source: %s", errs)
			}
			testType := findTestType(mod, "Test")
			if testType == nil {
				t.Fatalf("no test type")
			}
			if testType.Alias == nil {
				t.Fatalf("test type is not an alias")
			}
			if got := testType.Alias.Type.fullString(); got != test.want {
				t.Errorf("got:\n%s\nwanted:\n%s", got, test.want)
			}
		})
	}
}

// TestTypeInstMemo tests that the same type instances point to the same objects.
func TestTypeInstMemo(t *testing.T) {
	// The test setup expects src to have two alias types, Test0 and Test1.
	// The expectation is that Test0.Alias.Type==Test1.Alias.Type.
	tests := []struct {
		name    string
		src     string
		imports [][2]string
		trace   bool
	}{
		{
			name: "basic types",
			src: `
				type Test0 := Int64.
				type Test1 := Int64.
			`,
		},
		{
			name: "basic follow alias",
			src: `
				type Test0 := Int.
				type Test1 := Int.
			`,
		},
		{
			name: "basic follow different alias",
			src: `
				type Test0 := Int.
				type Test1 := Abc.
				type Abc := Int64.
			`,
		},
		{
			name: "inst built-in type",
			src: `
				type Test0 := Int64 Array.
				type Test1 := Int64 Array.
			`,
		},
		{
			name: "inst built-in type with aliases",
			src: `
				type Test0 := Int Array.
				type Test1 := Abc Array.
				type Abc := Int64.
			`,
		},
		{
			name: "multiple type parms",
			src: `
				type Test0 := (Int64, String) Map.
				type Test1 := (Int64, String) Map.
				type (K, V) Map {}
			`,
		},
		{
			name: "multiple type parms and aliases",
			src: `
				type Test0 := (Int64, String) Map.
				type Test1 := Abc.
				type Abc := (Int, OtherString) Map.
				type OtherString := String.
				type (K, V) Map {}
			`,
		},
		{
			name: "imported type",
			src: `
				import "bar"
				import "foo"
				type Test0 := #bar IntStringMap.
				type Test1 := #foo IntStringMap.
			`,
			imports: [][2]string{
				{"foo", `
					import "map"
					Type IntStringMap := (Int, String) #map Map.
				`},
				{"bar", `
					import "map"
					Type IntStringMap := (Int, String) #map Map.
				`},
				{"map", `
					Type (K, V) Map {}
				`},
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			p := ast.NewParser("#test")
			if err := p.Parse("", strings.NewReader(test.src)); err != nil {
				t.Fatalf("failed to parse source: %s", err)
			}
			cfg := Config{
				Importer: testImporter(test.imports),
				Trace:    test.trace,
			}
			mod, errs := Check(p.Mod(), cfg)
			if len(errs) > 0 {
				t.Fatalf("failed to check source: %s", errs)
			}
			testType0 := findTestType(mod, "Test0")
			if testType0 == nil {
				t.Fatalf("no test type")
			}
			if testType0.Alias == nil {
				t.Fatalf("test type is not an alias")
			}
			testType1 := findTestType(mod, "Test1")
			if testType1 == nil {
				t.Fatalf("no test type")
			}
			if testType1.Alias == nil {
				t.Fatalf("test type is not an alias")
			}
			if testType0.Alias.Type != testType1.Alias.Type {
				t.Logf("Test0=%s\nTest1=%s",
					pretty.String(testType0.Alias.Type),
					pretty.String(testType1.Alias.Type))
				t.Errorf("Test0=%p, Test1=%p",
					testType0.Alias.Type, testType1.Alias.Type)
			}
		})
	}
}

func findTestType(mod *Mod, name string) *Type {
	for _, def := range mod.Defs {
		if typ, ok := def.(*Type); ok && typ.Sig.Name == name {
			return typ
		}
	}
	return nil
}

type errorTest struct {
	name    string
	src     string
	imports [][2]string
	err     string // regexp, "" means no error
	trace   bool
}

func (test errorTest) run(t *testing.T) {
	p := ast.NewParser("#test")
	if err := p.Parse("", strings.NewReader(test.src)); err != nil {
		t.Fatalf("failed to parse source: %s", err)
	}
	cfg := Config{
		Importer: testImporter(test.imports),
		Trace:    test.trace,
	}
	switch _, errs := Check(p.Mod(), cfg); {
	case test.err == "" && len(errs) == 0:
		return
	case test.err == "" && len(errs) > 0:
		t.Errorf("got %v, expected nil", errs)
	case test.err != "" && len(errs) == 0:
		t.Errorf("got nil, expected matching %s", test.err)
	default:
		err := fmt.Sprintf("%v", errs)
		if !regexp.MustCompile(test.err).MatchString(err) {
			t.Errorf("got %v, expected matching %s", errs, test.err)
		}
	}
}

type testImporter [][2]string

func (imports testImporter) Import(cfg Config, path string) ([]Def, error) {
	for i := range imports {
		if imports[i][0] != path {
			continue
		}
		src := imports[i][1]
		p := ast.NewParser(path)
		if err := p.Parse(path, strings.NewReader(src)); err != nil {
			return nil, fmt.Errorf("failed to parse import: %s", err)
		}
		cfg.Trace = false
		mod, errs := Check(p.Mod(), cfg)
		if len(errs) > 0 {
			return nil, fmt.Errorf("failed to check import: %s", errs)
		}
		setMod(path, mod.Defs)
		return mod.Defs, nil
	}
	return nil, errors.New("not found")
}
