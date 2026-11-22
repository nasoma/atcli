package service

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/nasoma/atcli/internal/africastalking"
	"github.com/nasoma/atcli/internal/config"
	"github.com/nasoma/atcli/internal/money"
	"github.com/nasoma/atcli/internal/phone"
	"github.com/nasoma/atcli/internal/storage"
	"strings"
)

var ErrRejected = errors.New("provider rejected airtime request")
var ErrUnresolved = errors.New("purchase outcome unresolved; reconcile before repeating")

type API interface {
	Balance(context.Context) (africastalking.Balance, error)
	Send(context.Context, string, string, string, string) (africastalking.Entry, error)
	Status(context.Context, string) (string, error)
}
type Service struct {
	Config    config.Config
	Env, User string
	API       API
	Store     *storage.Store
}

func (s *Service) Preview(recipient, amount, currency string) (storage.Record, error) {
	if s.Env != "sandbox" && s.Env != "production" {
		return storage.Record{}, fmt.Errorf("invalid environment")
	}
	p, e := phone.Normalize(recipient, currency)
	if e != nil {
		return storage.Record{}, e
	}
	a, e := money.Parse(amount)
	if e != nil {
		return storage.Record{}, e
	}
	max, e := money.Parse(s.Config.Maximum)
	if e != nil {
		return storage.Record{}, e
	}
	var market money.Market
	for _, m := range money.Markets {
		if m.Currency == currency {
			market = m
		}
	}
	if a < market.Min || a > market.Max {
		return storage.Record{}, fmt.Errorf("provider limit: %s %s–%s", currency, money.Format(market.Min), money.Format(market.Max))
	}
	if a > max {
		return storage.Record{}, fmt.Errorf("configured maximum %s exceeded", s.Config.Maximum)
	}
	return storage.Record{Account: s.User, Recipient: p, Amount: a, Currency: currency, Environment: s.Env}, nil
}
func (s *Service) Send(ctx context.Context, r storage.Record) (storage.Record, error) {
	checked, e := s.Preview(r.Recipient, money.Format(r.Amount), r.Currency)
	if e != nil {
		return r, e
	}
	r = checked
	b := make([]byte, 16)
	if _, e = rand.Read(b); e != nil {
		return r, e
	}
	r.ID = hex.EncodeToString(b)
	r.Created = storage.Now()
	r.Status = "unknown"
	r.Submission = "intent"
	daily := int64(0)
	if s.Config.Daily != "0" {
		daily, e = money.Parse(s.Config.Daily)
		if e != nil {
			return r, e
		}
	}
	if e = s.Store.Reserve(ctx, r, daily); e != nil {
		return r, e
	}
	v, sendErr := s.API.Send(ctx, r.Recipient, r.Currency, money.Format(r.Amount), r.ID)
	if sendErr != nil {
		r.Message = "Submission outcome unresolved; reconcile in provider dashboard before repeating"
		r.Submission = "unresolved"
	} else {
		r.RequestID = v.RequestID
		r.Message = v.ErrorMessage
		r.Submission = "unresolved"
		r.Status = "unknown"
		switch v.Status {
		case "Sent", "Queued":
			r.Submission = "accepted"
			r.Status = "pending"
		case "Failed", "Rejected":
			r.Submission = "rejected"
			r.Status = "failed"
		}
		if r.Message == "" {
			r.Message = v.Status
		}
	}
	if e = s.Store.Update(context.Background(), r); e != nil {
		return r, fmt.Errorf("%w: local update failed, operation %s: %v", ErrUnresolved, r.ID, e)
	}
	if sendErr != nil {
		return r, fmt.Errorf("%w: operation %s: %v", ErrUnresolved, r.ID, sendErr)
	}
	if r.Status == "failed" {
		return r, ErrRejected
	}
	if r.Status == "unknown" {
		return r, ErrUnresolved
	}
	return r, nil
}
func (s *Service) Status(ctx context.Context, id string) (storage.Record, error) {
	r, e := s.Store.Find(ctx, s.Env, s.User, id)
	local := true
	if errors.Is(e, sql.ErrNoRows) && strings.HasPrefix(id, "ATQid_") && len(id) <= 200 {
		local = false
		r = storage.Record{RequestID: id, Account: s.User, Environment: s.Env}
	} else if e != nil {
		return r, fmt.Errorf("transaction not found; supply a local operation ID or provider ATQid identifier")
	}

	if r.RequestID == "" {
		return r, fmt.Errorf("no provider ID recorded; reconcile operation %s in the provider dashboard; do not resend", r.ID)
	}
	v, e := s.API.Status(ctx, r.RequestID)
	if e != nil {
		return r, e
	}
	r.Status = africastalking.Delivery(v)
	r.Message = v
	r.Checked = storage.Now()
	if !local {
		return r, nil
	}
	return r, s.Store.Update(ctx, r)
}
