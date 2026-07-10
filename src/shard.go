package main

import "fmt"

type ShardStatus string

const (
	ShardOpen    ShardStatus = "open"
	ShardClosing ShardStatus = "closing"
	ShardClosed  ShardStatus = "closed"
	ShardDrained ShardStatus = "drained"
)

type Shard struct {
	ID               string            `json:"id"`
	Name             string            `json:"name"`
	Region           string            `json:"region"`
	Status           ShardStatus       `json:"status"`
	Priority         int               `json:"priority"`
	Revision         int64             `json:"revision"`
	OpenedAt         int64             `json:"opened_at"`
	ClosedAt         int64             `json:"closed_at"`
	Accounts         *AccountBook      `json:"-"`
	SettlementBuffer map[string]Amount `json:"settlement_buffer"`
	PendingCredit    map[string]Amount `json:"pending_credit"`
	PendingDebit     map[string]Amount `json:"pending_debit"`
	Tags             TagSet            `json:"tags"`
	Notes            []string          `json:"notes"`
	Certificate      string            `json:"certificate,omitempty"`
}

type ShardReport struct {
	ID               string         `json:"id"`
	Name             string         `json:"name"`
	Region           string         `json:"region"`
	Status           ShardStatus    `json:"status"`
	Priority         int            `json:"priority"`
	Revision         int64          `json:"revision"`
	OpenedAt         int64          `json:"opened_at"`
	ClosedAt         int64          `json:"closed_at"`
	AccountCount     int            `json:"account_count"`
	Available        []AmountBucket `json:"available"`
	Held             []AmountBucket `json:"held"`
	PendingCredit    []AmountBucket `json:"pending_credit"`
	PendingDebit     []AmountBucket `json:"pending_debit"`
	SettlementBuffer []AmountBucket `json:"settlement_buffer"`
	Withdrawn        []AmountBucket `json:"withdrawn"`
	Externalized     []AmountBucket `json:"externalized"`
	Certificate      string         `json:"certificate,omitempty"`
	Tags             TagSet         `json:"tags"`
	Notes            []string       `json:"notes"`
}

func NewShard(id string, name string, region string, priority int) *Shard {
	return &Shard{
		ID:               MustID("shard", id),
		Name:             name,
		Region:           region,
		Status:           ShardOpen,
		Priority:         priority,
		Revision:         1,
		OpenedAt:         0,
		ClosedAt:         0,
		Accounts:         NewAccountBook(),
		SettlementBuffer: map[string]Amount{},
		PendingCredit:    map[string]Amount{},
		PendingDebit:     map[string]Amount{},
		Tags:             NewTagSet(),
		Notes:            []string{},
	}
}

func (s *Shard) CanAcceptTransfers() bool {
	return s.Status == ShardOpen
}

func (s *Shard) AddAccount(account *Account) error {
	if account == nil {
		return fmt.Errorf("nil account")
	}
	if account.ShardID != s.ID {
		return fmt.Errorf("account shard mismatch: account=%s account_shard=%s shard=%s", account.ID, account.ShardID, s.ID)
	}
	if err := s.Accounts.Add(account); err != nil {
		return err
	}
	s.Revision++
	return nil
}

func (s *Shard) MustAddAccount(account *Account) {
	if err := s.AddAccount(account); err != nil {
		panic(err)
	}
}

func (s *Shard) Account(id string) (*Account, bool) {
	return s.Accounts.Get(id)
}

func (s *Shard) MustAccount(id string) *Account {
	return s.Accounts.MustGet(id)
}

func (s *Shard) Credit(accountID string, asset string, amount Amount) error {
	account, ok := s.Account(accountID)
	if !ok {
		return fmt.Errorf("missing account %s on shard %s", accountID, s.ID)
	}
	if err := account.Credit(asset, amount); err != nil {
		return err
	}
	AddBucket(s.PendingCredit, asset, amount)
	s.Revision++
	return nil
}

func (s *Shard) Debit(accountID string, asset string, amount Amount) error {
	account, ok := s.Account(accountID)
	if !ok {
		return fmt.Errorf("missing account %s on shard %s", accountID, s.ID)
	}
	if err := account.Debit(asset, amount); err != nil {
		return err
	}
	AddBucket(s.PendingDebit, asset, amount)
	s.Revision++
	return nil
}

func (s *Shard) Hold(accountID string, asset string, amount Amount) error {
	account, ok := s.Account(accountID)
	if !ok {
		return fmt.Errorf("missing account %s on shard %s", accountID, s.ID)
	}
	if err := account.Hold(asset, amount); err != nil {
		return err
	}
	AddBucket(s.PendingDebit, asset, amount)
	s.Revision++
	return nil
}

func (s *Shard) Withdraw(accountID string, asset string, amount Amount) error {
	account, ok := s.Account(accountID)
	if !ok {
		return fmt.Errorf("missing account %s on shard %s", accountID, s.ID)
	}
	if err := account.Withdraw(asset, amount); err != nil {
		return err
	}
	s.Revision++
	return nil
}

func (s *Shard) Available(asset string) Amount {
	return s.Accounts.TotalAvailable(asset)
}

func (s *Shard) Held(asset string) Amount {
	return s.Accounts.TotalHeld(asset)
}

func (s *Shard) Withdrawn(asset string) Amount {
	return s.Accounts.TotalWithdrawn(asset)
}

func (s *Shard) Externalized(asset string) Amount {
	return s.Accounts.TotalExternalized(asset)
}

func (s *Shard) BeginClose(clock int64, reason string) {
	if s.Status == ShardClosed || s.Status == ShardDrained {
		return
	}
	s.Status = ShardClosing
	s.ClosedAt = clock
	s.Note(reason)
	s.Revision++
}

func (s *Shard) FinalizeClose(clock int64, certificate string) {
	s.Status = ShardClosed
	s.ClosedAt = clock
	s.Certificate = certificate
	s.Revision++
}

func (s *Shard) Drain(clock int64, reason string) {
	s.Status = ShardDrained
	s.ClosedAt = clock
	s.Note(reason)
	s.Revision++
}

func (s *Shard) Note(value string) {
	if value == "" {
		return
	}
	s.Notes = append(s.Notes, value)
}

func (s *Shard) AssetIDs() []string {
	values := map[string]bool{}
	for _, account := range s.Accounts.List() {
		for _, asset := range account.BalanceAssets() {
			values[asset] = true
		}
	}
	for asset := range s.PendingCredit {
		values[asset] = true
	}
	for asset := range s.PendingDebit {
		values[asset] = true
	}
	keys := make([]string, 0, len(values))
	for asset := range values {
		keys = append(keys, asset)
	}
	SortStrings(keys)
	return keys
}

func (s *Shard) Report() ShardReport {
	assets := s.AssetIDs()
	available := []AmountBucket{}
	held := []AmountBucket{}
	pendingCredit := []AmountBucket{}
	pendingDebit := []AmountBucket{}
	buffer := []AmountBucket{}
	withdrawn := []AmountBucket{}
	externalized := []AmountBucket{}
	for _, asset := range assets {
		available = append(available, Bucket(asset, s.Available(asset)))
		held = append(held, Bucket(asset, s.Held(asset)))
		pendingCredit = append(pendingCredit, Bucket(asset, s.PendingCredit[asset]))
		pendingDebit = append(pendingDebit, Bucket(asset, s.PendingDebit[asset]))
		buffer = append(buffer, Bucket(asset, s.SettlementBuffer[asset]))
		withdrawn = append(withdrawn, Bucket(asset, s.Withdrawn(asset)))
		externalized = append(externalized, Bucket(asset, s.Externalized(asset)))
	}
	return ShardReport{
		ID:               s.ID,
		Name:             s.Name,
		Region:           s.Region,
		Status:           s.Status,
		Priority:         s.Priority,
		Revision:         s.Revision,
		OpenedAt:         s.OpenedAt,
		ClosedAt:         s.ClosedAt,
		AccountCount:     len(s.Accounts.order),
		Available:        available,
		Held:             held,
		PendingCredit:    pendingCredit,
		PendingDebit:     pendingDebit,
		SettlementBuffer: buffer,
		Withdrawn:        withdrawn,
		Externalized:     externalized,
		Certificate:      s.Certificate,
		Tags:             s.Tags.Clone(),
		Notes:            CloneStrings(s.Notes),
	}
}
