package flags

import (
	"flag"
	"fmt"
	"time"
)

func Time(flagSet *flag.FlagSet, name string, usage string) *time.Time {
	var timeValue time.Time
	var date FlagTime = FlagTime{Time: &timeValue}
	flagSet.Var(&date, name, usage)
	return &timeValue
}

func (flagSet *FlagSet) Time(name string, usage string) *time.Time {
	return Time(flagSet.FlagSet, name, usage)
}

type FlagTime struct {
	*time.Time
}

func (date *FlagTime) Set(value string) error {
	if value != "" {
		switch value {
		case "today":
			*date.Time = time.Now().UTC().Truncate(24 * time.Hour)
			return nil
		default:
			parsedDate, err := time.Parse("2006-01-02 15:04:05", value)
			if err != nil {
				parsedDate, err = time.Parse("2006-01-02", value)
				if err != nil {
					return fmt.Errorf("invalid date: %v", err)
				}
			}
			*date.Time = parsedDate
			return nil
		}
	}
	return nil
}

func (date *FlagTime) String() string {
	if date.Time == nil || date.Time.IsZero() {
		return ""
	}
	return date.Time.Format(time.RFC3339)
}

type FlagTimeEndOfDay struct {
	FlagTime
}

func (date *FlagTimeEndOfDay) Set(value string) error {
	if err := date.FlagTime.Set(value); err != nil {
		return err
	}

	*date.Time = date.Time.Truncate(24 * time.Hour).Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	return nil
}
