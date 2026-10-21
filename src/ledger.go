package main

import "fmt"

type Ledger struct {
	Protocol      string            `json:"protocol"`
	NetworkID     string            `json:"network_id"`
	Clock         *Clock            `json:"-"`
	Assets        *AssetStore       `json:"-"`
	Shards        *ShardStore       `json:"-"`
	Transfers     *TransferStore    `json:"-"`
	Withdrawals   *WithdrawalStore  `json:"-"`
	Closures      *ClosureStore     `json:"-"`
	Journal       *Journal          `json:"-"`
	Queue         *ReconcileQueue   `json:"-"`
	Notes         []string          `json:"notes"`
	OpeningSupply map[string]Amount `json:"opening_supply"`
}

func NewLedger() *Ledger {
	return &Ledger{
		Protocol:      "RiftDTL",
		NetworkID:     "rift-local-shardnet",
		Clock:         NewClock(),
		Assets:        NewAssetStore(),
		Shards:        NewShardStore(),
		Transfers:     NewTransferStore(),
		Withdrawals:   NewWithdrawalStore(),
		Closures:      NewClosureStore(),
		Journal:       NewJournal(),
		Queue:         NewReconcileQueue(),
		Notes:         []string{},
		OpeningSupply: map[string]Amount{},
	}
}

func (l *Ledger) Advance(delta int64) int64 {
	return l.Clock.Advance(delta)
}

func (l *Ledger) Tick() int64 {
	return l.Clock.Tick()
}

func (l *Ledger) AddAsset(asset Asset) {
	l.Assets.MustAdd(asset)
	l.Event("asset.added", asset.ID, map[string]any{"asset": asset.ID})
}

func (l *Ledger) AddShard(shard *Shard) {
	shard.OpenedAt = l.Clock.Snapshot()
	l.Shards.MustAdd(shard)
	l.Event("shard.added", shard.ID, map[string]any{"shard_id": shard.ID})
}

func (l *Ledger) AddAccount(shardID string, account *Account) {
	shard := l.Shards.MustGet(shardID)
	shard.MustAddAccount(account)
	l.Event("account.added", account.ID, map[string]any{
		"shard_id": shard.ID,
		"account":  account.ID,
	})
}

func (l *Ledger) Event(kind string, subject string, attrs map[string]any) Event {
	return l.Journal.Add(l.Clock.Snapshot(), kind, subject, attrs)
}

func (l *Ledger) Note(value string) {
	if value == "" {
		return
	}
	l.Notes = append(l.Notes, value)
}

func (l *Ledger) SubmitTransfer(intent TransferIntent) (*Transfer, error) {
	if err := RequireAmount("transfer", intent.Amount); err != nil {
		return nil, err
	}
	if !l.Assets.Has(intent.Asset) {
		return nil, fmt.Errorf("missing asset %s", intent.Asset)
	}
	source, ok := l.Shards.Get(intent.SourceShard)
	if !ok {
		return nil, fmt.Errorf("missing source shard %s", intent.SourceShard)
	}
	dest, ok := l.Shards.Get(intent.DestShard)
	if !ok {
		return nil, fmt.Errorf("missing destination shard %s", intent.DestShard)
	}
	if !source.CanAcceptTransfers() {
		return nil, fmt.Errorf("source shard %s cannot accept transfers in status %s", source.ID, source.Status)
	}
	if _, ok := source.Account(intent.FromAccount); !ok {
		return nil, fmt.Errorf("missing source account %s", intent.FromAccount)
	}
	if _, ok := dest.Account(intent.ToAccount); !ok {
		return nil, fmt.Errorf("missing destination account %s", intent.ToAccount)
	}
	nonce := l.Transfers.NextNonce()
	fee := l.Assets.Fee(intent.Asset, intent.Amount, false)
	transfer := NewTransfer(intent, nonce, fee, l.Clock.Snapshot())
	if err := l.Transfers.Add(transfer); err != nil {
		return nil, err
	}
	l.Queue.AddTransfer(transfer)
	l.Event("transfer.submitted", transfer.ID, map[string]any{
		"transfer": transfer.ID,
		"shard_id": transfer.SourceShard,
		"account":  transfer.FromAccount,
		"asset":    transfer.Asset,
		"amount":   transfer.Amount,
	})
	return transfer, nil
}

func (l *Ledger) TransfersForShard(shardID string) int {
	shardID = NormalizeID(shardID)
	total := 0
	for _, transfer := range l.Transfers.Pointers() {
		if transfer.SourceShard == shardID || transfer.DestShard == shardID {
			total++
		}
	}
	return total
}

func (l *Ledger) CaptureOpeningSupply() {
	out := map[string]Amount{}
	for _, asset := range l.Assets.IDs() {
		out[asset] = l.Shards.TotalAvailable(asset) + l.Shards.TotalHeld(asset)
	}
	l.OpeningSupply = out
}

func (l *Ledger) Digest() string {
	return DigestJSON(map[string]any{
		"clock":       l.Clock.Snapshot(),
		"assets":      l.Assets.List(),
		"shards":      l.Shards.Reports(),
		"accounts":    l.Shards.Accounts(),
		"transfers":   l.Transfers.List(),
		"withdrawals": l.Withdrawals.List(),
		"closures":    l.Closures.List(),
		"events":      l.Journal.List(),
	})
}
