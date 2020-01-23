package basic

// copyBBlks returns a copy of the basic blocks.
// The returned blocks and their values are all
// properly, internally linked.
// Deleted statements are not copied.
// The numbers of the returned blocks
// begin sequentially from len(bs0).
// The numbers of the values in the return blocks
// begin sequentially from nval.
//
// copyBBlks assumes that the input BBlks
// and their Vals are numbered sequentially from 0.
func copyBBlks(bs0 []*BBlk, nval int) []*BBlk {
	bs1 := make([]*BBlk, len(bs0))
	bblkMap := makeBBlkMap(2 * len(bs0))
	valMap := makeValMap(2 * nval)
	for i, b0 := range bs0 {
		b1 := new(BBlk)
		b1.In = nil
		b1.N = b0.N + len(bs0)
		b1.Stmts = make([]Stmt, 0, len(b0.Stmts))
		for _, s0 := range b0.Stmts {
			if s0.deleted() {
				continue
			}
			s1 := s0.shallowCopy()
			b1.Stmts = append(b1.Stmts, s1)
			if v, ok := s1.(Val); ok {
				// A following subVals will fix users.
				v.value().users = nil
				v.value().n = nval
				nval++
				valMap.add(s0.(Val), v)
			}
		}
		bs1[i] = b1
		bblkMap.add(b0, b1)
	}
	subVals(bs1, valMap)
	for _, b1 := range bs1 {
		term := b1.Stmts[len(b1.Stmts)-1].(Term)
		term.subBBlk(bblkMap)
		for _, o := range term.Out() {
			o.addIn(b1)
		}
	}
	return bs1
}

func (n Comment) shallowCopy() Stmt    { return &n }
func (n Store) shallowCopy() Stmt      { return &n }
func (n Copy) shallowCopy() Stmt       { return &n }
func (n MakeArray) shallowCopy() Stmt  { return &n }
func (n MakeSlice) shallowCopy() Stmt  { return &n }
func (n MakeString) shallowCopy() Stmt { return &n }

func (n MakeAnd) shallowCopy() Stmt {
	n.Fields = append([]Val{}, n.Fields...)
	return &n
}

func (n MakeOr) shallowCopy() Stmt   { return &n }
func (n MakeVirt) shallowCopy() Stmt { return &n }
func (n Panic) shallowCopy() Stmt    { return &n }

func (n Call) shallowCopy() Stmt {
	n.Args = append([]Val{}, n.Args...)
	return &n
}

func (n VirtCall) shallowCopy() Stmt {
	n.Args = append([]Val{}, n.Args...)
	return &n
}

func (n Ret) shallowCopy() Stmt { return &n }
func (n Jmp) shallowCopy() Stmt { return &n }

func (n Switch) shallowCopy() Stmt {
	n.Dsts = append([]*BBlk{}, n.Dsts...)
	return &n
}

func (n IntLit) shallowCopy() Stmt   { return &n }
func (n FloatLit) shallowCopy() Stmt { return &n }

func (n Op) shallowCopy() Stmt {
	n.Args = append([]Val{}, n.Args...)
	return &n
}

func (n Load) shallowCopy() Stmt   { return &n }
func (n Alloc) shallowCopy() Stmt  { return &n }
func (n Arg) shallowCopy() Stmt    { return &n }
func (n Global) shallowCopy() Stmt { return &n }
func (n Index) shallowCopy() Stmt  { return &n }
func (n Field) shallowCopy() Stmt  { return &n }
