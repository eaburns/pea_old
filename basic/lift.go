package basic

func liftAllocs(f *Fun, liftParms bool) bool {
	var n int
	valMap := makeValMap(f.NVals)
	for _, s := range f.BBlks[0].Stmts {
		if s.deleted() {
			continue
		}
		alloc, store := liftableAlloc(f, s, liftParms)
		if alloc == nil || store == nil {
			continue
		}
		n++
		alloc.delete()
		store.delete()
		for _, u := range alloc.Users() {
			if load, ok := u.(*Load); ok {
				load.delete()
				valMap.add(load, store.Val)
			}
		}
	}
	subVals(f.BBlks, valMap)
	return n > 0
}

// liftableAlloc returns the Stmt as an *Alloc and it's only *Store
// if the statement is a liftable Alloc.
// An Alloc is liftable if it allocates a SimpleType,
// and its only users are *Loads and a single *Store
// for which the Alloc is the Dst.
func liftableAlloc(f *Fun, s Stmt, liftParms bool) (*Alloc, *Store) {
	alloc, ok := s.(*Alloc)
	if !ok || !isRefType(alloc) || !SimpleType(refElemType(alloc)) {
		return nil, nil
	}
	if !liftParms && isParm(f, alloc) {
		return nil, nil
	}
	var store *Store
	for _, u := range alloc.Users() {
		switch u := u.(type) {
		case *Store:
			if store != nil || u.Dst != alloc {
				return nil, nil
			}
			store = u
		case *Load:
			break
		default:
			return nil, nil
		}
	}
	return alloc, store
}

func isParm(f *Fun, alloc *Alloc) bool {
	for _, parm := range f.Parms {
		if parm.Var == alloc.Var {
			return true
		}
	}
	return false
}
