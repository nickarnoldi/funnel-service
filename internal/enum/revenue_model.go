package enum

type RevenueModelEnum int

func (r RevenueModelEnum) String() string {
	switch RevenueModel[r] {
	case "RPC":
		return "Revenue Per Click"
	case "RPA":
		return "Revenue Per Action"
	case "RPS":
		return "Revenue Per Sale"
	default:
		return "Invalid"
	}
}

const (
	RevenueModelRevenuePerClick RevenueModelEnum = iota
	RevenueModelRevenuePerAction
	RevenueModelRevenuePerSale
)

var RevenueModel = map[RevenueModelEnum]string{
	RevenueModelRevenuePerClick:  "RPC",
	RevenueModelRevenuePerAction: "RPA",
	RevenueModelRevenuePerSale:   "RPS",
}
