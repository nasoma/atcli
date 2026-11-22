package service

import (
	"context"
	"fmt"
	"github.com/nasoma/atcli/internal/africastalking"
	"github.com/nasoma/atcli/internal/config"
	"github.com/nasoma/atcli/internal/storage"
	"path/filepath"
	"testing"
)

type fakeAPI struct {
	calls     int
	uncertain bool
}

func (f *fakeAPI) Balance(context.Context) (africastalking.Balance, error) {
	return africastalking.Balance{}, nil
}
func (f *fakeAPI) Send(_ context.Context, p, c, a, id string) (africastalking.Entry, error) {
	f.calls++
	if f.uncertain {
		return africastalking.Entry{}, fmt.Errorf("timeout")
	}
	return africastalking.Entry{Phone: p, RequestID: "ATQid_1", Status: "Sent"}, nil
}
func (f *fakeAPI) Status(context.Context, string) (string, error) { return "Success", nil }
func TestSendSafety(t *testing.T) {
	for _, uncertain := range []bool{false, true} {
		t.Run(fmt.Sprint(uncertain), func(t *testing.T) {
			st, e := storage.Open(filepath.Join(t.TempDir(), "db"))
			if e != nil {
				t.Fatal(e)
			}
			defer st.DB.Close()
			api := &fakeAPI{uncertain: uncertain}
			s := Service{config.Default(), "sandbox", "sandbox", api, st}
			r, e := s.Preview("0712345678", "100", "KES")
			if e != nil {
				t.Fatal(e)
			}
			r, e = s.Send(context.Background(), r)
			if (e != nil) != uncertain {
				t.Fatal(e)
			}
			if r.Status == "delivered" {
				t.Fatal("initial response treated as delivery")
			}
			_, e = s.Send(context.Background(), r)
			if e == nil || api.calls != 1 {
				t.Fatal("duplicate sent", e)
			}
			if !uncertain {
				r, e = s.Status(context.Background(), r.ID)
				if e != nil || r.Status != "delivered" {
					t.Fatal(r, e)
				}
			}
		})
	}
}
