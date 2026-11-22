package storage

import (
	"context"
	"path/filepath"
	"testing"
)

func TestDurableIntent(t *testing.T) {
	ctx := context.Background()
	p := filepath.Join(t.TempDir(), "history.sqlite")
	s, e := Open(p)
	if e != nil {
		t.Fatal(e)
	}
	r := Record{ID: "one", Account: "u", Recipient: "+254712345678", Amount: 10000, Currency: "KES", Environment: "sandbox", Created: Now()}
	if e = s.Reserve(ctx, r, 15000); e != nil {
		t.Fatal(e)
	}
	r.ID = "two"
	if e = s.Reserve(ctx, r, 15000); e == nil {
		t.Fatal("duplicate accepted")
	}
	r.Recipient = "+254722345678"
	if e = s.Reserve(ctx, r, 15000); e == nil {
		t.Fatal("daily limit bypassed")
	}
	s.DB.Close()
	s, e = Open(p)
	if e != nil {
		t.Fatal(e)
	}
	defer s.DB.Close()
	rs, e := s.List(ctx, "sandbox", "unknown", 20)
	if e != nil || len(rs) != 1 || rs[0].Submission != "intent" {
		t.Fatal(rs, e)
	}
	r = rs[0]
	r.RequestID = "ATQid_1"
	r.Status = "pending"
	r.Submission = "accepted"
	if e = s.Update(ctx, r); e != nil {
		t.Fatal(e)
	}
	r, e = s.Find(ctx, "sandbox", "u", "ATQid_1")
	if e != nil || r.Status != "pending" {
		t.Fatal(r, e)
	}
	r.Status = "failed"
	if e = s.Update(ctx, r); e != nil {
		t.Fatal(e)
	}
	r.ID = "three"
	if e = s.Reserve(ctx, r, 15000); e != nil {
		t.Fatal(e)
	}
}
