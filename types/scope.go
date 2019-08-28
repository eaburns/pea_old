package types

type scope struct {
	*state
	up *scope

	// One of each of the following fields is non-nil.
	univ []Def
	mod  *Mod
	file *file
}

func newUnivScope(x *state) *scope {
	univ, err := x.cfg.Importer.Import(x.cfg, "")
	if err != nil {
		panic(err.Error())
	}
	return &scope{state: x, univ: univ.Defs}
}

func (x *scope) new() *scope {
	return &scope{state: x.state, up: x}
}
