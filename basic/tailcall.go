// © 2020 the Pea Authors under the MIT license. See AUTHORS for the list of authors.

package basic

func rmSelfTailCalls(f *Fun) bool {
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
		for i, s := range b.Stmts {
			if s.deleted() || !selfTailCall(f, b, i, s) {
				continue
			}
			n++
			s.delete()
			call := s.(*Call)
			b0, b1 := splitBBlk(b, i)
			for j, arg := range call.Args {
				var dst Val
				var parm *Parm
				if j >= len(f.Parms) {
					continue
				}
				parm = f.Parms[j]
				dst = findParm(f, b0, parm.Var)
				if load, ok := arg.(*Load); ok && load.Src == dst && readOnly(dst) {
					// Avoid adding stores to unchanged parameters.
					continue
				}
				if parm.Value && !parm.Self {
					// Remove the extra copy made for pass-by-value.
					// The pass-by-value copy is read-only
					// after we remove the function call,
					// so propagating the single store Src
					// is safe without an explicit read-only check.
					copy := singleStore(arg).(*Copy)
					copy.delete()
					arg = copy.Src
				}
				if !parm.Value && SimpleType(arg.Type()) {
					addStore(b, dst, arg)
				} else {
					addCopy(b, dst, arg)
				}
			}
			if len(bblks) == 1 {
				addJmp(b0, b0)
			} else {
				addJmp(b0, bblks[1])
			}

			todo = append([]*BBlk{b1}, todo...)
			b = b0 // added to bblks after the break
			break
		}
		b.N = len(bblks)
		bblks = append(bblks, b)
	}
	f.BBlks = bblks
	return n > 0
}

func readOnly(v Val) bool {
	var def Stmt
	for _, u := range v.Users() {
		if u.storesTo(v) {
			if def != nil {
				return false
			}
			def = u
			continue
		}
		switch u.(type) {
		case *Load:
			continue
		case *Copy:
			continue
		}
		return false
	}
	return true
}

func selfTailCall(f *Fun, b *BBlk, i int, s Stmt) bool {
	call, ok := s.(*Call)
	if !ok || call.Fun != f {
		return false
	}
	next, b, i := nextStmt(b, i)
	switch next1 := next.(type) {
	case *Ret:
		return true
	case *Load:
		if len(next1.Users()) != 1 || len(call.Args) == 0 {
			return false
		}
		if call.Args[len(call.Args)-1] != next1.Src {
			return false
		}
		next2, b, i := nextStmt(b, i)
		store, ok := next2.(*Store)
		if !ok || store.Val != next {
			return false
		}
		arg, ok := store.Dst.(*Arg)
		if !ok || arg.Parm != f.Ret {
			return false
		}
		next3, _, _ := nextStmt(b, i)
		_, ok = next3.(*Ret)
		return ok
	case *Copy:
		// TODO: tail-call elimination for non-simple return types
		return false
	}
	return false
}

func nextStmt(b *BBlk, i int) (Stmt, *BBlk, int) {
	for {
		i++
		if i >= len(b.Stmts) {
			return nil, nil, 0
		}
		s := b.Stmts[i]
		if jmp, ok := s.(*Jmp); ok {
			b = jmp.Dst
			i = -1
			continue
		}
		if _, isComment := s.(*Comment); !isComment && !s.deleted() {
			return s, b, i
		}
	}
}
