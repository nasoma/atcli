package service

import (
	"context"
	"errors"
	"github.com/nasoma/atcli/internal/africastalking"
	"github.com/nasoma/atcli/internal/config"
	"github.com/nasoma/atcli/internal/storage"
	"path/filepath"
	"testing"
)

type rejectedAPI struct {
	fakeAPI
	status string
}

func (f *rejectedAPI) Send(_ context.Context, p, c, a, id string) (africastalking.Entry, error) {
	return africastalking.Entry{Phone: p, Status: f.status}, nil
}
func TestRejectedAndUnexpected(t *testing.T) {
	for _, status := range []string{"Failed", "Unexpected"} {
		st, e := storage.Open(filepath.Join(t.TempDir(), "db"))
		if e != nil {
			t.Fatal(e)
		}
		s := Service{config.Default(), "sandbox", "sandbox", &rejectedAPI{status: status}, st}
		r, e := s.Preview("0712345678", "100", "KES")
		if e != nil {
			t.Fatal(e)
		}
		r, e = s.Send(context.Background(), r)
		want := ErrRejected
		if status == "Unexpected" {
			want = ErrUnresolved
		}
		if !errors.Is(e, want) {
			t.Fatal(r, e)
		}
		st.DB.Close()
	}
}
