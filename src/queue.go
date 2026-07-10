package main

type QueueSide string

const (
	QueueCredit QueueSide = "credit"
	QueueDebit  QueueSide = "debit"
)

type QueueItem struct {
	ID        string    `json:"id"`
	Transfer  string    `json:"transfer"`
	Side      QueueSide `json:"side"`
	ShardID   string    `json:"shard_id"`
	AccountID string    `json:"account_id"`
	Asset     string    `json:"asset"`
	Amount    Amount    `json:"amount"`
	Arrival   int64     `json:"arrival"`
	GroupKey  string    `json:"group_key"`
	Priority  int       `json:"priority"`
}

type CreditGroup struct {
	Key       string   `json:"key"`
	ShardID   string   `json:"shard_id"`
	AccountID string   `json:"account_id"`
	Asset     string   `json:"asset"`
	Amount    Amount   `json:"amount"`
	Transfers []string `json:"transfers"`
}

type DebitApplication struct {
	Transfer string `json:"transfer"`
	ShardID  string `json:"shard_id"`
	Account  string `json:"account"`
	Asset    string `json:"asset"`
	Amount   Amount `json:"amount"`
	Fee      Amount `json:"fee"`
	Arrival  int64  `json:"arrival"`
	Applied  bool   `json:"applied"`
	Reason   string `json:"reason,omitempty"`
}

type ReconcileQueue struct {
	credit []QueueItem
	debit  []QueueItem
}

func NewReconcileQueue() *ReconcileQueue {
	return &ReconcileQueue{credit: []QueueItem{}, debit: []QueueItem{}}
}

func (q *ReconcileQueue) AddTransfer(transfer *Transfer) {
	if transfer == nil {
		return
	}
	q.credit = append(q.credit, QueueItem{
		ID:        "credit-" + transfer.ID,
		Transfer:  transfer.ID,
		Side:      QueueCredit,
		ShardID:   transfer.DestShard,
		AccountID: transfer.ToAccount,
		Asset:     transfer.Asset,
		Amount:    transfer.Amount,
		Arrival:   transfer.Arrival,
		GroupKey:  transfer.CreditGroup,
		Priority:  0,
	})
	q.debit = append(q.debit, QueueItem{
		ID:        "debit-" + transfer.ID,
		Transfer:  transfer.ID,
		Side:      QueueDebit,
		ShardID:   transfer.SourceShard,
		AccountID: transfer.FromAccount,
		Asset:     transfer.Asset,
		Amount:    transfer.TotalDebit(),
		Arrival:   transfer.Arrival,
		GroupKey:  transfer.DebitQueueKey,
		Priority:  0,
	})
}

func (q *ReconcileQueue) Credits() []QueueItem {
	out := make([]QueueItem, len(q.credit))
	copy(out, q.credit)
	return out
}

func (q *ReconcileQueue) Debits() []QueueItem {
	out := make([]QueueItem, len(q.debit))
	copy(out, q.debit)
	return out
}

func (q *ReconcileQueue) RemoveTransfer(id string) {
	q.credit = removeQueueItem(q.credit, id)
	q.debit = removeQueueItem(q.debit, id)
}

func removeQueueItem(values []QueueItem, transferID string) []QueueItem {
	out := values[:0]
	for _, value := range values {
		if value.Transfer != transferID {
			out = append(out, value)
		}
	}
	return out
}

func BuildCreditGroups(transfers []*Transfer) []CreditGroup {
	SortTransfersByGroup(transfers)
	byKey := map[string]*CreditGroup{}
	order := []string{}
	for _, transfer := range transfers {
		if transfer.CreditApplied {
			continue
		}
		group, ok := byKey[transfer.CreditGroup]
		if !ok {
			group = &CreditGroup{
				Key:       transfer.CreditGroup,
				ShardID:   transfer.DestShard,
				AccountID: transfer.ToAccount,
				Asset:     transfer.Asset,
				Transfers: []string{},
			}
			byKey[transfer.CreditGroup] = group
			order = append(order, transfer.CreditGroup)
		}
		group.Amount += transfer.Amount
		group.Transfers = append(group.Transfers, transfer.ID)
	}
	SortStrings(order)
	out := make([]CreditGroup, 0, len(order))
	for _, key := range order {
		out = append(out, *byKey[key])
	}
	return out
}
