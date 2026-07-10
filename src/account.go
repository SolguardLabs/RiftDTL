package main

import "fmt"

type Balance struct {
	Asset        string `json:"asset"`
	Available    Amount `json:"available"`
	Held         Amount `json:"held"`
	PendingIn    Amount `json:"pending_in"`
	PendingOut   Amount `json:"pending_out"`
	Withdrawn    Amount `json:"withdrawn"`
	Externalized Amount `json:"externalized"`
}

type Account struct {
	ID       string              `json:"id"`
	Owner    string              `json:"owner"`
	Role     string              `json:"role"`
	ShardID  string              `json:"shard_id"`
	Frozen   bool                `json:"frozen"`
	Sequence int64               `json:"sequence"`
	Balances map[string]*Balance `json:"balances"`
	Tags     TagSet              `json:"tags"`
	Notes    []string            `json:"notes"`
}

type AccountReport struct {
	ID           string         `json:"id"`
	Owner        string         `json:"owner"`
	Role         string         `json:"role"`
	ShardID      string         `json:"shard_id"`
	Frozen       bool           `json:"frozen"`
	Sequence     int64          `json:"sequence"`
	Balances     []Balance      `json:"balances"`
	Available    []AmountBucket `json:"available"`
	Held         []AmountBucket `json:"held"`
	PendingIn    []AmountBucket `json:"pending_in"`
	PendingOut   []AmountBucket `json:"pending_out"`
	Withdrawn    []AmountBucket `json:"withdrawn"`
	Externalized []AmountBucket `json:"externalized"`
	Tags         TagSet         `json:"tags"`
	Notes        []string       `json:"notes"`
}

func NewAccount(id string, owner string, role string, shardID string) *Account {
	return &Account{
		ID:       MustID("account", id),
		Owner:    owner,
		Role:     role,
		ShardID:  MustID("shard", shardID),
		Frozen:   false,
		Sequence: 0,
		Balances: map[string]*Balance{},
		Tags:     NewTagSet(),
		Notes:    []string{},
	}
}

func (a *Account) balance(asset string) *Balance {
	asset = NormalizeID(asset)
	if existing, ok := a.Balances[asset]; ok {
		return existing
	}
	balance := &Balance{Asset: asset}
	a.Balances[asset] = balance
	return balance
}

func (a *Account) Credit(asset string, amount Amount) error {
	if err := RequireAmount("credit", amount); err != nil {
		return err
	}
	balance := a.balance(asset)
	balance.Available += amount
	a.Sequence++
	return nil
}

func (a *Account) CreditPending(asset string, amount Amount) error {
	if err := RequireAmount("pending credit", amount); err != nil {
		return err
	}
	balance := a.balance(asset)
	balance.PendingIn += amount
	a.Sequence++
	return nil
}

func (a *Account) MakePendingAvailable(asset string, amount Amount) error {
	if err := RequireAmount("pending release", amount); err != nil {
		return err
	}
	balance := a.balance(asset)
	if balance.PendingIn < amount {
		return fmt.Errorf("pending credit below release: account=%s asset=%s pending=%d amount=%d", a.ID, asset, balance.PendingIn, amount)
	}
	balance.PendingIn -= amount
	balance.Available += amount
	a.Sequence++
	return nil
}

func (a *Account) Debit(asset string, amount Amount) error {
	if a.Frozen {
		return fmt.Errorf("account %s is frozen", a.ID)
	}
	if err := RequireAmount("debit", amount); err != nil {
		return err
	}
	balance := a.balance(asset)
	if err := RequireAvailable("account "+a.ID, balance.Available, amount); err != nil {
		return err
	}
	balance.Available -= amount
	a.Sequence++
	return nil
}

func (a *Account) Hold(asset string, amount Amount) error {
	if a.Frozen {
		return fmt.Errorf("account %s is frozen", a.ID)
	}
	if err := RequireAmount("hold", amount); err != nil {
		return err
	}
	balance := a.balance(asset)
	if err := RequireAvailable("account "+a.ID, balance.Available, amount); err != nil {
		return err
	}
	balance.Available -= amount
	balance.Held += amount
	balance.PendingOut += amount
	a.Sequence++
	return nil
}

func (a *Account) Release(asset string, amount Amount) error {
	if err := RequireAmount("release", amount); err != nil {
		return err
	}
	balance := a.balance(asset)
	if balance.Held < amount {
		return fmt.Errorf("hold below release: account=%s asset=%s held=%d amount=%d", a.ID, asset, balance.Held, amount)
	}
	balance.Held -= amount
	if balance.PendingOut >= amount {
		balance.PendingOut -= amount
	}
	balance.Available += amount
	a.Sequence++
	return nil
}

func (a *Account) ConsumeHold(asset string, amount Amount) error {
	if err := RequireAmount("consume hold", amount); err != nil {
		return err
	}
	balance := a.balance(asset)
	if balance.Held < amount {
		return fmt.Errorf("hold below debit: account=%s asset=%s held=%d amount=%d", a.ID, asset, balance.Held, amount)
	}
	balance.Held -= amount
	if balance.PendingOut >= amount {
		balance.PendingOut -= amount
	}
	a.Sequence++
	return nil
}

func (a *Account) Withdraw(asset string, amount Amount) error {
	if a.Frozen {
		return fmt.Errorf("account %s is frozen", a.ID)
	}
	if err := a.Debit(asset, amount); err != nil {
		return err
	}
	balance := a.balance(asset)
	balance.Withdrawn += amount
	balance.Externalized += amount
	return nil
}

func (a *Account) Available(asset string) Amount {
	return a.balance(asset).Available
}

func (a *Account) Held(asset string) Amount {
	return a.balance(asset).Held
}

func (a *Account) PendingIn(asset string) Amount {
	return a.balance(asset).PendingIn
}

func (a *Account) PendingOut(asset string) Amount {
	return a.balance(asset).PendingOut
}

func (a *Account) Withdrawn(asset string) Amount {
	return a.balance(asset).Withdrawn
}

func (a *Account) Externalized(asset string) Amount {
	return a.balance(asset).Externalized
}

func (a *Account) Freeze(reason string) {
	a.Frozen = true
	a.Note(reason)
}

func (a *Account) Unfreeze(reason string) {
	a.Frozen = false
	a.Note(reason)
}

func (a *Account) Note(value string) {
	if value == "" {
		return
	}
	a.Notes = append(a.Notes, value)
}

func (a *Account) BalanceAssets() []string {
	keys := make([]string, 0, len(a.Balances))
	for key := range a.Balances {
		keys = append(keys, key)
	}
	SortStrings(keys)
	return keys
}

func (a *Account) Snapshot() AccountReport {
	keys := a.BalanceAssets()
	balances := make([]Balance, 0, len(keys))
	available := []AmountBucket{}
	held := []AmountBucket{}
	pendingIn := []AmountBucket{}
	pendingOut := []AmountBucket{}
	withdrawn := []AmountBucket{}
	externalized := []AmountBucket{}
	for _, key := range keys {
		balance := *a.Balances[key]
		balances = append(balances, balance)
		available = append(available, Bucket(key, balance.Available))
		held = append(held, Bucket(key, balance.Held))
		pendingIn = append(pendingIn, Bucket(key, balance.PendingIn))
		pendingOut = append(pendingOut, Bucket(key, balance.PendingOut))
		withdrawn = append(withdrawn, Bucket(key, balance.Withdrawn))
		externalized = append(externalized, Bucket(key, balance.Externalized))
	}
	return AccountReport{
		ID:           a.ID,
		Owner:        a.Owner,
		Role:         a.Role,
		ShardID:      a.ShardID,
		Frozen:       a.Frozen,
		Sequence:     a.Sequence,
		Balances:     balances,
		Available:    available,
		Held:         held,
		PendingIn:    pendingIn,
		PendingOut:   pendingOut,
		Withdrawn:    withdrawn,
		Externalized: externalized,
		Tags:         a.Tags.Clone(),
		Notes:        CloneStrings(a.Notes),
	}
}
