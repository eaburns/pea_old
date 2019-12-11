package basic

// moveAllocsToStack sets Stack=true for some Allocs
// that can be statically proven not to escape their frame.
func moveAllocsToStack(f *Fun) bool {
	var n int
	var allocsToMove int
	for {
		var changed bool
		for i, b := range f.BBlks {
			for _, s := range b.Stmts {
				alloc, ok := s.(*Alloc)
				if !ok || alloc.Stack || escapes(alloc) {
					continue
				}
				n++
				alloc.Stack = true
				changed = true
				if i > 0 {
					allocsToMove++
				}
			}
		}
		if !changed {
			break
		}
	}

	for _, b := range f.BBlks[1:] {
		var i int
		if allocsToMove == 0 {
			break
		}
		for _, s := range b.Stmts {
			if alloc, ok := s.(*Alloc); ok && alloc.Stack {
				addAllocToEnd(f.BBlks[0], alloc)
				allocsToMove--
				continue
			}
			b.Stmts[i] = s
			i++
		}
		b.Stmts = b.Stmts[:i]
	}
	return n > 0
}

func addAllocToEnd(b *BBlk, a *Alloc) {
	s := b.Stmts
	n := len(s)
	t := s[n-1]
	s = append(append(s[:n-1], a), t)
	b.Stmts = s
}

func escapes(alloc *Alloc) bool {
	for _, u := range alloc.Users() {
		if u.storesTo(alloc) {
			continue
		}
		switch u := u.(type) {
		case *Load:
			continue
		case *Copy:
			continue
		case *Store:
			if alloc, ok := u.Dst.(*Alloc); ok && alloc.Stack {
				continue
			}
		}
		return true
	}
	return false
}
