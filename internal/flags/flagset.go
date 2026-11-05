package flags

import "flag"

type FlagSet struct {
	*flag.FlagSet
}

func NewFlagSet(name string, errorHandling flag.ErrorHandling) *FlagSet {
	return &FlagSet{flag.NewFlagSet(name, errorHandling)}
}
