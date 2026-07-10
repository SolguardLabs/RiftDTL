package main

import "fmt"

type AccountBook struct {
	order []string
	items map[string]*Account
}

func NewAccountBook() *AccountBook {
	return &AccountBook{order: []string{}, items: map[string]*Account{}}
}

func (b *AccountBook) Add(account *Account) error {
	if account == nil {
		return fmt.Errorf("nil account")
	}
	if account.ID == "" {
		return fmt.Errorf("account id required")
	}
	if _, ok := b.items[account.ID]; !ok {
		b.order = append(b.order, account.ID)
	}
	b.items[account.ID] = account
	return nil
}

func (b *AccountBook) MustAdd(account *Account) {
	if err := b.Add(account); err != nil {
		panic(err)
	}
}

func (b *AccountBook) Get(id string) (*Account, bool) {
	account, ok := b.items[NormalizeID(id)]
	return account, ok
}

func (b *AccountBook) MustGet(id string) *Account {
	account, ok := b.Get(id)
	if !ok {
		panic(fmt.Sprintf("missing account %s", id))
	}
	return account
}

func (b *AccountBook) Has(id string) bool {
	_, ok := b.Get(id)
	return ok
}

func (b *AccountBook) List() []*Account {
	out := make([]*Account, 0, len(b.order))
	for _, id := range b.order {
		if account, ok := b.items[id]; ok {
			out = append(out, account)
		}
	}
	return out
}

func (b *AccountBook) Reports() []AccountReport {
	accounts := b.List()
	out := make([]AccountReport, 0, len(accounts))
	for _, account := range accounts {
		out = append(out, account.Snapshot())
	}
	return out
}

func (b *AccountBook) IDs() []string {
	return CloneStrings(b.order)
}

func (b *AccountBook) Total(asset string) Amount {
	total := Amount(0)
	for _, account := range b.List() {
		total += account.Available(asset)
		total += account.Held(asset)
		total += account.PendingIn(asset)
	}
	return total
}

func (b *AccountBook) TotalAvailable(asset string) Amount {
	total := Amount(0)
	for _, account := range b.List() {
		total += account.Available(asset)
	}
	return total
}

func (b *AccountBook) TotalHeld(asset string) Amount {
	total := Amount(0)
	for _, account := range b.List() {
		total += account.Held(asset)
	}
	return total
}

func (b *AccountBook) TotalPendingIn(asset string) Amount {
	total := Amount(0)
	for _, account := range b.List() {
		total += account.PendingIn(asset)
	}
	return total
}

func (b *AccountBook) TotalPendingOut(asset string) Amount {
	total := Amount(0)
	for _, account := range b.List() {
		total += account.PendingOut(asset)
	}
	return total
}

func (b *AccountBook) TotalWithdrawn(asset string) Amount {
	total := Amount(0)
	for _, account := range b.List() {
		total += account.Withdrawn(asset)
	}
	return total
}

func (b *AccountBook) TotalExternalized(asset string) Amount {
	total := Amount(0)
	for _, account := range b.List() {
		total += account.Externalized(asset)
	}
	return total
}

func (b *AccountBook) FreezeAll(reason string) {
	for _, account := range b.List() {
		account.Freeze(reason)
	}
}

func (b *AccountBook) UnfreezeAll(reason string) {
	for _, account := range b.List() {
		account.Unfreeze(reason)
	}
}
