package enum

type RedirectMethodEnum int

func (r RedirectMethodEnum) String() string {
	switch RedirectMethod[r] {
	case "standard":
		return "Standard"
	default:
		return "Invalid"
	}
}

const (
	RedirectMethodStandard RedirectMethodEnum = iota
)

var RedirectMethod = map[RedirectMethodEnum]string{
	RedirectMethodStandard: "standard",
}
