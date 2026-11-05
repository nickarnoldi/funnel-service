package enum

type ConversionStatusEnum int

func (c ConversionStatusEnum) String() string {
	switch ConversionStatus[c] {
	case "approved":
		return "Approved"
	case "rejected":
		return "Rejected"
	case "pending":
		return "Pending"
	case "invalid":
		return "Invalid"
	default:
		return "Unknown"
	}
}

const (
	ConversionStatusApproved ConversionStatusEnum = iota
	ConversionStatusRejected
	ConversionStatusPending
	ConversionStatusInvalid
)

var ConversionStatus = map[ConversionStatusEnum]string{
	ConversionStatusApproved: "approved",
	ConversionStatusRejected: "rejected",
	ConversionStatusPending:  "pending",
	ConversionStatusInvalid:  "invalid",
}
