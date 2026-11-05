package enum

type PlatformEnum int

func (p PlatformEnum) String() string {
	return Platform[p]
}

const (
	PlatformLeadspedia PlatformEnum = iota
	PlatformEverflowPrimary
	PlatformEverflowSecondary
	PlatformUnknown
)

var Platform = map[PlatformEnum]string{
	PlatformLeadspedia:        "Leadspedia",
	PlatformEverflowPrimary:   "Everflow Primary",
	PlatformEverflowSecondary: "Everflow Secondary",
	PlatformUnknown:           "Unknown",
}
