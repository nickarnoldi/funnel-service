package enum

type AttributionPriorityEnum int

func (a AttributionPriorityEnum) String() string {
	switch AttributionPriority[a] {
	case "click":
		return "Click"
	case "coupon_code":
		return "Coupon Code"
	default:
		return "Invalid"
	}
}

const (
	AttributionPriorityClick AttributionPriorityEnum = iota
	AttributionPriorityCouponCode
)

var AttributionPriority = map[AttributionPriorityEnum]string{
	AttributionPriorityClick:      "click",
	AttributionPriorityCouponCode: "coupon_code",
}
