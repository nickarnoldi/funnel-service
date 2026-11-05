package enum

type StatusEnum int

func (s StatusEnum) String() string {
	switch Status[s] {
	case "active":
		return "Active"
	case "inactive":
		return "Inactive"
	case "pending":
		return "Pending"
	case "suspended":
		return "Suspended"
	case "deleted":
		return "Deleted"
	default:
		return "Invalid"
	}
}

const (
	StatusActive StatusEnum = iota
	StatusInactive
	StatusPending
	StatusSuspended
	StatusDeleted
)

var Status = map[StatusEnum]string{
	StatusActive:    "active",
	StatusInactive:  "inactive",
	StatusPending:   "pending",
	StatusSuspended: "suspended",
	StatusDeleted:   "deleted",
}
