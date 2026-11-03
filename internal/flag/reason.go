package flag

import "errors"

const (
	ReasonPolicy  = "policy"
	ReasonConduct = "conduct"
	ReasonDeal    = "deal"
)

var ErrInvalidReason = errors.New("invalid flag reason")

type ReasonInfo struct {
	Code     string
	Label    string
	Examples []string
}

// ReasonsAtAd is for inquirer throwing at a listing.
var ReasonsAtAd = []ReasonInfo{
	{
		Code:  ReasonPolicy,
		Label: "Listing or content violates policies",
		Examples: []string{
			"Scam or fake listing",
			"Illegal goods",
			"Spam or impersonation",
			"Sexual or prohibited images",
			"Stolen photos or IP issues",
		},
	},
	{
		Code:  ReasonConduct,
		Label: "Seller harassment or bad-faith conduct",
		Examples: []string{
			"Threats or abuse in messages",
			"Pressure to share personal info",
			"Repeated hostile contact",
			"Bad-faith negotiation",
		},
	},
	{
		Code:  ReasonDeal,
		Label: "Deal or transaction went wrong",
		Examples: []string{
			"No-show",
			"Item not as described",
			"Payment or meetup dispute",
			"Bait-and-switch after chat",
		},
	},
}

// ReasonsAtUser is for owner throwing at an inquirer.
var ReasonsAtUser = []ReasonInfo{
	{
		Code:  ReasonPolicy,
		Label: "Scam, spam, or prohibited requests",
		Examples: []string{
			"Advance-fee or deposit scam",
			"Spam or mass messaging",
			"Requests for illegal goods or services",
			"Phishing or credential harvesting",
		},
	},
	{
		Code:  ReasonConduct,
		Label: "Harassment or bad-faith conduct",
		Examples: []string{
			"Threats or abuse in messages",
			"Pressure to share personal info",
			"Repeated hostile contact",
			"Bad-faith negotiation",
		},
	},
	{
		Code:  ReasonDeal,
		Label: "Deal or meetup went wrong",
		Examples: []string{
			"No-show",
			"Changed terms after agreeing",
			"Payment dispute",
			"Unsafe or dishonest meetup behavior",
		},
	},
}

// Reasons is the union used for code validation.
var Reasons = ReasonsAtAd

func ReasonsForTarget(atAd bool) []ReasonInfo {
	if atAd {
		return ReasonsAtAd
	}
	return ReasonsAtUser
}

func ValidReason(code string) bool {
	switch code {
	case ReasonPolicy, ReasonConduct, ReasonDeal:
		return true
	default:
		return false
	}
}

func ReasonLabel(code string) string {
	for _, r := range ReasonsAtAd {
		if r.Code == code {
			return r.Label
		}
	}
	return ""
}

func ReasonLabelForTarget(code string, atAd bool) string {
	for _, r := range ReasonsForTarget(atAd) {
		if r.Code == code {
			return r.Label
		}
	}
	return ReasonLabel(code)
}
