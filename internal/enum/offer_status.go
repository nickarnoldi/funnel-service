package enum

type OfferStatusEnum int

func (s OfferStatusEnum) String() string {
	switch OfferStatus[s] {
	case "active":
		return "Active"
	case "paused":
		return "Paused"
	case "pending":
		return "Pending"
	case "deleted":
		return "Deleted"
	default:
		return "Invalid"
	}
}

const (
	OfferStatusActive OfferStatusEnum = iota
	OfferStatusPaused
	OfferStatusPending
	OfferStatusDeleted
)

var OfferStatus = map[OfferStatusEnum]string{
	OfferStatusActive:  "active",
	OfferStatusPaused:  "paused",
	OfferStatusPending: "pending",
	OfferStatusDeleted: "deleted",
}
