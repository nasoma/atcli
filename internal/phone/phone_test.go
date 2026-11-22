package phone

import "testing"

func TestNormalize(t *testing.T) {
	for _, s := range []string{"0712345678", "712345678", "254712345678", "+254712345678", "0112345678"} {
		v, e := Normalize(s, "KES")
		if e != nil || len(v) != 13 {
			t.Errorf("%s: %s %v", s, v, e)
		}
	}
	for _, s := range []string{"+254123", "0712x45678", "+12025550123", "+254202345678"} {
		if _, e := Normalize(s, "KES"); e == nil {
			t.Errorf("accepted %s", s)
		}
	}
	if _, e := Normalize("+254712345678", "NGN"); e == nil {
		t.Fatal("currency mismatch")
	}
}
