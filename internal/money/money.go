package money

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var decimal = regexp.MustCompile(`^[0-9]+(?:\.[0-9]{1,2})?$`)

func Parse(s string) (int64, error) {
	if !decimal.MatchString(s) {
		return 0, fmt.Errorf("amount must be a positive decimal with at most two fractional digits")
	}
	p := strings.Split(s, ".")
	whole, e := strconv.ParseInt(p[0], 10, 64)
	if e != nil || whole > 900000000000000 {
		return 0, fmt.Errorf("amount too large")
	}
	n := whole * 100
	if len(p) == 2 {
		f := p[1]
		if len(f) == 1 {
			f += "0"
		}
		v, _ := strconv.ParseInt(f, 10, 64)
		n += v
	}
	if n <= 0 {
		return 0, fmt.Errorf("amount must be greater than zero")
	}
	return n, nil
}
func Format(n int64) string { return fmt.Sprintf("%d.%02d", n/100, n%100) }

type Market struct {
	Region, Currency string
	Min, Max         int64
}

var Markets = map[string]Market{"KE": {"KE", "KES", 1000, 1000000}, "UG": {"UG", "UGX", 5000, 20000000}, "TZ": {"TZ", "TZS", 100000, 20000000}, "NG": {"NG", "NGN", 5000, 500000}}

func Currency(s string) bool {
	for _, m := range Markets {
		if m.Currency == s {
			return true
		}
	}
	return false
}
