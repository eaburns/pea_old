// Copyright © 2020 The Pea Authors under an MIT-style license.

package basic

import (
	"fmt"
	"strings"
	"testing"

	"github.com/eaburns/pea/ast"
	"github.com/eaburns/pea/types"
)

// Test inlining a function call and then a subsequent block literal value call.
// These are needed to efficiently implement additional control structures
// like ifTrue: and ifFalse, built on ifTrue:ifFalse:, for example.
func TestIfTrue(t *testing.T) {
	// The call to Bool ifTrue in foo should be inlined;
	// the subsequent block literal value call should be inlined;
	// and the remaining code should have no calls,
	// just switch and jmps.
	const src = `
		Meth Bool [ifTrue: f Nil Fun | self ifTrue: f ifFalse: []]
		func [foo ^Int |
			1 < 10 ifTrue: [^3].
			^5
		]
	`
	mod, errs := compile(src)
	if len(errs) > 0 {
		t.Fatalf("failed to compile: %s", errs)
	}
	foo := findTestFunBySelector(mod, "foo")
	if s := foo.String(); strings.Contains(s, "BUG") {
		t.Errorf("foo a bug:\n%s", s)
	}
	if s := foo.String(); strings.Contains(s, "call") {
		t.Errorf("foo contains a call:\n%s\nexpected no call", s)
	}
}

// Test tail-call optimization, inlining a function call,
// and then a subsequent block literal value call.
// These are needed to efficiently implement loops
// like to:do: (a for-loop).
func TestToDo(t *testing.T) {
	// The call to Bool ifTrue in foo should be inlined;
	// the subsequent block literal value call should be inlined;
	// and the remaining code should have no calls,
	// just switch and jmps.
	const src = `
		Meth Int [to: e Int do: f (Int, Nil) Fun |
			self <= e ifTrue: [
				f value: self.
				self + 1 to: e do: f
			] ifFalse: []
		]
		func [foo ^Int |
			i := 0.
			1 to: 10 do: [:_ | i := i + 1].
			^i
		]
	`
	mod, errs := compile(src)
	if len(errs) > 0 {
		t.Fatalf("failed to compile: %s", errs)
	}
	foo := findTestFunBySelector(mod, "foo")
	if s := foo.String(); strings.Contains(s, "BUG") {
		t.Errorf("foo a bug:\n%s", s)
	}
	if s := foo.String(); strings.Contains(s, "call") {
		t.Errorf("foo contains a call:\n%s\nexpected no call", s)
	}
}

// Test that nested block literals are inlined.
func TestInlineNestedBlocks(t *testing.T) {
	// The call to Bool ifTrue in foo should be inlined;
	// the subsequent block literal value call should be inlined;
	// and the remaining code should have no calls,
	// just switch and jmps.
	const src = `
		func [foo ^Int | ^[ [ [ 3 ] value ] value ] value]
	`
	mod, errs := compile(src)
	if len(errs) > 0 {
		t.Fatalf("failed to compile: %s", errs)
	}
	foo := findTestFunBySelector(mod, "foo")
	if s := foo.String(); strings.Contains(s, "BUG") {
		t.Errorf("foo a bug:\n%s", s)
	}
	if s := foo.String(); strings.Contains(s, "call") {
		t.Errorf("foo contains a call:\n%s\nexpected no call", s)
	}
}

// Tests that we don't crash optimizing an And Type with empty-type fields.
func TestEmptyTypeAndField(t *testing.T) {
	const src = `
		Type AndType {x: Int y: Nil z: Nil}
		Func [foo ^AndType | ^{x: 1 y: {} z: {}}]
	`
	compile(src)
}

// Tests that allocs passed to MakeArray for value-type elements do not escape.
func TestMakeArrayValueElemNoEscape(t *testing.T) {
	const src = `
		Func [foo ^String Array | ^{"a"; "b"}]
	`
	mod, errs := compile(src)
	if len(errs) > 0 {
		t.Fatalf("failed to compile: %s", errs)
	}
	foo := findTestFunBySelector(mod, "foo")
	if s := foo.String(); strings.Contains(s, "BUG") {
		t.Errorf("foo a bug:\n%s", s)
	}
	if s := foo.String(); strings.Contains(s, "alloc(String)") {
		t.Errorf("foo contains an alloc(String):\n%s\nexpected only alloca", s)
	}
}

// Tests that allocs passed to MakeArray for reference-type elements do escape.
func TestMakeArrayRefElemEscape(t *testing.T) {
	const src = `
		Func [foo ^String& Array | ^{"a"; "b"}]
	`
	mod, errs := compile(src)
	if len(errs) > 0 {
		t.Fatalf("failed to compile: %s", errs)
	}
	foo := findTestFunBySelector(mod, "foo")
	if s := foo.String(); strings.Contains(s, "BUG") {
		t.Errorf("foo a bug:\n%s", s)
	}
	if s := foo.String(); strings.Contains(s, "alloca(String)") {
		t.Errorf("foo contains an alloca(String):\n%s\nexpected only alloc", s)
	}
}

// Tests that allocs passed to MakeAnd for value-type fields do not escape.
func TestMakeAndValueFieldNoEscape(t *testing.T) {
	const src = `
		Type (X, Y) Pair {x: X y: Y}
		Func [foo ^(String, String) Pair | ^{x: "a" y: "b"}]
	`
	mod, errs := compile(src)
	if len(errs) > 0 {
		t.Fatalf("failed to compile: %s", errs)
	}
	foo := findTestFunBySelector(mod, "foo")
	if s := foo.String(); strings.Contains(s, "BUG") {
		t.Errorf("foo a bug:\n%s", s)
	}
	if s := foo.String(); strings.Contains(s, "alloc(String)") {
		t.Errorf("foo contains an alloc(String):\n%s\nexpected only alloca", s)
	}
}

// Tests that allocs passed to MakeAnd for reference-type fields do escape.
func TestMakeAndRefFieldEscape(t *testing.T) {
	const src = `
		Type (X, Y) Pair {x: X y: Y}
		Func [foo ^(String&, String&) Pair | ^{x: "a" y: "b"}]
	`
	mod, errs := compile(src)
	if len(errs) > 0 {
		t.Fatalf("failed to compile: %s", errs)
	}
	foo := findTestFunBySelector(mod, "foo")
	if s := foo.String(); strings.Contains(s, "BUG") {
		t.Errorf("foo a bug:\n%s", s)
	}
	if s := foo.String(); strings.Contains(s, "alloca(String)") {
		t.Errorf("foo contains an alloca(String):\n%s\nexpected only alloc", s)
	}
}

// Tests that allocs passed to MakeOr for value-type case do not escape.
func TestMakeOrValueCaseNoEscape(t *testing.T) {
	const src = `
		Type T? {none | some: T}
		Func [foo ^String? | ^{some: "a"}]
	`
	mod, errs := compile(src)
	if len(errs) > 0 {
		t.Fatalf("failed to compile: %s", errs)
	}
	foo := findTestFunBySelector(mod, "foo")
	if s := foo.String(); strings.Contains(s, "BUG") {
		t.Errorf("foo a bug:\n%s", s)
	}
	if s := foo.String(); strings.Contains(s, "alloc(String)") {
		t.Errorf("foo contains an alloc(String):\n%s\nexpected only alloca", s)
	}
}

// Tests that allocs passed to MakeOr for reference-type case do escape.
func TestMakeOrRefCaseEscape(t *testing.T) {
	const src = `
		Type T? {none | some: T}
		Func [foo ^String& ? | ^{some: "a"}]
	`
	mod, errs := compile(src)
	if len(errs) > 0 {
		t.Fatalf("failed to compile: %s", errs)
	}
	foo := findTestFunBySelector(mod, "foo")
	if s := foo.String(); strings.Contains(s, "BUG") {
		t.Errorf("foo a bug:\n%s", s)
	}
	if s := foo.String(); strings.Contains(s, "alloca(String)") {
		t.Errorf("foo contains an alloca(String):\n%s\nexpected only alloc", s)
	}
}

// Test that we don't inline a value call on a Fun
// that is the receiver of a method.
// A method can modify its receiver value,
// so receiving a method makes the Fun non-const,
// and should prevent inlining, as we can't determine it's Block.
func TestMethodMarksFunAsNonConst(t *testing.T) {
	// The call to Bool ifTrue in foo should be inlined;
	// the subsequent block literal value call should be inlined;
	// and the remaining code should have no calls,
	// just switch and jmps.
	const src = `
		func [foo ^Int |
			x := [1].
			x changeSelf.
			^x value
		]
		type IntFun := Int Fun.
		meth IntFun [changeSelf |
			// We need to prevent inlining
			// so our Fun actually is
			// the receiver of a call.
			preventInlining.
			self := [2]
		]
		func [preventInlining]
	`
	mod, errs := compile(src)
	if len(errs) > 0 {
		t.Fatalf("failed to compile: %s", errs)
	}
	foo := findTestFunBySelector(mod, "foo")
	if s := foo.String(); strings.Contains(s, "BUG") {
		t.Errorf("foo a bug:\n%s", s)
	}
	if s := foo.String(); !strings.Contains(s, "virt call") {
		t.Errorf("foo contains no virt call:\n%s\nexpected a virt call", s)
	}
}

// Tests that we only inline leaf calls.
func TestInlineOnlyLeaves(t *testing.T) {
	const src = `
		func [leaf ^String | ^"Hello, World"]
		func [internal ^String | ^leaf]
		func [foo ^String | ^internal]
	`
	mod, errs := compile(src)
	if len(errs) > 0 {
		t.Fatalf("failed to compile: %s", errs)
	}
	foo := findTestFunBySelector(mod, "foo")
	if s := foo.String(); strings.Contains(s, "BUG") {
		t.Errorf("foo a bug:\n%s", s)
	}
	if s := foo.String(); !strings.Contains(s, "call") {
		t.Errorf("foo contains no call:\n%s\nexpected a call", s)
	}
}

// Tests that we override the leaf-call restriction for inlining
// in the case that the function has a Fun-type parameter.
// Inlining this may allow further block-literal inlining,
// which can remove expensive virt calls.
func TestInlineFunParmFuncs(t *testing.T) {
	const src = `
		func [leaf: f String Fun ^String | ^f value]
		func [internal: f String Fun ^String | ^leaf: f]
		func [foo ^String | ^internal: ["Hello"]]
	`
	mod, errs := compile(src)
	if len(errs) > 0 {
		t.Fatalf("failed to compile: %s", errs)
	}
	foo := findTestFunBySelector(mod, "foo")
	if s := foo.String(); strings.Contains(s, "BUG") {
		t.Errorf("foo a bug:\n%s", s)
	}
	if s := foo.String(); strings.Contains(s, "call") {
		t.Errorf("foo contains a call:\n%s\nexpected no call", s)
	}
}

func compile(src string) (*Mod, []error) {
	p := ast.NewParser("#test")
	if err := p.Parse("", strings.NewReader(src)); err != nil {
		return nil, []error{err}
	}
	typesMod, errs := types.Check(p.Mod(), types.Config{})
	if len(errs) > 0 {
		return nil, errs
	}
	basicMod := Build(typesMod)
	Optimize(basicMod)
	return basicMod, nil
}

func findTestFunBySelector(mod *Mod, sel string) *Fun {
	for _, fun := range mod.Funs {
		if fun.Block == nil && fun.Fun.Sig.Sel == sel {
			return fun
		}
	}
	panic(fmt.Sprintf("fun %s not found", sel))
}
