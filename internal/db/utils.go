package db

import (
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nickarnoldi/funnel-service/internal/constants"
)

type Integer interface {
	~int32 | ~int
}

func BigInt(v *int64) pgtype.Int8 {
	if v == nil {
		return pgtype.Int8{}
	}

	return pgtype.Int8{
		Int64: *v,
		Valid: true,
	}
}

func Int[T Integer](v *T) pgtype.Int4 {
	if v == nil {
		return pgtype.Int4{}
	}

	switch v := any(v).(type) {
	case *int32:
		return pgtype.Int4{
			Int32: *v,
			Valid: true,
		}
	case *int:
		return pgtype.Int4{
			Int32: int32(*v),
			Valid: true,
		}
	default:
		fmt.Printf("Unsupported type for Int conversion: %T\n", v)
		return pgtype.Int4{}
	}
}

func SmallInt(v *int16) pgtype.Int2 {
	if v == nil {
		return pgtype.Int2{}
	}

	return pgtype.Int2{
		Int16: *v,
		Valid: true,
	}
}

func String(v *string) pgtype.Text {
	if v == nil || *v == "" {
		return pgtype.Text{}
	}

	return pgtype.Text{
		String: *v,
		Valid:  true,
	}
}

func Boolean(v *bool) pgtype.Bool {
	if v == nil {
		return pgtype.Bool{}
	}

	return pgtype.Bool{
		Bool:  *v,
		Valid: true,
	}
}

func Numeric[T ~string](v *T) pgtype.Numeric {
	if v == nil {
		return pgtype.Numeric{}
	}

	var numeric pgtype.Numeric
	if err := numeric.Scan(string(*v)); err != nil && string(*v) != "" {
		fmt.Printf("Error scanning string to pgtype.Numeric: '%s' - %v\n", *v, err)
	}

	return numeric
}

func (s *Store) GenerateSafeDateRange(startDate time.Time, endDate time.Time) (time.Time, time.Time, error) {
	var safeStartDate time.Time
	var safeEndDate time.Time

	if startDate.IsZero() {
		safeStartDate = constants.FunnelLoggingStartDate
	} else {
		safeStartDate = startDate
	}

	if endDate.IsZero() {
		safeEndDate = safeStartDate.Add(18 * time.Hour)
		safeEndDate = time.Date(safeEndDate.Year(), safeEndDate.Month(), safeEndDate.Day()+1, 0, 0, 0, 0, time.UTC).Add(-time.Microsecond)
	} else {
		safeEndDate = endDate
		if safeEndDate.Before(safeStartDate) {
			return time.Time{}, time.Time{}, fmt.Errorf("end date %s cannot be before start date %s", safeEndDate.Format("2006-01-02 15:04:05"), safeStartDate.Format("2006-01-02 15:04:05"))
		}
	}

	if safeEndDate.After(time.Now()) {
		safeEndDate = time.Now().UTC()
	}

	return safeStartDate, safeEndDate, nil
}
