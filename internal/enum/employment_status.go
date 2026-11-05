package enum

import "strings"

type EmploymentStatusEnum int16

func (c EmploymentStatusEnum) String() string {
	return EmploymentStatus[c]
}

func (c *EmploymentStatusEnum) FromString(status string) {
	status = strings.ToLower(status)
	for k, v := range EmploymentStatus {
		if strings.ToLower(v) == status {
			*c = k
			return
		}
	}

	*c = EmploymentStatusUnknown
}

const (
	EmploymentStatusUnknown EmploymentStatusEnum = iota
	EmploymentStatusUnemployed
	EmploymentStatusRetired
	EmploymentStatusDisability
	EmploymentStatusEmployed
	EmploymentStatusSelfEmployed
	EmploymentStatusOther
)

var EmploymentStatus = map[EmploymentStatusEnum]string{
	EmploymentStatusUnknown:      "Unknown",
	EmploymentStatusUnemployed:   "Unemployed",
	EmploymentStatusRetired:      "Retired",
	EmploymentStatusDisability:   "Disability",
	EmploymentStatusEmployed:     "Employed",
	EmploymentStatusSelfEmployed: "Self Employed",
	EmploymentStatusOther:        "Other",
}
