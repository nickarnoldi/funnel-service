package enum

type AttributionMethodEnum int

func (a AttributionMethodEnum) String() string {
	switch AttributionMethod[a] {
	case "last_touch":
		return "Last Touch"
	case "first_touch":
		return "First Touch"
	default:
		return "Invalid"
	}
}

const (
	AttributionMethodLastTouch AttributionMethodEnum = iota
	AttributionMethodFirstTouch
)

var AttributionMethod = map[AttributionMethodEnum]string{
	AttributionMethodLastTouch:  "last_touch",
	AttributionMethodFirstTouch: "first_touch",
}
