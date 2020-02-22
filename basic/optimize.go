// Copyright © 2020 The Pea Authors under an MIT-style license.

package basic

import (
	"github.com/eaburns/pea/types"
)

// Optimize applies some simple optimizations.
func Optimize(m *Mod) {
	for _, f := range m.Funs {
		optimize(f)
	}
	rmDeletedFuns(m)
}

func rmDeletedFuns(m *Mod) {
	var i int
	for _, f := range m.Funs {
		if (f.Block != nil || f.Val != nil) && f.BBlks == nil {
			continue
		}
		m.Funs[i] = f
		i++
	}
	m.Funs = m.Funs[:i]
}

func optimize(f *Fun) {
	if len(f.BBlks) == 0 {
		f.CanInline = f.BBlks != nil
		return
	}
	var inlinedCall bool
	if inlinedCall = inlineCalls(f); inlinedCall {
		cleanUp(f)
	}
	if inlineBlockLits(f) {
		cleanUp(f)
	}
	// Lift allocs here helps in detecting return value tails.
	// But we don't want to lift param allocs,
	// because rmSelfTailCalls assumes they remain.
	if liftAllocs(f, false) {
		cleanUp(f)
	}
	if rmSelfTailCalls(f) {
		cleanUp(f)
	}
	if liftAllocs(f, true) {
		cleanUp(f)
	}
	moveAllocsToStack(f)
	// Unconditionally do a cleanUp pass at the end
	// to ensure we cleanUp once even if
	// none of the above passes triggered.
	cleanUp(f)
	if f.Block == nil && f.CanFarRet {
		// We may have removed the far ret, so re-scan for it.
		f.CanFarRet = canFarRet(f)
	}
	f.CanInline = canInline(f) && (hasFunParm(f) || !inlinedCall)

}

func hasFunParm(f *Fun) bool {
	for _, p := range f.Parms {
		if p.Type.BuiltIn == types.RefType &&
			p.Type.Args[0].Type.BuiltIn == types.FunType {
			return true
		}
	}
	return false
}

func canFarRet(f *Fun) bool {
	for _, b := range f.BBlks {
		for _, s := range b.Stmts {
			ma, ok := s.(*MakeAnd)
			if !ok || ma.BlockFun == nil {
				continue
			}
			if hasFarRet(ma.BlockFun) || canFarRet(ma.BlockFun) {
				return true
			}
		}
	}
	return false
}

func hasFarRet(f *Fun) bool {
	for _, b := range f.BBlks {
		for _, s := range b.Stmts {
			if r, ok := s.(*Ret); ok && r.Far {
				return true
			}
		}
	}
	return false
}
