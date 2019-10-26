package sem

import (
	"fmt"
	"sort"
	"strings"

	"github.com/eaburns/pea/syn"
)

type checkError struct {
	loc   syn.Loc
	msg   string
	notes []string
	cause []checkError
}

func note(err *checkError, f string, vs ...interface{}) {
	err.notes = append(err.notes, fmt.Sprintf(f, vs...))
}

func (err *checkError) Error() string {
	var s strings.Builder
	buildError(&s, "", err)
	return s.String()
}

func buildError(s *strings.Builder, ident string, err *checkError) {
	s.WriteString(ident)
	s.WriteString(err.loc.String())
	s.WriteString(": ")
	s.WriteString(err.msg)
	ident2 := ident + "	"
	for _, n := range err.notes {
		s.WriteRune('\n')
		s.WriteString(ident2)
		s.WriteString(n)
	}
	for i := range err.cause {
		s.WriteRune('\n')
		buildError(s, ident2, &err.cause[i])
	}
}

func convertErrors(cerrs []checkError) []error {
	var errs []error
	for i := range sortErrors(cerrs) {
		errs = append(errs, &cerrs[i])
	}
	return errs
}

func sortErrors(errs []checkError) []checkError {
	if len(errs) == 0 {
		return errs
	}
	sort.Slice(errs, func(i, j int) bool {
		switch ei, ej := errs[i].loc, &errs[j].loc; {
		case ei.Path == ej.Path && ei.Line[0] == ej.Line[0]:
			return ei.Col[0] < ej.Col[0]
		case ei.Path == ej.Path:
			return ei.Line[0] < ej.Line[0]
		default:
			return ei.Path < ej.Path
		}
	})
	dedup := []checkError{errs[0]}
	for _, e := range errs[1:] {
		d := &dedup[len(dedup)-1]
		if e.loc != d.loc || e.msg != d.msg {
			dedup = append(dedup, e)
		}
	}
	for i := range dedup {
		dedup[i].cause = sortErrors(dedup[i].cause)
	}
	return dedup
}
