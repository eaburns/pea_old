// © 2020 the Pea Authors under the MIT license. See AUTHORS for the list of authors.

package basic

import (
	"github.com/eaburns/pea/types"
)

func canInline(f *Fun) bool {
	if f.CanFarRet {
		// We do not inline functions that can far return,
		// so codegen only needs to insert a far return catch
		// in function premables, not internal to a function body.
		return false
	}
	for _, b := range f.BBlks {
		for _, s := range b.Stmts {
			if s.deleted() {
				continue
			}
			switch s.(type) {
			// TODO: disallow inline with VirtCall unless it's a Fun value.
			case *Call:
				return false
			}
		}
	}
	return f.BBlks != nil
}

func inlineCalls(f *Fun) bool {
	if f.Block == nil && f.Fun != nil && f.Fun.Test {
		// Don't inline calls in tests,
		// because we want panics to report line numbers
		// within the body of the test source.
		return false
	}
	// There can be no calls in the 0th bblock,
	// and we need to make sure it's here to copy
	// inlined block0 allocs, so just copy it over now.
	bblks := make([]*BBlk, 0, len(f.BBlks))
	bblks = append(bblks, f.BBlks[0])

	var n int
	todo := f.BBlks[1:]
	for len(todo) > 0 {
		b := todo[0]
		todo = todo[1:]
		for j, s := range b.Stmts {
			if s.deleted() {
				continue
			}
			call, ok := s.(*Call)
			if !ok {
				continue
			}
			if !call.Fun.CanInline {
				continue
			}

			n++
			s.delete()
			b0, b1 := splitBBlk(b, j)
			bs := copyForInline(call.Fun, f, b1, nil, call.Args)
			f.NVals += call.Fun.NVals
			moveAllocs(bblks[0], bs[0])
			addJmp(b0, bs[0])
			todo = append(append(bs, b1), todo...)
			b = b0 // added to bblks after the break
			break
		}
		b.N = len(bblks)
		bblks = append(bblks, b)
	}
	f.BBlks = bblks
	return n > 0
}

func inlineBlockLits(f *Fun) bool {
	// There can be no calls in the 0th bblock,
	// and we need to make sure it's here to copy
	// inlined block0 allocs, so just copy it over now.
	bblks := make([]*BBlk, 0, len(f.BBlks))
	bblks = append(bblks, f.BBlks[0])

	var n int
	todo := f.BBlks[1:]
	for len(todo) > 0 {
		b := todo[0]
		todo = todo[1:]
		for j, s := range b.Stmts {
			if s.deleted() {
				continue
			}
			call, lit, init := inlineableBlockLitVirtCall(s)
			if call == nil {
				continue
			}
			n++
			s.delete()
			b0, b1 := splitBBlk(b, j)
			bs := copyForInline(lit, f, b1, init.Fields, call.Args)
			f.NVals += lit.NVals
			moveAllocs(bblks[0], bs[0])
			addJmp(b0, bs[0])
			todo = append(append(bs, b1), todo...)
			b = b0 // added to bblks after the break
			break
		}
		b.N = len(bblks)
		bblks = append(bblks, b)
	}
	f.BBlks = bblks
	return n > 0
}

// inlineableBlockLitVirtCall returns
// Stmt as a *VirtCall to a block literal's value funciton,
// the corresponding block literal's *Fun,
// and the *MakeAnd constructor of the block's captures
// if the Stmt is a virual call to an inlinable block literal value function.
// Otherwise it returns nil for all three.
func inlineableBlockLitVirtCall(s Stmt) (*VirtCall, *Fun, *MakeAnd) {
	virtCall, ok := s.(*VirtCall)
	if !ok {
		return nil, nil, nil
	}

	// Try to find a single value used to initialize the VirtCall,
	// following chains of Copys if needed.
	val := virtCall.Self
	var makeVirt *MakeVirt
loop:
	// We trace along chains of single-store read-only Copys.
	for {
		switch init := singleStore(val).(type) {
		case *MakeVirt:
			makeVirt = init
			break loop
		case *Copy:
			if !readOnlyBesidesVirtCall(val, virtCall) {
				return nil, nil, nil
			}
			val = init.Src
		default:
			return nil, nil, nil
		}
	}

	blockAlloc, ok := makeVirt.Obj.(*Alloc)
	if !ok || refElemType(blockAlloc).BuiltIn != types.BlockType {
		return nil, nil, nil
	}
	// Checking singleStore without checking read-only
	// is OK here, because MakeAnd of a block literal
	// is never used except to store a MakeVirt.
	blockInit, ok := singleStore(blockAlloc).(*MakeAnd)
	if !ok {
		return nil, nil, nil
	}
	if len(makeVirt.Virts) != 1 {
		panic("impossible")
	}
	return virtCall, makeVirt.Virts[0], blockInit
}

// splitBBlk splits BBlk at statement i and returns the two halves.
// This modifies the input *BBlk, and returns it as the first return value.
func splitBBlk(b0 *BBlk, i int) (*BBlk, *BBlk) {
	b1 := &BBlk{N: b0.N}
	for _, o := range b0.Out() {
		o.rmIn(b0)
		o.addIn(b1)
	}
	b1.Stmts = b0.Stmts[i:]
	b0.Stmts = b0.Stmts[:i:i]
	return b0, b1
}

// copyForInline returns a copy of the src.BBlks to be inlined into dst.
// If caps is non-nil, then src is assumed to be a block literal Fun.
//
// The returned BBlks and their Vals are all fully, internally linked.
//
// Returns are converted to Jmps to bRet.
//
// Args(i) are substituted with the corresponding args[i] Val,
// with the exception that the 0th arg of a block literal.
// This is the capture block, and there is nothing meaningful to substitute.
// It will become disused and removed by later passes.
//
// Loads of a block literal capture are substituted
// with their corresponding value in caps.
//
// The numbers of the returned BBlks are sequential
// beginning with len(src.BBlks).
// The numbers of the returned Vals are sequential
// beginning with dst.NVals.
//
// copyForInline assumes that the src Fun
// BBlks and their Vals are numbered sequentially from 0.
func copyForInline(src, dst *Fun, bRet *BBlk, caps, args []Val) []*BBlk {
	n := dst.NVals
	bblks := copyBBlks(src.BBlks, src.NVals)
	valMap := makeValMap(src.NVals + dst.NVals)
	for _, b := range bblks {
		for i, s := range b.Stmts {
			switch s := s.(type) {
			case *Ret:
				if i != len(b.Stmts)-1 {
					// Deleted Stmts are not copied by copyBBlks,
					// and it is impossible for a non-deleted
					// Ret to be in a position other than final.
					panic("impossible")
				}
				if s.Far {
					// Inlining a block literal into a function
					// changes a far return into a normal return.
					s.Far = dst.Block != nil
					continue
				}
				s.delete()
				addJmp(b, bRet)
			case *Arg:
				s.value().n = n
				n++
				if caps == nil || s.Parm.N > 0 {
					// For block literals we don't substitute
					// the 0th argument, which is the capture block.
					s.delete()
					valMap.add(s, args[s.Parm.N])
				}
			case *Load:
				s.value().n = n
				n++
				if i := captureLoad(s); caps != nil && i >= 0 {
					valMap.add(s, caps[i])
				}
			case Val:
				s.value().n = n
				n++
			}
		}
	}
	subVals(bblks, valMap)
	return bblks
}

// captureLoad returns capture index if this is a capture load or -1.
//
// A capture access is a field access on an object
// that is the self parameter of a Block.
// If called directly after the build-pass,
// this will be an alloc set from arg(0).
func captureLoad(load *Load) int {
	field, ok := load.Src.(*Field)
	if !ok {
		return -1
	}
	if arg, ok := field.Obj.(*Arg); !ok || arg.Parm.N != 0 {
		return -1
	}
	return field.Index
}

func moveAllocs(dst, src *BBlk) {
	term := dst.Stmts[len(dst.Stmts)-1]
	dst.Stmts = dst.Stmts[:len(dst.Stmts)-1]
	var i int
	for _, s := range src.Stmts {
		if alloc, ok := s.(*Alloc); ok && alloc.Stack {
			dst.Stmts = append(dst.Stmts, alloc)
		} else {
			src.Stmts[i] = s
			i++
		}
	}
	src.Stmts = src.Stmts[:i]
	dst.Stmts = append(dst.Stmts, term)
}

// singleStore returns the single initialization Stmt of a Val
// or nil if there is not exactly one initialization.
//
// Note that this function doesn't handle maybe-store cases,
// like if the Val is passed to a function call.
// So care must be taken when using this to ensure that
// the value cannot be changed between its single store
// and whatever other use is under consideration.
func singleStore(v Val) Stmt {
	var def Stmt
	for _, u := range v.Users() {
		if u.storesTo(v) {
			if def != nil {
				return nil
			}
			def = u
		}
	}
	return def
}

func readOnlyBesidesVirtCall(v Val, call *VirtCall) bool {
	var def Stmt
	for _, u := range v.Users() {
		if u.storesTo(v) {
			if def != nil {
				return false
			}
			def = u
			continue
		}
		switch u := u.(type) {
		case *Load:
			continue
		case *Copy:
			continue
		case *VirtCall:
			if u == call {
				continue
			}
		}
		return false
	}
	return true
}

func (*stmt) storesTo(Val) bool           { return false }
func (n *Store) storesTo(v Val) bool      { return n.Dst == v }
func (n *Copy) storesTo(v Val) bool       { return n.Dst == v }
func (n *MakeArray) storesTo(v Val) bool  { return n.Dst == v }
func (n *NewArray) storesTo(v Val) bool   { return n.Dst == v }
func (n *MakeSlice) storesTo(v Val) bool  { return n.Dst == v }
func (n *MakeString) storesTo(v Val) bool { return n.Dst == v }
func (n *NewString) storesTo(v Val) bool  { return n.Dst == v }
func (n *MakeAnd) storesTo(v Val) bool    { return n.Dst == v }
func (n *MakeOr) storesTo(v Val) bool     { return n.Dst == v }
func (n *MakeVirt) storesTo(v Val) bool   { return n.Dst == v }
