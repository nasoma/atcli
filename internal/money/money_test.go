package money

import "testing"

func TestParse(t *testing.T) {
	for _, s := range []string{"0", "-1", "1.001", "NaN", "1e2", " 10", "999999999999999999999999"} {
		if _, e := Parse(s); e == nil {
			t.Errorf("accepted %q", s)
		}
	}
	for s, want := range map[string]int64{"10": 1000, "250.50": 25050, "1.1": 110} {
		v, e := Parse(s)
		if e != nil || v != want {
			t.Errorf("%s: %d %v", s, v, e)
		}
	}
}
