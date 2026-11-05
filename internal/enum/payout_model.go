package enum

type PayoutModelEnum int

func (p PayoutModelEnum) String() string {
	switch PayoutModel[p] {
	case "CPC":
		return "Cost Per Click"
	case "CPA":
		return "Cost Per Action"
	case "CPS":
		return "Cost Per Sale"
	case "PRV":
		return "Percentage of Revenue"
	default:
		return "Invalid"
	}
}

const (
	PayoutModelCostPerClick PayoutModelEnum = iota
	PayoutModelCostPerAction
	PayoutModelCostPerSale
	PayoutModelPercentageOfRevenue
)

var PayoutModel = map[PayoutModelEnum]string{
	PayoutModelCostPerClick:        "CPC",
	PayoutModelCostPerAction:       "CPA",
	PayoutModelCostPerSale:         "CPS",
	PayoutModelPercentageOfRevenue: "PRV",
}
