package africastalking

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func mock(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	s := httptest.NewServer(handler)
	t.Cleanup(s.Close)
	return &Client{s.URL, "sandbox", "secret", s.Client()}
}
func TestBalance(t *testing.T) {
	for _, tc := range []struct {
		body string
		code int
		bad  bool
	}{{`{"UserData":{"balance":"KES 250.50"}}`, 200, false}, {`{"UserData":{"balance":"KES 0.00"}}`, 200, false}, {`{`, 200, true}, {`{}`, 200, true}, {`{"UserData":{"balance":"KES no"}}`, 200, true}, {`secret`, 401, true}, {`bad`, 500, true}} {
		t.Run(tc.body, func(t *testing.T) {
			c := mock(t, func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/version1/user" || r.URL.Query().Get("username") != "sandbox" || r.Header.Get("apiKey") != "secret" {
					t.Error("wrong request")
				}
				w.WriteHeader(tc.code)
				fmt.Fprint(w, tc.body)
			})
			_, e := c.Balance(context.Background())
			if (e != nil) != tc.bad {
				t.Fatal(e)
			}
			if e != nil && strings.Contains(e.Error(), "secret") {
				t.Fatal("key leak")
			}
		})
	}
}
func TestSend(t *testing.T) {
	for _, status := range []string{"Sent", "Failed", "Queued"} {
		c := mock(t, func(w http.ResponseWriter, r *http.Request) {
			r.ParseForm()
			if r.Method != "POST" || r.URL.Path != "/version1/airtime/send" || r.Header.Get("Idempotency-Key") != "operation" || r.Form.Get("maxNumRetry") != "0" {
				t.Error("wrong send request")
			}
			if !strings.Contains(r.Form.Get("recipients"), "KES 100.00") {
				t.Error("recipient amount")
			}
			fmt.Fprintf(w, `{"responses":[{"phoneNumber":"+254712345678","amount":"KES 100.00","status":%q,"requestId":"ATQid_1"}]}`, status)
		})
		v, e := c.Send(context.Background(), "+254712345678", "KES", "100.00", "operation")
		if e != nil || v.Status != status {
			t.Fatal(v, e)
		}
	}
}
func TestStatus(t *testing.T) {
	for _, s := range []string{"Success", "Failed", "Sent", "Unrecognized"} {
		c := mock(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/query/transaction/find" || r.URL.Query().Get("transactionId") != "ATQid_1" {
				t.Error("wrong status request")
			}
			fmt.Fprintf(w, `{"status":%q}`, s)
		})
		got, e := c.Status(context.Background(), "ATQid_1")
		if e != nil || got != s {
			t.Fatal(got, e)
		}
	}
	if Delivery("Sent") != "pending" || Delivery("Unrecognized") != "unknown" {
		t.Fatal("unsafe delivery mapping")
	}
}
func TestTimeoutAndRedirect(t *testing.T) {
	c := mock(t, func(w http.ResponseWriter, r *http.Request) { time.Sleep(30 * time.Millisecond); fmt.Fprint(w, `{}`) })
	c.http.Timeout = time.Millisecond
	if _, e := c.Balance(context.Background()); e == nil {
		t.Fatal("expected timeout")
	}
	c.http.CloseIdleConnections()
	prod, _ := New("production", "u", "k")
	if prod.http.CheckRedirect(nil, nil) != http.ErrUseLastResponse {
		t.Fatal("redirect allowed")
	}
}

func TestFractionalBalance(t *testing.T) {
	for _, amount := range []string{"190.0000", "0.0012"} {
		c := mock(t, func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, `{"UserData":{"balance":"KES %s"}}`, amount)
		})
		v, e := c.Balance(context.Background())
		if e != nil {
			t.Fatal(e)
		}
		if amount == "0.0012" && (v.Amount != amount || v.Minor != nil) {
			t.Fatal("fractional balance rounded", v)
		}
	}
}

func TestInvalidAirtimeResponse(t *testing.T) {
	for _, body := range []string{`{}`, `{"responses":[{"phoneNumber":"+254712345678","status":"Sent"}]}`, `{"responses":[{"phoneNumber":"+254722345678","status":"Sent","requestId":"ATQid_1"}]}`} {
		c := mock(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) })
		if _, e := c.Send(context.Background(), "+254712345678", "KES", "100.00", "op"); e == nil {
			t.Fatal("unsafe response accepted", body)
		}
	}
}
func TestBoundedAndRedacted(t *testing.T) {
	c := mock(t, func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, strings.Repeat("x", 1048577)) })
	if _, e := c.Balance(context.Background()); e == nil {
		t.Fatal("oversized response accepted")
	}
	if strings.Contains(c.clean("secret\x1b[31m"), "secret") || strings.Contains(c.clean("x\x1b"), "\x1b") {
		t.Fatal("secret or terminal control leaked")
	}
}
