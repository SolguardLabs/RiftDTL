package main

import "fmt"

type WithdrawalStatus string

const (
	WithdrawalRequested WithdrawalStatus = "requested"
	WithdrawalApproved  WithdrawalStatus = "approved"
	WithdrawalFinalized WithdrawalStatus = "finalized"
	WithdrawalRejected  WithdrawalStatus = "rejected"
)

type Withdrawal struct {
	ID               string           `json:"id"`
	Nonce            int64            `json:"nonce"`
	ShardID          string           `json:"shard_id"`
	AccountID        string           `json:"account_id"`
	Asset            string           `json:"asset"`
	Amount           Amount           `json:"amount"`
	Fee              Amount           `json:"fee"`
	CreatedAt        int64            `json:"created_at"`
	FinalizedAt      int64            `json:"finalized_at"`
	Status           WithdrawalStatus `json:"status"`
	Reason           string           `json:"reason,omitempty"`
	Window           string           `json:"window,omitempty"`
	SourceTransfer   string           `json:"source_transfer,omitempty"`
	BeforeDebitFinal bool             `json:"before_debit_final"`
	Tags             TagSet           `json:"tags"`
}

type WithdrawalIntent struct {
	ShardID        string
	AccountID      string
	Asset          string
	Amount         Amount
	Window         string
	SourceTransfer string
	Tags           TagSet
}

type WithdrawalStore struct {
	order []string
	items map[string]*Withdrawal
	seq   int64
}

func NewWithdrawalStore() *WithdrawalStore {
	return &WithdrawalStore{order: []string{}, items: map[string]*Withdrawal{}, seq: 1}
}

func (s *WithdrawalStore) Add(withdrawal *Withdrawal) error {
	if withdrawal == nil {
		return fmt.Errorf("nil withdrawal")
	}
	if _, ok := s.items[withdrawal.ID]; !ok {
		s.order = append(s.order, withdrawal.ID)
	}
	s.items[withdrawal.ID] = withdrawal
	if withdrawal.Nonce >= s.seq {
		s.seq = withdrawal.Nonce + 1
	}
	return nil
}

func (s *WithdrawalStore) NextNonce() int64 {
	nonce := s.seq
	s.seq++
	return nonce
}

func (s *WithdrawalStore) Get(id string) (*Withdrawal, bool) {
	withdrawal, ok := s.items[NormalizeID(id)]
	return withdrawal, ok
}

func (s *WithdrawalStore) List() []Withdrawal {
	out := make([]Withdrawal, 0, len(s.order))
	for _, id := range s.order {
		if withdrawal, ok := s.items[id]; ok {
			out = append(out, *withdrawal)
		}
	}
	return out
}

func (s *WithdrawalStore) Pointers() []*Withdrawal {
	out := make([]*Withdrawal, 0, len(s.order))
	for _, id := range s.order {
		if withdrawal, ok := s.items[id]; ok {
			out = append(out, withdrawal)
		}
	}
	return out
}

type WithdrawalEngine struct {
	Ledger *Ledger
}

func NewWithdrawalEngine(ledger *Ledger) *WithdrawalEngine {
	return &WithdrawalEngine{Ledger: ledger}
}

func (e *WithdrawalEngine) Request(intent WithdrawalIntent) (*Withdrawal, error) {
	if err := RequireAmount("withdrawal", intent.Amount); err != nil {
		return nil, err
	}
	shardID := MustID("shard", intent.ShardID)
	accountID := MustID("account", intent.AccountID)
	asset := MustID("asset", intent.Asset)
	shard, ok := e.Ledger.Shards.Get(shardID)
	if !ok {
		return nil, fmt.Errorf("missing shard %s", shardID)
	}
	if _, ok := shard.Account(accountID); !ok {
		return nil, fmt.Errorf("missing account %s on shard %s", accountID, shardID)
	}
	fee := e.Ledger.Assets.Fee(asset, intent.Amount, true)
	tags := intent.Tags
	if tags == nil {
		tags = NewTagSet()
	}
	nonce := e.Ledger.Withdrawals.NextNonce()
	withdrawal := &Withdrawal{
		ID:             WithdrawalKey(shardID, accountID, asset, nonce),
		Nonce:          nonce,
		ShardID:        shardID,
		AccountID:      accountID,
		Asset:          asset,
		Amount:         intent.Amount,
		Fee:            fee,
		CreatedAt:      e.Ledger.Clock.Snapshot(),
		Status:         WithdrawalRequested,
		Window:         intent.Window,
		SourceTransfer: NormalizeID(intent.SourceTransfer),
		Tags:           tags.Clone(),
	}
	if err := e.Ledger.Withdrawals.Add(withdrawal); err != nil {
		return nil, err
	}
	e.Ledger.Event("withdrawal.requested", withdrawal.ID, map[string]any{
		"shard_id": shardID,
		"account":  accountID,
		"asset":    asset,
		"amount":   intent.Amount,
		"window":   intent.Window,
	})
	return withdrawal, nil
}

func (e *WithdrawalEngine) Finalize(id string, beforeDebitFinal bool) (*Withdrawal, error) {
	withdrawal, ok := e.Ledger.Withdrawals.Get(id)
	if !ok {
		return nil, fmt.Errorf("missing withdrawal %s", id)
	}
	if withdrawal.Status == WithdrawalFinalized {
		return withdrawal, nil
	}
	shard, ok := e.Ledger.Shards.Get(withdrawal.ShardID)
	if !ok {
		return nil, fmt.Errorf("missing shard %s", withdrawal.ShardID)
	}
	total := withdrawal.Amount + withdrawal.Fee
	if err := shard.Withdraw(withdrawal.AccountID, withdrawal.Asset, total); err != nil {
		withdrawal.Status = WithdrawalRejected
		withdrawal.Reason = err.Error()
		e.Ledger.Event("withdrawal.rejected", withdrawal.ID, map[string]any{
			"shard_id": withdrawal.ShardID,
			"account":  withdrawal.AccountID,
			"asset":    withdrawal.Asset,
			"amount":   withdrawal.Amount,
			"window":   withdrawal.Window,
		})
		return withdrawal, err
	}
	withdrawal.Status = WithdrawalFinalized
	withdrawal.FinalizedAt = e.Ledger.Clock.Snapshot()
	withdrawal.BeforeDebitFinal = beforeDebitFinal
	if withdrawal.SourceTransfer != "" {
		if transfer, ok := e.Ledger.Transfers.Get(withdrawal.SourceTransfer); ok {
			transfer.AttachWithdrawal(withdrawal.Amount)
		}
	}
	e.Ledger.Event("withdrawal.finalized", withdrawal.ID, map[string]any{
		"shard_id": withdrawal.ShardID,
		"account":  withdrawal.AccountID,
		"asset":    withdrawal.Asset,
		"amount":   withdrawal.Amount,
		"window":   withdrawal.Window,
	})
	return withdrawal, nil
}

func (e *WithdrawalEngine) RequestAndFinalize(intent WithdrawalIntent, beforeDebitFinal bool) (*Withdrawal, error) {
	withdrawal, err := e.Request(intent)
	if err != nil {
		return nil, err
	}
	return e.Finalize(withdrawal.ID, beforeDebitFinal)
}
