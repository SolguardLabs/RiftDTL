package main

import "fmt"

type ClosureStatus string

const (
	ClosureStarted   ClosureStatus = "started"
	ClosureCertified ClosureStatus = "certified"
	ClosureRejected  ClosureStatus = "rejected"
)

type ClosureCertificate struct {
	ID             string         `json:"id"`
	ShardID        string         `json:"shard_id"`
	OpenedAt       int64          `json:"opened_at"`
	ClosedAt       int64          `json:"closed_at"`
	Status         ClosureStatus  `json:"status"`
	AccountCount   int            `json:"account_count"`
	TransferCount  int            `json:"transfer_count"`
	PendingCredits []AmountBucket `json:"pending_credits"`
	PendingDebits  []AmountBucket `json:"pending_debits"`
	StateDigest    string         `json:"state_digest"`
	Reason         string         `json:"reason,omitempty"`
}

type ClosureStore struct {
	order []string
	items map[string]*ClosureCertificate
}

func NewClosureStore() *ClosureStore {
	return &ClosureStore{order: []string{}, items: map[string]*ClosureCertificate{}}
}

func (s *ClosureStore) Add(cert *ClosureCertificate) error {
	if cert == nil {
		return fmt.Errorf("nil closure certificate")
	}
	if _, ok := s.items[cert.ID]; !ok {
		s.order = append(s.order, cert.ID)
	}
	s.items[cert.ID] = cert
	return nil
}

func (s *ClosureStore) List() []ClosureCertificate {
	out := make([]ClosureCertificate, 0, len(s.order))
	for _, id := range s.order {
		if cert, ok := s.items[id]; ok {
			out = append(out, *cert)
		}
	}
	return out
}

type ClosureEngine struct {
	Ledger *Ledger
}

func NewClosureEngine(ledger *Ledger) *ClosureEngine {
	return &ClosureEngine{Ledger: ledger}
}

func (e *ClosureEngine) Begin(shardID string, reason string) error {
	shard, ok := e.Ledger.Shards.Get(shardID)
	if !ok {
		return fmt.Errorf("missing shard %s", shardID)
	}
	shard.BeginClose(e.Ledger.Clock.Snapshot(), reason)
	e.Ledger.Event("shard.closing", shard.ID, map[string]any{"shard_id": shard.ID})
	return nil
}

func (e *ClosureEngine) Certify(shardID string, reason string) (*ClosureCertificate, error) {
	shard, ok := e.Ledger.Shards.Get(shardID)
	if !ok {
		return nil, fmt.Errorf("missing shard %s", shardID)
	}
	pendingCredits := BucketsFromMap(shard.PendingCredit)
	pendingDebits := BucketsFromMap(shard.PendingDebit)
	cert := &ClosureCertificate{
		ID:             ClosureKey(shard.ID, e.Ledger.Clock.Snapshot()),
		ShardID:        shard.ID,
		OpenedAt:       shard.OpenedAt,
		ClosedAt:       e.Ledger.Clock.Snapshot(),
		Status:         ClosureCertified,
		AccountCount:   len(shard.Accounts.order),
		TransferCount:  e.Ledger.TransfersForShard(shard.ID),
		PendingCredits: pendingCredits,
		PendingDebits:  pendingDebits,
		StateDigest: DigestJSON(map[string]any{
			"shard":   shard.Report(),
			"credits": pendingCredits,
			"debits":  pendingDebits,
		}),
		Reason: reason,
	}
	shard.FinalizeClose(e.Ledger.Clock.Snapshot(), cert.ID)
	if err := e.Ledger.Closures.Add(cert); err != nil {
		return nil, err
	}
	e.Ledger.Event("shard.closed", shard.ID, map[string]any{"shard_id": shard.ID})
	return cert, nil
}
