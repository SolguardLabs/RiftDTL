package main

import "fmt"

type TransferStatus string

const (
	TransferQueued        TransferStatus = "queued"
	TransferCreditApplied TransferStatus = "credit_applied"
	TransferDebitApplied  TransferStatus = "debit_applied"
	TransferFinalized     TransferStatus = "finalized"
	TransferDebitFailed   TransferStatus = "debit_failed"
	TransferRejected      TransferStatus = "rejected"
)

type Transfer struct {
	ID             string         `json:"id"`
	Nonce          int64          `json:"nonce"`
	SourceShard    string         `json:"source_shard"`
	DestShard      string         `json:"dest_shard"`
	FromAccount    string         `json:"from_account"`
	ToAccount      string         `json:"to_account"`
	Asset          string         `json:"asset"`
	Amount         Amount         `json:"amount"`
	Fee            Amount         `json:"fee"`
	CreatedAt      int64          `json:"created_at"`
	Arrival        int64          `json:"arrival"`
	CreditRound    int64          `json:"credit_round"`
	DebitRound     int64          `json:"debit_round"`
	CreditApplied  bool           `json:"credit_applied"`
	DebitApplied   bool           `json:"debit_applied"`
	Withdrawn      Amount         `json:"withdrawn"`
	Status         TransferStatus `json:"status"`
	Reason         string         `json:"reason,omitempty"`
	CreditGroup    string         `json:"credit_group"`
	DebitQueueKey  string         `json:"debit_queue_key"`
	RevertRequired bool           `json:"revert_required"`
	Tags           TagSet         `json:"tags"`
}

type TransferIntent struct {
	SourceShard string
	DestShard   string
	FromAccount string
	ToAccount   string
	Asset       string
	Amount      Amount
	Arrival     int64
	Tags        TagSet
}

type TransferStore struct {
	order []string
	items map[string]*Transfer
	seq   int64
}

func NewTransferStore() *TransferStore {
	return &TransferStore{order: []string{}, items: map[string]*Transfer{}, seq: 1}
}

func NewTransfer(intent TransferIntent, nonce int64, fee Amount, createdAt int64) *Transfer {
	source := MustID("source shard", intent.SourceShard)
	dest := MustID("destination shard", intent.DestShard)
	asset := MustID("asset", intent.Asset)
	arrival := intent.Arrival
	if arrival <= 0 {
		arrival = createdAt
	}
	tags := intent.Tags
	if tags == nil {
		tags = NewTagSet()
	}
	return &Transfer{
		ID:            TransferKey(source, dest, asset, nonce),
		Nonce:         nonce,
		SourceShard:   source,
		DestShard:     dest,
		FromAccount:   MustID("from account", intent.FromAccount),
		ToAccount:     MustID("to account", intent.ToAccount),
		Asset:         asset,
		Amount:        intent.Amount,
		Fee:           fee,
		CreatedAt:     createdAt,
		Arrival:       arrival,
		Status:        TransferQueued,
		CreditGroup:   CreditGroupKey(dest, intent.ToAccount, asset),
		DebitQueueKey: DebitQueueKey(source, TransferKey(source, dest, asset, nonce)),
		Tags:          tags.Clone(),
	}
}

func (s *TransferStore) Add(transfer *Transfer) error {
	if transfer == nil {
		return fmt.Errorf("nil transfer")
	}
	if transfer.ID == "" {
		return fmt.Errorf("transfer id required")
	}
	if _, ok := s.items[transfer.ID]; !ok {
		s.order = append(s.order, transfer.ID)
	}
	s.items[transfer.ID] = transfer
	if transfer.Nonce >= s.seq {
		s.seq = transfer.Nonce + 1
	}
	return nil
}

func (s *TransferStore) NextNonce() int64 {
	nonce := s.seq
	s.seq++
	return nonce
}

func (s *TransferStore) Get(id string) (*Transfer, bool) {
	transfer, ok := s.items[NormalizeID(id)]
	return transfer, ok
}

func (s *TransferStore) MustGet(id string) *Transfer {
	transfer, ok := s.Get(id)
	if !ok {
		panic(fmt.Sprintf("missing transfer %s", id))
	}
	return transfer
}

func (s *TransferStore) List() []Transfer {
	out := make([]Transfer, 0, len(s.order))
	for _, id := range s.order {
		if transfer, ok := s.items[id]; ok {
			out = append(out, *transfer)
		}
	}
	return out
}

func (s *TransferStore) Pointers() []*Transfer {
	out := make([]*Transfer, 0, len(s.order))
	for _, id := range s.order {
		if transfer, ok := s.items[id]; ok {
			out = append(out, transfer)
		}
	}
	return out
}

func (s *TransferStore) Pending() []*Transfer {
	out := []*Transfer{}
	for _, transfer := range s.Pointers() {
		if transfer.Status == TransferQueued || transfer.Status == TransferCreditApplied {
			out = append(out, transfer)
		}
	}
	return out
}

func (s *TransferStore) ByCreditGroup() map[string][]*Transfer {
	groups := map[string][]*Transfer{}
	for _, transfer := range s.Pending() {
		if transfer.CreditApplied {
			continue
		}
		groups[transfer.CreditGroup] = append(groups[transfer.CreditGroup], transfer)
	}
	return groups
}

func (s *TransferStore) ByArrival() []*Transfer {
	transfers := s.Pending()
	SortTransfersByArrival(transfers)
	return transfers
}

func (t *Transfer) TotalDebit() Amount {
	return t.Amount + t.Fee
}

func (t *Transfer) MarkCredit(round int64) {
	t.CreditApplied = true
	t.CreditRound = round
	if t.Status == TransferQueued {
		t.Status = TransferCreditApplied
	}
}

func (t *Transfer) MarkDebit(round int64) {
	t.DebitApplied = true
	t.DebitRound = round
	if t.CreditApplied {
		t.Status = TransferFinalized
	} else {
		t.Status = TransferDebitApplied
	}
}

func (t *Transfer) MarkDebitFailed(reason string) {
	t.Status = TransferDebitFailed
	t.Reason = reason
	if t.CreditApplied {
		t.RevertRequired = true
	}
}

func (t *Transfer) MarkRejected(reason string) {
	t.Status = TransferRejected
	t.Reason = reason
}

func (t *Transfer) AttachWithdrawal(amount Amount) {
	t.Withdrawn += amount
}

func SortTransfersByArrival(values []*Transfer) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			a := values[i]
			b := values[j]
			if b.Arrival < a.Arrival || (b.Arrival == a.Arrival && b.ID < a.ID) {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

func SortTransfersByGroup(values []*Transfer) {
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			a := values[i]
			b := values[j]
			if b.CreditGroup < a.CreditGroup || (b.CreditGroup == a.CreditGroup && b.ID < a.ID) {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}
