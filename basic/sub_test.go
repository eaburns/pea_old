package basic

import (
	"math/big"
	"testing"

	"github.com/eaburns/pea/types"
)

func TestSubVals(t *testing.T) {
	intType := &types.Type{Name: "Int", BuiltIn: types.IntType}
	intRefType := makeTestRefType(intType)
	intArrayRefType := makeTestRefType(makeTestArrayType(intType))
	floatType := &types.Type{Name: "Float", BuiltIn: types.FloatType}
	floatRefType := makeTestRefType(floatType)
	floatArrayRefType := makeTestArrayType(floatType)
	stringType := &types.Type{Name: "String", BuiltIn: types.StringType}
	stringRefType := makeTestRefType(stringType)

	pointType := &types.Type{
		Name:   "Point",
		Fields: []types.Var{{Name: "x"}, {Name: "y"}},
	}
	pointRefType := makeTestRefType(pointType)

	intOptType := &types.Type{
		Name: "?",
		Cases: []types.Var{
			{Name: "none"},
			{Name: "some:", TypeName: &types.TypeName{Name: "Int"}},
		},
	}
	intOptRefType := makeTestRefType(intOptType)

	someVirtRefType := &types.Type{Name: "Virt", BuiltIn: types.RefType}

	tests := []struct {
		name string
		stmt Stmt
		sub  []Val
		want string
	}{
		{
			name: "comment",
			stmt: &Comment{Text: "hello"},
			sub:  nil,
			want: "// hello",
		},
		{
			name: "store no sub",
			stmt: &Store{
				Dst: &Alloc{val: val{n: 0, typ: intRefType}},
				Val: &IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
			},
			sub:  []Val{},
			want: "store($0, $1)",
		},
		{
			name: "store sub",
			stmt: &Store{
				Dst: &Alloc{val: val{n: 0, typ: intRefType}},
				Val: &IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
			},
			sub: []Val{
				&Alloc{val: val{n: 12, typ: intRefType}},
				&FloatLit{val: val{n: 13, typ: floatType}, Val: big.NewFloat(5)},
			},
			want: "store($12, $13)",
		},
		{
			name: "copy no sub",
			stmt: &Copy{
				Dst: &Alloc{val: val{n: 0, typ: intRefType}},
				Src: &Alloc{val: val{n: 1, typ: intRefType}},
			},
			sub:  []Val{},
			want: "copy($0, $1, Int)",
		},
		{
			name: "copy sub",
			stmt: &Copy{
				Dst: &Alloc{val: val{n: 0, typ: intRefType}},
				Src: &Alloc{val: val{n: 1, typ: intRefType}},
			},
			sub: []Val{
				&Alloc{val: val{n: 12, typ: floatRefType}},
				&Alloc{val: val{n: 13, typ: floatRefType}},
			},
			want: "copy($12, $13, Float)",
		},
		{
			name: "make array no sub",
			stmt: &MakeArray{
				Dst: &Alloc{val: val{n: 0, typ: intArrayRefType}},
				Args: []Val{
					&IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
				},
			},
			sub:  []Val{},
			want: "array($0, {$1})",
		},
		{
			name: "make array sub",
			stmt: &MakeArray{
				Dst: &Alloc{val: val{n: 0, typ: intArrayRefType}},
				Args: []Val{
					&IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
				},
			},
			sub: []Val{
				&Alloc{val: val{n: 12, typ: floatArrayRefType}},
				&FloatLit{val: val{n: 13, typ: floatType}, Val: big.NewFloat(5)},
			},
			want: "array($12, {$13})",
		},
		{
			name: "make slice no sub",
			stmt: &MakeSlice{
				Dst:  &Alloc{val: val{n: 0, typ: intArrayRefType}},
				Ary:  &Alloc{val: val{n: 1, typ: intArrayRefType}},
				From: &IntLit{val: val{n: 2, typ: intType}, Val: big.NewInt(5)},
				To:   &IntLit{val: val{n: 3, typ: intType}, Val: big.NewInt(6)},
			},
			sub:  []Val{},
			want: "slice($0, $1[$2:$3])",
		},
		{
			name: "make slice sub",
			stmt: &MakeSlice{
				Dst:  &Alloc{val: val{n: 0, typ: intArrayRefType}},
				Ary:  &Alloc{val: val{n: 1, typ: intArrayRefType}},
				From: &IntLit{val: val{n: 2, typ: intType}, Val: big.NewInt(5)},
				To:   &IntLit{val: val{n: 3, typ: intType}, Val: big.NewInt(6)},
			},
			sub: []Val{
				&Alloc{val: val{n: 14, typ: floatArrayRefType}},
				&Alloc{val: val{n: 15, typ: floatArrayRefType}},
				&FloatLit{val: val{n: 16, typ: floatType}, Val: big.NewFloat(5)},
				&FloatLit{val: val{n: 17, typ: floatType}, Val: big.NewFloat(6)},
			},
			want: "slice($14, $15[$16:$17])",
		},
		{
			name: "make string no sub",
			stmt: &MakeString{
				Dst:  &Alloc{val: val{n: 0, typ: stringRefType}},
				Data: &String{N: 1},
			},
			sub:  []Val{},
			want: "string($0, string1)",
		},
		{
			name: "make string sub",
			stmt: &MakeString{
				Dst:  &Alloc{val: val{n: 0, typ: stringRefType}},
				Data: &String{N: 1},
			},
			sub: []Val{
				&Alloc{val: val{n: 11, typ: stringRefType}},
			},
			want: "string($11, string1)",
		},
		{
			name: "make and no sub",
			stmt: &MakeAnd{
				Dst: &Alloc{val: val{n: 0, typ: pointRefType}},
				Fields: []Val{
					&IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
					&IntLit{val: val{n: 2, typ: intType}, Val: big.NewInt(6)},
				},
			},
			sub:  []Val{},
			want: "and($0, {x: $1 y: $2})",
		},
		{
			name: "make and sub",
			stmt: &MakeAnd{
				Dst: &Alloc{val: val{n: 0, typ: pointRefType}},
				Fields: []Val{
					&IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
					&IntLit{val: val{n: 2, typ: intType}, Val: big.NewInt(6)},
				},
			},
			sub: []Val{
				&Alloc{val: val{n: 13, typ: pointRefType}},
				&IntLit{val: val{n: 14, typ: intType}, Val: big.NewInt(5)},
				&IntLit{val: val{n: 15, typ: intType}, Val: big.NewInt(6)},
			},
			want: "and($13, {x: $14 y: $15})",
		},
		{
			name: "make or no val no sub",
			stmt: &MakeOr{
				Dst:  &Alloc{val: val{n: 0, typ: intOptRefType}},
				Case: 0,
			},
			sub:  []Val{},
			want: "or($0, {0=none})",
		},
		{
			name: "make or no val sub",
			stmt: &MakeOr{
				Dst:  &Alloc{val: val{n: 0, typ: intOptRefType}},
				Case: 0,
			},
			sub: []Val{
				&Alloc{val: val{n: 11, typ: intOptRefType}},
			},
			want: "or($11, {0=none})",
		},
		{
			name: "make or val no sub",
			stmt: &MakeOr{
				Dst:  &Alloc{val: val{n: 0, typ: intOptRefType}},
				Case: 1,
				Val:  &IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
			},
			sub:  []Val{},
			want: "or($0, {1=some: $1})",
		},
		{
			name: "make or val sub",
			stmt: &MakeOr{
				Dst:  &Alloc{val: val{n: 0, typ: intOptRefType}},
				Case: 1,
				Val:  &IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
			},
			sub: []Val{
				&Alloc{val: val{n: 12, typ: intOptRefType}},
				&IntLit{val: val{n: 13, typ: intType}, Val: big.NewInt(5)},
			},
			want: "or($12, {1=some: $13})",
		},
		{
			name: "make virt no sub",
			stmt: &MakeVirt{
				Dst: &Alloc{val: val{n: 0, typ: someVirtRefType}},
				Obj: &Alloc{val: val{n: 1, typ: intRefType}},
			},
			sub:  []Val{},
			want: "virt($0, $1, {})",
		},
		{
			name: "make virt sub",
			stmt: &MakeVirt{
				Dst: &Alloc{val: val{n: 0, typ: someVirtRefType}},
				Obj: &Alloc{val: val{n: 1, typ: intRefType}},
			},
			sub: []Val{
				&Alloc{val: val{n: 12, typ: someVirtRefType}},
				&Alloc{val: val{n: 13, typ: intRefType}},
			},
			want: "virt($12, $13, {})",
		},
		{
			name: "call no sub",
			stmt: &Call{
				Fun: &Fun{N: 0},
				Args: []Val{
					&IntLit{val: val{n: 0, typ: intType}, Val: big.NewInt(5)},
					&IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
				},
			},
			sub:  []Val{},
			want: "call function0($0, $1)",
		},
		{
			name: "call sub",
			stmt: &Call{
				Fun: &Fun{N: 0},
				Args: []Val{
					&IntLit{val: val{n: 0, typ: intType}, Val: big.NewInt(5)},
					&IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
				},
			},
			sub: []Val{
				&IntLit{val: val{n: 12, typ: intType}, Val: big.NewInt(5)},
				&IntLit{val: val{n: 13, typ: intType}, Val: big.NewInt(5)},
			},
			want: "call function0($12, $13)",
		},
		{
			name: "virt call no sub",
			stmt: &VirtCall{
				Self: &Alloc{val: val{n: 0, typ: someVirtRefType}},
				Args: []Val{
					&IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
					&IntLit{val: val{n: 2, typ: intType}, Val: big.NewInt(5)},
				},
			},
			sub:  []Val{},
			want: "virt call $0.0($1, $2)",
		},
		{
			name: "virt call sub",
			stmt: &VirtCall{
				Self: &Alloc{val: val{n: 0, typ: someVirtRefType}},
				Args: []Val{
					&IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
					&IntLit{val: val{n: 2, typ: intType}, Val: big.NewInt(5)},
				},
			},
			sub: []Val{
				&IntLit{val: val{n: 13, typ: intType}, Val: big.NewInt(5)},
				&IntLit{val: val{n: 14, typ: intType}, Val: big.NewInt(5)},
				&IntLit{val: val{n: 15, typ: intType}, Val: big.NewInt(5)},
			},
			want: "virt call $13.0($14, $15)",
		},
		{
			name: "ret",
			stmt: &Ret{},
			sub: []Val{
				&IntLit{val: val{n: 0, typ: intType}, Val: big.NewInt(5)},
				&IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
				&IntLit{val: val{n: 2, typ: intType}, Val: big.NewInt(5)},
			},
			want: "return",
		},
		{
			name: "jmp",
			stmt: &Jmp{Dst: &BBlk{N: 1}},
			sub: []Val{
				&IntLit{val: val{n: 10, typ: intType}, Val: big.NewInt(5)},
				&IntLit{val: val{n: 11, typ: intType}, Val: big.NewInt(5)},
				&IntLit{val: val{n: 12, typ: intType}, Val: big.NewInt(5)},
			},
			want: "jmp 1",
		},
		{
			name: "switch",
			stmt: &Switch{Val: &IntLit{val: val{n: 0, typ: intType}, Val: big.NewInt(5)}},
			sub: []Val{
				&IntLit{val: val{n: 10, typ: intType}, Val: big.NewInt(5)},
			},
			want: "switch $10",
		},
		{
			name: "int lit",
			stmt: &IntLit{val: val{n: 0, typ: intType}, Val: big.NewInt(5)},
			sub: []Val{
				&IntLit{val: val{n: 10, typ: intType}, Val: big.NewInt(6)},
				&IntLit{val: val{n: 11, typ: intType}, Val: big.NewInt(7)},
				&IntLit{val: val{n: 12, typ: intType}, Val: big.NewInt(8)},
			},
			// It doesn't substitute itself, just its Uses, of which there are none.
			want: "5",
		},
		{
			name: "float lit",
			stmt: &FloatLit{val: val{n: 0, typ: floatType}, Val: big.NewFloat(5)},
			sub: []Val{
				&IntLit{val: val{n: 10, typ: intType}, Val: big.NewInt(6)},
				&IntLit{val: val{n: 11, typ: intType}, Val: big.NewInt(7)},
				&IntLit{val: val{n: 12, typ: intType}, Val: big.NewInt(8)},
			},
			want: "5",
		},
		{
			name: "op no sub",
			stmt: &Op{
				Code: PlusOp,
				Args: []Val{
					&IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
					&IntLit{val: val{n: 2, typ: intType}, Val: big.NewInt(5)},
				},
			},
			sub:  []Val{},
			want: "$1 + $2",
		},
		{
			name: "op sub",
			stmt: &Op{
				Code: PlusOp,
				Args: []Val{
					&IntLit{val: val{n: 1, typ: intType}, Val: big.NewInt(5)},
					&IntLit{val: val{n: 2, typ: intType}, Val: big.NewInt(5)},
				},
			},
			sub: []Val{
				&IntLit{val: val{n: 12, typ: intType}, Val: big.NewInt(5)},
				&IntLit{val: val{n: 13, typ: intType}, Val: big.NewInt(6)},
				&IntLit{val: val{n: 14, typ: intType}, Val: big.NewInt(7)},
			},
			want: "$13 + $14",
		},
		{
			name: "load no sub",
			stmt: &Load{
				Src: &Alloc{val: val{n: 1, typ: intRefType}},
			},
			sub:  []Val{},
			want: "load($1)",
		},
		{
			name: "load sub",
			stmt: &Load{
				Src: &Alloc{val: val{n: 1, typ: intRefType}},
			},
			sub: []Val{
				&Alloc{val: val{n: 12, typ: intRefType}},
				&Alloc{val: val{n: 13, typ: intRefType}},
			},
			want: "load($13)",
		},
		{
			name: "alloc",
			stmt: &Alloc{val: val{n: 0, typ: intRefType}},
			sub: []Val{
				&IntLit{val: val{n: 11, typ: intType}, Val: big.NewInt(6)},
				&IntLit{val: val{n: 12, typ: intType}, Val: big.NewInt(7)},
				&IntLit{val: val{n: 13, typ: intType}, Val: big.NewInt(8)},
			},
			want: "alloc(Int)",
		},
		{
			name: "arg",
			stmt: &Arg{Parm: &Parm{Type: intType}},
			sub: []Val{
				&IntLit{val: val{n: 11, typ: intType}, Val: big.NewInt(6)},
				&IntLit{val: val{n: 12, typ: intType}, Val: big.NewInt(7)},
				&IntLit{val: val{n: 13, typ: intType}, Val: big.NewInt(8)},
			},
			want: "arg(0)",
		},
		{
			name: "global",
			stmt: &Global{Val: &types.Val{Var: types.Var{Name: "x"}}},
			sub: []Val{
				&IntLit{val: val{n: 11, typ: intType}, Val: big.NewInt(6)},
				&IntLit{val: val{n: 12, typ: intType}, Val: big.NewInt(7)},
				&IntLit{val: val{n: 13, typ: intType}, Val: big.NewInt(8)},
			},
			want: "global(x)",
		},
		{
			name: "index no sub",
			stmt: &Index{
				Ary:   &Alloc{val: val{n: 1, typ: floatArrayRefType}},
				Index: &IntLit{val: val{n: 2, typ: intType}, Val: big.NewInt(8)},
			},
			sub:  []Val{},
			want: "$1[$2]",
		},
		{
			name: "index sub",
			stmt: &Index{
				Ary:   &Alloc{val: val{n: 1, typ: floatArrayRefType}},
				Index: &IntLit{val: val{n: 2, typ: intType}, Val: big.NewInt(8)},
			},
			sub: []Val{
				&Alloc{val: val{n: 13, typ: floatArrayRefType}},
				&Alloc{val: val{n: 14, typ: floatArrayRefType}},
				&IntLit{val: val{n: 15, typ: intType}, Val: big.NewInt(8)},
			},
			want: "$14[$15]",
		},
		{
			name: "field no sub",
			stmt: &Field{
				Obj: &Alloc{val: val{n: 1, typ: pointRefType}},
			},
			sub:  []Val{},
			want: "$1.0",
		},
		{
			name: "field sub",
			stmt: &Field{
				Obj: &Alloc{val: val{n: 1, typ: pointRefType}},
			},
			sub: []Val{
				&Alloc{val: val{n: 12, typ: floatArrayRefType}},
				&Alloc{val: val{n: 13, typ: pointRefType}},
				&IntLit{val: val{n: 14, typ: intType}, Val: big.NewInt(8)},
			},
			want: "$13.0",
		},
		{
			name: "no sub on nil",
			stmt: &Load{
				Src: &Alloc{val: val{n: 1, typ: intRefType}},
			},
			sub: []Val{
				nil, // 0
				nil, // 1
				nil, // 2
			},
			want: "load($1)",
		},
		{
			name: "multi-hop sub",
			stmt: &Load{
				Src: &Alloc{val: val{n: 1, typ: intRefType}},
			},
			sub: []Val{
				nil,                                     // 0
				&Alloc{val: val{n: 2, typ: intRefType}}, // 1 → 2
				&Alloc{val: val{n: 3, typ: intRefType}}, // 2 → 3
				&Alloc{val: val{n: 4, typ: intRefType}}, // 3 → 4
				nil,                                     // 4
			},
			want: "load($4)",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			max := len(test.sub)
			for _, u := range append(test.sub, test.stmt.Uses()...) {
				if u != nil && u.Num() > max {
					max = u.Num()
				}
			}
			valMap := makeValMap(max + 1)
			for i, v := range test.sub {
				valMap[i] = v
			}
			test.stmt.subVals(valMap)
			got := buildString(test.stmt)
			if got != test.want {
				t.Errorf("got %s, want %s", got, test.want)
			}
		})
	}
}

func makeTestArrayType(typ *types.Type) *types.Type {
	return &types.Type{
		Name:    "Array",
		BuiltIn: types.ArrayType,
		Args: []types.TypeName{{
			Name: typ.Name,
			Type: typ,
		}},
	}
}

func makeTestRefType(typ *types.Type) *types.Type {
	return &types.Type{
		Name:    "&",
		BuiltIn: types.RefType,
		Args: []types.TypeName{{
			Name: typ.Name,
			Type: typ,
		}},
	}
}
