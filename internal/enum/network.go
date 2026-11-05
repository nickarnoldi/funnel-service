package enum

type NetworkEnum int

func (r NetworkEnum) String() string {
	switch r {
	case NetworkPrimary:
		return "Everflow Primary"
	case NetworkSecondary:
		return "Everflow Secondary"
	default:
		return "Invalid"
	}
}

func (r NetworkEnum) GetPlatform() PlatformEnum {
	switch r {
	case NetworkPrimary:
		return PlatformEverflowPrimary
	case NetworkSecondary:
		return PlatformEverflowSecondary
	case 0:
		return PlatformLeadspedia
	default:
		return PlatformUnknown
	}
}

func FilterNetwork(requested []NetworkEnum) map[NetworkEnum]string {
	available := Network
	if len(requested) > 0 {
		available = make(map[NetworkEnum]string)
		for _, ID := range requested {
			if name, exists := Network[ID]; exists {
				available[ID] = name
			}
		}
	}

	return available
}

const (
	NetworkPrimary   NetworkEnum = 1
	NetworkSecondary NetworkEnum = 2
)

var Network = map[NetworkEnum]string{
	NetworkPrimary:   "primary",
	NetworkSecondary: "secondary",
}
