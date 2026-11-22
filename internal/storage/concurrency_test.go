package storage

import (
	"context"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
)

func TestConcurrentReservations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "db")
	a, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer a.DB.Close()
	b, e := Open(path)
	if e != nil {
		t.Fatal(e)
	}
	defer b.DB.Close()
	var wg sync.WaitGroup
	var accepted atomic.Int32
	for i, s := range []*Store{a, b} {
		wg.Add(1)
		go func(id int, s *Store) {
			defer wg.Done()
			r := Record{ID: string(rune('a' + id)), Account: "u", Recipient: "+254712345678", Currency: "KES", Amount: 10000, Environment: "sandbox", Created: Now()}
			if s.Reserve(context.Background(), r, 10000) == nil {
				accepted.Add(1)
			}
		}(i, s)
	}
	wg.Wait()
	if accepted.Load() != 1 {
		t.Fatalf("accepted %d concurrent purchases", accepted.Load())
	}
}
