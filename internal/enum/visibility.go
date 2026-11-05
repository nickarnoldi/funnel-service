package enum

type VisibilityEnum int

func (v VisibilityEnum) String() string {
	switch Visibility[v] {
	case "private":
		return "Private"
	case "require_approval":
		return "Require Approval"
	case "public":
		return "Public"
	default:
		return "Invalid"
	}
}

const (
	VisibilityPrivate VisibilityEnum = iota
	VisibilityRequireApproval
	VisibilityPublic
)

var Visibility = map[VisibilityEnum]string{
	VisibilityPrivate:         "private",
	VisibilityRequireApproval: "require_approval",
	VisibilityPublic:          "public",
}
