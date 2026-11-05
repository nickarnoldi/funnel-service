package enum

type EmailAttributionMethodEnum int

func (a EmailAttributionMethodEnum) String() string {
	switch EmailAttributionMethod[a] {
	case "last_affiliate_attribution":
		return "Last Partner Attribution"
	case "first_affiliate_attribution":
		return "First Partner Attribution"
	default:
		return "Invalid"
	}
}

const (
	EmailAttributionMethodLastPartnerAttribution EmailAttributionMethodEnum = iota
	EmailAttributionMethodFirstPartnerAttribution
)

var EmailAttributionMethod = map[EmailAttributionMethodEnum]string{
	EmailAttributionMethodLastPartnerAttribution:  "last_affiliate_attribution",
	EmailAttributionMethodFirstPartnerAttribution: "first_affiliate_attribution",
}
