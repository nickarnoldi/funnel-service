package flags

import (
	"flag"
	"fmt"
)

func Int64Slice(flagSet *flag.FlagSet, flagName string, flagDescription string) *FlagInt64Slice {
	var ids FlagInt64Slice
	flagSet.Var(&ids, flagName, flagDescription)
	return &ids
}

func (flagSet *FlagSet) Int64Slice(flagName string, flagDescription string) *FlagInt64Slice {
	return Int64Slice(flagSet.FlagSet, flagName, flagDescription)
}

type FlagInt64Slice []int64

func (s *FlagInt64Slice) String() string {
	return fmt.Sprintf("%v", *s)
}

func (s *FlagInt64Slice) Set(value string) error {
	var id int64
	_, err := fmt.Sscanf(value, "%d", &id)
	if err != nil {
		return err
	}
	*s = append(*s, id)
	return nil
}
