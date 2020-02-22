// Copyright © 2020 The Pea Authors under an MIT-style license.

package basic

import (
	"sort"
)

func cleanUp(f *Fun) {
again:
	propagateDeletes(f)
	rmDeletes(f)
	renumber(f)
	collapseChains(f)
	if deleteEmptyBBlks(f) {
		// We marked more statements as deleted after rmDeletes.
		// We need to repropagate and go again.
		goto again
	}
	renumber(f)
}

func propagateDeletes(f *Fun) {
	ds := findDeletes(f.BBlks)
	for len(ds) > 0 {
		d := ds[len(ds)-1]
		ds = ds[:len(ds)-1]
		if d, ok := d.(*MakeVirt); ok && len(d.Virts) == 1 && d.Virts[0].Block != nil {
			// We are deleting the creation of a block literal.
			// Block literals can only be created in one place,
			// so we can remove the corresponding Fun.
			// It will now be unused.
			d.Virts[0].BBlks = nil
		}
		if d, ok := d.(*Call); ok && d.Fun.Val != nil {
			// We are deleting the call to a module-level variable init Fun.
			// There is only ever one call such a Fun; remove the def.
			d.Fun.BBlks = nil
		}
		for _, u := range d.Uses() {
			u.value().rmUser(d)
			if !u.deleted() && unused(u) {
				ds = deleteValueAndUsers(ds, u)
			}
		}
	}
}

func findDeletes(bs []*BBlk) []Stmt {
	var ds []Stmt
	for _, b := range bs {
		for _, s := range b.Stmts {
			if s.deleted() {
				if term, ok := s.(Term); ok {
					for _, o := range term.Out() {
						o.rmIn(b)
					}
				}
				ds = append(ds, s)
				continue
			}
			if v, ok := s.(Val); ok && unused(v) {
				ds = deleteValueAndUsers(ds, v)
				continue
			}
			if c, ok := s.(*Copy); ok && c.Src == c.Dst {
				ds = append(ds, c)
				continue
			}
		}
	}
	return ds
}

func deleteValueAndUsers(ds []Stmt, v Val) []Stmt {
	v.delete()
	ds = append(ds, v)
	for _, u := range v.Users() {
		u.delete()
		ds = append(ds, u)
	}
	return ds
}

func unused(v Val) bool {
	// Initialization of Allocs is not visible outside the function.
	// So they can be remove if their only uses are initializations.
	// Other Vals can only be removed if they have no uses whatsoever.
	alloc, ok := v.(*Alloc)
	if !ok {
		return len(v.Users()) == 0
	}
	for _, u := range alloc.Users() {
		if !u.storesTo(alloc) {
			return false
		}
	}
	return true
}

func deleteEmptyBBlks(f *Fun) bool {
	changed := false
	sub := makeBBlkMap(len(f.BBlks))
	for _, b := range f.BBlks {
		if len(b.Stmts) == 1 && len(b.Out()) == 1 {
			sub.add(b, b.Out()[0])
		}
	}
	subBBlks(f.BBlks, sub)
	for i, b := range f.BBlks {
		if i == 0 || len(b.In) > 0 {
			continue
		}
		changed = true
		for _, s := range b.Stmts {
			s.delete()
		}
		for _, o := range b.Out() {
			o.rmIn(b)
		}
	}
	return changed
}

func rmDeletes(f *Fun) bool {
	changed := false
	var bi int
	for _, b := range f.BBlks {
		var si int
		for _, s := range b.Stmts {
			if _, ok := s.(*Comment); ok {
				// delete comments.
				continue
			}
			if !s.deleted() {
				b.Stmts[si] = s
				si++
			}
		}
		b.Stmts = b.Stmts[:si]
		if len(b.Stmts) > 0 {
			f.BBlks[bi] = b
			bi++
		}
	}
	f.BBlks = f.BBlks[:bi]
	return changed
}

func collapseChains(f *Fun) {
	i := 1
	for _, b := range f.BBlks[1:] {
		if b.Stmts == nil {
			continue
		}
		for len(b.Out()) == 1 && len(b.Out()[0].In) == 1 {
			o := b.Out()[0]
			b.Stmts = append(b.Stmts[:len(b.Stmts)-1], o.Stmts...)
			for _, oo := range o.Out() {
				oo.rmIn(o)
				oo.addIn(b)
			}
			// Setting o.Stmts=nil marks it as deleted on the next iteration.
			o.Stmts = nil
			o.In = nil
		}
		b.N = i
		f.BBlks[i] = b
		i++
	}
	f.BBlks = f.BBlks[:i]
}

func renumber(f *Fun) {
	var iv int
	for ib, b := range f.BBlks {
		b.N = ib
		for _, s := range b.Stmts {
			if v, ok := s.(Val); ok {
				v.value().n = iv
				iv++
			}
		}
	}
	f.NVals = iv
	for _, b := range f.BBlks {
		sort.Slice(b.In, func(i, j int) bool { return b.In[i].N < b.In[j].N })
	}
}
