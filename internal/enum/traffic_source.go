package enum

import "strings"

type TrafficSourceEnum int

func (s TrafficSourceEnum) String() string {
	switch TrafficSource[s] {
	case "email":
		return "Email"
	case "sms":
		return "SMS"
	case "meta":
		return "Meta"
	case "google":
		return "Google"
	case "twitter":
		return "Twitter"
	case "linkedin":
		return "LinkedIn"
	case "tiktok":
		return "TikTok"
	case "pinterest":
		return "Pinterest"
	case "snapchat":
		return "Snapchat"
	case "taboola":
		return "Taboola"
	case "outbrain":
		return "Outbrain"
	default:
		return "Other"
	}
}

func (s *TrafficSourceEnum) FromString(str string) {
	str = strings.ToLower(str)
	for index, name := range TrafficSource {
		if name == str || strings.Contains(str, name) {
			*s = index
			return
		}
	}

	// exception for Facebook as it can appear as "facebook" or "fb"
	if strings.Contains(str, "facebook") || strings.Contains(str, "fb") || strings.Contains(str, "instagram") {
		*s = TrafficSourceMeta
		return
	}

	*s = TrafficSourceOther
}

const (
	TrafficSourceOther TrafficSourceEnum = iota
	TrafficSourceEmail
	TrafficSourceSMS
	TrafficSourceMeta
	TrafficSourceGoogle
	TrafficSourceTwitter
	TrafficSourceLinkedIn
	TrafficSourceTikTok
	TrafficSourcePinterest
	TrafficSourceSnapchat
	TrafficSourceTaboola
	TrafficSourceOutbrain
)

var TrafficSource = map[TrafficSourceEnum]string{
	TrafficSourceOther:     "other",
	TrafficSourceEmail:     "email",
	TrafficSourceSMS:       "sms",
	TrafficSourceMeta:      "meta",
	TrafficSourceGoogle:    "google",
	TrafficSourceTwitter:   "twitter",
	TrafficSourceLinkedIn:  "linkedin",
	TrafficSourceTikTok:    "tiktok",
	TrafficSourcePinterest: "pinterest",
	TrafficSourceSnapchat:  "snapchat",
	TrafficSourceTaboola:   "taboola",
	TrafficSourceOutbrain:  "outbrain",
}
