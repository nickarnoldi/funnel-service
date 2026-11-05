package enum

type OfferURLStatusEnum int

func (s OfferURLStatusEnum) String() string {
	switch OfferURLStatus[s] {
	case "active":
		return "Active"
	case "paused":
		return "Paused"
	case "deleted":
		return "Deleted"
	default:
		return "Invalid"
	}
}

const (
	OfferURLStatusActive OfferURLStatusEnum = iota
	OfferURLStatusPaused
	OfferURLStatusDeleted
)

var OfferURLStatus = map[OfferURLStatusEnum]string{
	OfferURLStatusActive:  "active",
	OfferURLStatusPaused:  "paused",
	OfferURLStatusDeleted: "deleted",
}
