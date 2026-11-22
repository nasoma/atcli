package phone

import (
	"fmt"
	"github.com/nasoma/atcli/internal/money"
	"github.com/nyaruka/phonenumbers"
	"regexp"
	"strings"
)

var chars = regexp.MustCompile(`^\+?[0-9]+$`)

func Normalize(s, currency string) (string, error) {
	if !chars.MatchString(s) {
		return "", fmt.Errorf("invalid phone number: use digits and an optional leading +")
	}
	if strings.HasPrefix(s, "254") {
		s = "+" + s
	}
	p, e := phonenumbers.Parse(s, "KE")
	if e != nil || !phonenumbers.IsValidNumber(p) {
		return "", fmt.Errorf("invalid phone number; expected +254712345678")
	}
	region := phonenumbers.GetRegionCodeForNumber(p)
	market, ok := money.Markets[region]
	if !ok || market.Currency != currency {
		return "", fmt.Errorf("unsupported destination or currency mismatch (supported: KE/KES, UG/UGX, TZ/TZS, NG/NGN)")
	}
	t := phonenumbers.GetNumberType(p)
	if t != phonenumbers.MOBILE && t != phonenumbers.FIXED_LINE_OR_MOBILE {
		return "", fmt.Errorf("recipient must be a mobile number")
	}
	return phonenumbers.Format(p, phonenumbers.E164), nil
}
