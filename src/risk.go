package main

type RiskMetrics struct {
	OpeningSupply           []AmountBucket `json:"opening_supply"`
	CurrentSupply           []AmountBucket `json:"current_supply"`
	SupplyDelta             []AmountBucket `json:"supply_delta"`
	CreditsApplied          []AmountBucket `json:"credits_applied"`
	DebitsApplied           []AmountBucket `json:"debits_applied"`
	UnbackedCredits         []AmountBucket `json:"unbacked_credits"`
	WithdrawnBeforeDebit    []AmountBucket `json:"withdrawn_before_debit"`
	WithdrawnUnbacked       []AmountBucket `json:"withdrawn_unbacked"`
	Externalized            []AmountBucket `json:"externalized"`
	FailedDebitCredits      int            `json:"failed_debit_credits"`
	CheckpointWithdrawals   int            `json:"checkpoint_withdrawals"`
	CreditBeforeDebitRounds int            `json:"credit_before_debit_rounds"`
	PhantomLiquidity        Amount         `json:"phantom_liquidity"`
	RetirablePhantom        Amount         `json:"retirable_phantom"`
	MaxSingleFailedDebit    Amount         `json:"max_single_failed_debit"`
}

type InvariantReport struct {
	AccountsNonNegative           bool `json:"accounts_non_negative"`
	ShardLinksValid               bool `json:"shard_links_valid"`
	TransferLinksValid            bool `json:"transfer_links_valid"`
	ClosureLinksValid             bool `json:"closure_links_valid"`
	NoUnbackedCredit              bool `json:"no_unbacked_credit"`
	WithdrawalsBackedByFinalDebit bool `json:"withdrawals_backed_by_final_debit"`
	NoSupplyExpansion             bool `json:"no_supply_expansion"`
	GroupedCreditCheckpoint       bool `json:"grouped_credit_checkpoint"`
	ExploitReachable              bool `json:"exploit_reachable"`
}

type RiskEngine struct {
	Ledger          *Ledger
	Reconciliations []ReconcileResult
}

func NewRiskEngine(ledger *Ledger, reconciliations []ReconcileResult) *RiskEngine {
	return &RiskEngine{Ledger: ledger, Reconciliations: reconciliations}
}

func (r *RiskEngine) Metrics() RiskMetrics {
	credits := map[string]Amount{}
	debits := map[string]Amount{}
	unbacked := map[string]Amount{}
	withdrawnBefore := map[string]Amount{}
	withdrawnUnbacked := map[string]Amount{}
	current := map[string]Amount{}
	delta := map[string]Amount{}
	externalized := map[string]Amount{}
	failedDebitCredits := 0
	maxFailed := Amount(0)
	for _, transfer := range r.Ledger.Transfers.Pointers() {
		if transfer.CreditApplied {
			AddBucket(credits, transfer.Asset, transfer.Amount)
		}
		if transfer.DebitApplied {
			AddBucket(debits, transfer.Asset, transfer.Amount)
		}
		if transfer.CreditApplied && !transfer.DebitApplied {
			AddBucket(unbacked, transfer.Asset, transfer.Amount)
			failedDebitCredits++
			if transfer.Amount > maxFailed {
				maxFailed = transfer.Amount
			}
		}
	}
	for _, withdrawal := range r.Ledger.Withdrawals.Pointers() {
		if withdrawal.Status != WithdrawalFinalized {
			continue
		}
		AddBucket(externalized, withdrawal.Asset, withdrawal.Amount)
		if withdrawal.BeforeDebitFinal {
			AddBucket(withdrawnBefore, withdrawal.Asset, withdrawal.Amount)
		}
		if transfer, ok := r.Ledger.Transfers.Get(withdrawal.SourceTransfer); ok {
			if transfer.CreditApplied && !transfer.DebitApplied {
				AddBucket(withdrawnUnbacked, withdrawal.Asset, withdrawal.Amount)
			}
		}
	}
	for _, asset := range r.Ledger.Assets.IDs() {
		credits[asset] += 0
		debits[asset] += 0
		unbacked[asset] += 0
		withdrawnBefore[asset] += 0
		withdrawnUnbacked[asset] += 0
		externalized[asset] += 0
		current[asset] = r.Ledger.Shards.TotalAvailable(asset) + r.Ledger.Shards.TotalHeld(asset) + r.Ledger.Shards.TotalExternalized(asset)
		delta[asset] = current[asset] - r.Ledger.OpeningSupply[asset]
	}
	checkpointWithdrawals := 0
	creditBeforeDebitRounds := 0
	for _, result := range r.Reconciliations {
		if result.CreditBeforeDebit {
			creditBeforeDebitRounds++
		}
		checkpointWithdrawals += len(result.Withdrawals)
	}
	phantom := SumBucketList(BucketsFromMap(unbacked))
	retirable := SumBucketList(BucketsFromMap(withdrawnUnbacked))
	return RiskMetrics{
		OpeningSupply:           BucketsFromMap(r.Ledger.OpeningSupply),
		CurrentSupply:           BucketsFromMap(current),
		SupplyDelta:             BucketsFromMap(delta),
		CreditsApplied:          BucketsFromMap(credits),
		DebitsApplied:           BucketsFromMap(debits),
		UnbackedCredits:         BucketsFromMap(unbacked),
		WithdrawnBeforeDebit:    BucketsFromMap(withdrawnBefore),
		WithdrawnUnbacked:       BucketsFromMap(withdrawnUnbacked),
		Externalized:            BucketsFromMap(externalized),
		FailedDebitCredits:      failedDebitCredits,
		CheckpointWithdrawals:   checkpointWithdrawals,
		CreditBeforeDebitRounds: creditBeforeDebitRounds,
		PhantomLiquidity:        phantom,
		RetirablePhantom:        retirable,
		MaxSingleFailedDebit:    maxFailed,
	}
}

func (r *RiskEngine) Invariants() InvariantReport {
	metrics := r.Metrics()
	noUnbacked := metrics.PhantomLiquidity == 0
	withdrawalsBacked := metrics.RetirablePhantom == 0
	noSupplyExpansion := true
	for _, bucket := range metrics.SupplyDelta {
		if bucket.Amount > 0 {
			noSupplyExpansion = false
			break
		}
	}
	grouped := metrics.CreditBeforeDebitRounds > 0
	exploit := metrics.RetirablePhantom > 0 && metrics.FailedDebitCredits > 0
	return InvariantReport{
		AccountsNonNegative:           r.accountsNonNegative(),
		ShardLinksValid:               r.shardLinksValid(),
		TransferLinksValid:            r.transferLinksValid(),
		ClosureLinksValid:             r.closureLinksValid(),
		NoUnbackedCredit:              noUnbacked,
		WithdrawalsBackedByFinalDebit: withdrawalsBacked,
		NoSupplyExpansion:             noSupplyExpansion,
		GroupedCreditCheckpoint:       grouped,
		ExploitReachable:              exploit,
	}
}

func (r *RiskEngine) accountsNonNegative() bool {
	for _, account := range r.Ledger.Shards.Accounts() {
		for _, balance := range account.Balances {
			if balance.Available < 0 || balance.Held < 0 || balance.PendingIn < 0 || balance.PendingOut < 0 {
				return false
			}
		}
	}
	return true
}

func (r *RiskEngine) shardLinksValid() bool {
	for _, shard := range r.Ledger.Shards.List() {
		for _, account := range shard.Accounts.List() {
			if account.ShardID != shard.ID {
				return false
			}
		}
	}
	return true
}

func (r *RiskEngine) transferLinksValid() bool {
	for _, transfer := range r.Ledger.Transfers.Pointers() {
		source, ok := r.Ledger.Shards.Get(transfer.SourceShard)
		if !ok {
			return false
		}
		dest, ok := r.Ledger.Shards.Get(transfer.DestShard)
		if !ok {
			return false
		}
		if _, ok := source.Account(transfer.FromAccount); !ok {
			return false
		}
		if _, ok := dest.Account(transfer.ToAccount); !ok {
			return false
		}
		if !r.Ledger.Assets.Has(transfer.Asset) {
			return false
		}
	}
	return true
}

func (r *RiskEngine) closureLinksValid() bool {
	for _, cert := range r.Ledger.Closures.List() {
		if _, ok := r.Ledger.Shards.Get(cert.ShardID); !ok {
			return false
		}
	}
	return true
}
