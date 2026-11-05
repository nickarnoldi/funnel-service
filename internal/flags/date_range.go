package flags

import (
	"flag"
	"time"
)

/**
 * flagDateRange sets up start and end date flags and returns a DateRangeResult
 * containing pointers to the dateRangeFlag variables for start and end dates.
 */
func DateRange(flagSet *flag.FlagSet, flagNames ...string) (*time.Time, *time.Time) {
	var startName string = "start"
	var endName string = "end"
	if len(flagNames) > 0 {
		startName = flagNames[0]
		if len(flagNames) > 1 {
			endName = flagNames[1]
		}
	}

	var startTime time.Time
	var endTime time.Time

	flagStart := &FlagTime{
		Time: &startTime,
	}
	flagEnd := &FlagTimeEndOfDay{
		FlagTime: FlagTime{Time: &endTime},
	}

	flagSet.Var(flagStart, startName, "Start date for syncing")
	flagSet.Var(flagEnd, endName, "End date for syncing")
	return &startTime, &endTime
}

func (flagSet *FlagSet) DateRange(flagNames ...string) (*time.Time, *time.Time) {
	return DateRange(flagSet.FlagSet, flagNames...)
}
