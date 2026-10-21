package main

import "fmt"

type PhaseTrace struct {
	Step     int    `json:"step"`
	Phase    string `json:"phase"`
	Action   string `json:"action"`
	Subject  string `json:"subject"`
	Transfer string `json:"transfer,omitempty"`
	Group    string `json:"group,omitempty"`
	ShardID  string `json:"shard_id,omitempty"`
	Account  string `json:"account,omitempty"`
	Asset    string `json:"asset,omitempty"`
	Amount   Amount `json:"amount,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

type ReconcileResult struct {
	Name                string             `json:"name"`
	Round               int64              `json:"round"`
	Window              string             `json:"window"`
	Policy              ReconcilePolicy    `json:"policy"`
	CreditGroups        []CreditGroup      `json:"credit_groups"`
	Debits              []DebitApplication `json:"debits"`
	Withdrawals         []string           `json:"withdrawals"`
	Completed           []string           `json:"completed"`
	Failed              []string           `json:"failed"`
	Rejected            []string           `json:"rejected"`
	CreditVariance      []AmountBucket     `json:"credit_variance"`
	PublishedCheckpoint bool               `json:"published_checkpoint"`
	CreditBeforeDebit   bool               `json:"credit_before_debit"`
	Trace               []PhaseTrace       `json:"trace"`
	Notes               []string           `json:"notes"`
}

type Reconciler struct {
	Ledger *Ledger
	Policy ReconcilePolicy
	Round  int64
}

func NewReconciler(ledger *Ledger, policy ReconcilePolicy) *Reconciler {
	return &Reconciler{Ledger: ledger, Policy: policy, Round: 0}
}

func (r *Reconciler) Start(name string) *ReconcileResult {
	r.Round++
	window := fmt.Sprintf("round-%02d-%s", r.Round, NormalizeID(name))
	result := &ReconcileResult{
		Name:                name,
		Round:               r.Round,
		Window:              window,
		Policy:              r.Policy,
		CreditGroups:        []CreditGroup{},
		Debits:              []DebitApplication{},
		Withdrawals:         []string{},
		Completed:           []string{},
		Failed:              []string{},
		Rejected:            []string{},
		CreditVariance:      []AmountBucket{},
		PublishedCheckpoint: false,
		CreditBeforeDebit:   false,
		Trace:               []PhaseTrace{},
		Notes:               []string{},
	}
	result.trace("round", "start", window, "", "", "", "", 0, "reconciliation round opened")
	r.Ledger.Event("reconcile.started", window, map[string]any{"window": window})
	return result
}

func (r *Reconciler) Reconcile(name string, withdrawals []WithdrawalIntent) ReconcileResult {
	result := r.Start(name)
	r.ApplyCredits(result)
	if len(withdrawals) > 0 {
		r.ProcessCheckpointWithdrawals(result, withdrawals)
	}
	r.ApplyDebits(result)
	r.Close(result)
	return *result
}

func (r *Reconciler) ApplyCredits(result *ReconcileResult) {
	pending := r.Ledger.Transfers.Pending()
	if r.Policy.ValidateCreditBeforeDebit {
		pending = r.filterSolventTransfers(result, pending)
	}
	groups := BuildCreditGroups(pending)
	for _, group := range groups {
		shard, ok := r.Ledger.Shards.Get(group.ShardID)
		if !ok {
			result.Rejected = append(result.Rejected, group.Key)
			result.trace("credit", "reject", group.Key, "", group.Key, group.ShardID, group.AccountID, group.Amount, "destination shard missing")
			continue
		}
		if err := shard.Credit(group.AccountID, group.Asset, group.Amount); err != nil {
			result.Rejected = append(result.Rejected, group.Key)
			result.trace("credit", "reject", group.Key, "", group.Key, group.ShardID, group.AccountID, group.Amount, err.Error())
			continue
		}
		result.CreditGroups = append(result.CreditGroups, group)
		result.CreditBeforeDebit = true
		result.trace("credit", "apply-group", group.Key, "", group.Key, group.ShardID, group.AccountID, group.Amount, "destination credit published before origin debit")
		for _, transferID := range group.Transfers {
			transfer, ok := r.Ledger.Transfers.Get(transferID)
			if !ok {
				continue
			}
			transfer.MarkCredit(result.Round)
			r.Ledger.Event("credit.applied", transfer.ID, map[string]any{
				"transfer": transfer.ID,
				"shard_id": transfer.DestShard,
				"account":  transfer.ToAccount,
				"asset":    transfer.Asset,
				"amount":   transfer.Amount,
				"window":   result.Window,
			})
		}
	}
	if r.Policy.PublishCreditCheckpoint && len(groups) > 0 {
		result.PublishedCheckpoint = true
		result.trace("checkpoint", "publish", result.Window, "", "", "", "", 0, "grouped destination credits are withdrawable")
		r.Ledger.Event("checkpoint.published", result.Window, map[string]any{"window": result.Window})
	}
}

func (r *Reconciler) ProcessCheckpointWithdrawals(result *ReconcileResult, withdrawals []WithdrawalIntent) {
	if !r.Policy.AllowWithdrawalDuringCheckpoint {
		result.trace("withdrawal", "blocked", result.Window, "", "", "", "", 0, "policy blocks withdrawals during credit checkpoint")
		return
	}
	engine := NewWithdrawalEngine(r.Ledger)
	for _, intent := range withdrawals {
		if intent.Window == "" {
			intent.Window = result.Window
		}
		withdrawal, err := engine.RequestAndFinalize(intent, true)
		if err != nil {
			result.trace("withdrawal", "reject", intent.AccountID, "", "", intent.ShardID, intent.AccountID, intent.Amount, err.Error())
			continue
		}
		result.Withdrawals = append(result.Withdrawals, withdrawal.ID)
		result.trace("withdrawal", "finalize", withdrawal.ID, withdrawal.SourceTransfer, "", withdrawal.ShardID, withdrawal.AccountID, withdrawal.Amount, "withdrawal finalized before debit phase")
	}
}

func (r *Reconciler) ApplyDebits(result *ReconcileResult) {
	transfers := r.Ledger.Transfers.ByArrival()
	for _, transfer := range transfers {
		if transfer.DebitApplied || transfer.Status == TransferDebitFailed || transfer.Status == TransferRejected {
			continue
		}
		source, ok := r.Ledger.Shards.Get(transfer.SourceShard)
		if !ok {
			transfer.MarkDebitFailed("source shard missing")
			result.Failed = append(result.Failed, transfer.ID)
			result.Debits = append(result.Debits, DebitApplicationFor(transfer, false, "source shard missing"))
			continue
		}
		err := source.Debit(transfer.FromAccount, transfer.Asset, transfer.TotalDebit())
		if err != nil {
			transfer.MarkDebitFailed(err.Error())
			result.Failed = append(result.Failed, transfer.ID)
			result.Debits = append(result.Debits, DebitApplicationFor(transfer, false, err.Error()))
			result.trace("debit", "fail", transfer.ID, transfer.ID, "", transfer.SourceShard, transfer.FromAccount, transfer.TotalDebit(), err.Error())
			r.Ledger.Event("debit.failed", transfer.ID, map[string]any{
				"transfer": transfer.ID,
				"shard_id": transfer.SourceShard,
				"account":  transfer.FromAccount,
				"asset":    transfer.Asset,
				"amount":   transfer.TotalDebit(),
				"window":   result.Window,
			})
			continue
		}
		transfer.MarkDebit(result.Round)
		result.Completed = append(result.Completed, transfer.ID)
		result.Debits = append(result.Debits, DebitApplicationFor(transfer, true, ""))
		result.trace("debit", "apply", transfer.ID, transfer.ID, "", transfer.SourceShard, transfer.FromAccount, transfer.TotalDebit(), "origin debit finalized by arrival order")
		r.Ledger.Event("debit.applied", transfer.ID, map[string]any{
			"transfer": transfer.ID,
			"shard_id": transfer.SourceShard,
			"account":  transfer.FromAccount,
			"asset":    transfer.Asset,
			"amount":   transfer.TotalDebit(),
			"window":   result.Window,
		})
	}
}

func (r *Reconciler) Close(result *ReconcileResult) {
	variance := map[string]Amount{}
	for _, transfer := range r.Ledger.Transfers.Pointers() {
		if transfer.CreditApplied && !transfer.DebitApplied {
			AddBucket(variance, transfer.Asset, transfer.Amount)
		}
		if transfer.Status == TransferFinalized || transfer.Status == TransferDebitFailed || transfer.Status == TransferRejected {
			r.Ledger.Queue.RemoveTransfer(transfer.ID)
		}
	}
	result.CreditVariance = BucketsFromMap(variance)
	if len(result.CreditVariance) > 0 {
		result.trace("round", "credit-variance", result.Window, "", "", "", "", SumBucketList(result.CreditVariance), "settlement variance observed at round close")
	}
	result.trace("round", "close", result.Window, "", "", "", "", 0, "reconciliation round closed")
	r.Ledger.Event("reconcile.closed", result.Window, map[string]any{"window": result.Window})
}

func (r *Reconciler) filterSolventTransfers(result *ReconcileResult, transfers []*Transfer) []*Transfer {
	out := []*Transfer{}
	for _, transfer := range transfers {
		source, ok := r.Ledger.Shards.Get(transfer.SourceShard)
		if !ok {
			transfer.MarkRejected("source shard missing")
			result.Rejected = append(result.Rejected, transfer.ID)
			continue
		}
		account, ok := source.Account(transfer.FromAccount)
		if !ok {
			transfer.MarkRejected("source account missing")
			result.Rejected = append(result.Rejected, transfer.ID)
			continue
		}
		if account.Available(transfer.Asset) < transfer.TotalDebit() {
			transfer.MarkRejected("pre-credit solvency check failed")
			result.Rejected = append(result.Rejected, transfer.ID)
			result.trace("credit", "precheck-reject", transfer.ID, transfer.ID, transfer.CreditGroup, transfer.SourceShard, transfer.FromAccount, transfer.TotalDebit(), "strict policy rejected insolvent origin before credit")
			continue
		}
		out = append(out, transfer)
	}
	return out
}

func DebitApplicationFor(transfer *Transfer, applied bool, reason string) DebitApplication {
	return DebitApplication{
		Transfer: transfer.ID,
		ShardID:  transfer.SourceShard,
		Account:  transfer.FromAccount,
		Asset:    transfer.Asset,
		Amount:   transfer.Amount,
		Fee:      transfer.Fee,
		Arrival:  transfer.Arrival,
		Applied:  applied,
		Reason:   reason,
	}
}

func (r *ReconcileResult) trace(phase string, action string, subject string, transfer string, group string, shardID string, account string, amount Amount, detail string) {
	r.Trace = append(r.Trace, PhaseTrace{
		Step:     len(r.Trace) + 1,
		Phase:    phase,
		Action:   action,
		Subject:  subject,
		Transfer: transfer,
		Group:    group,
		ShardID:  shardID,
		Account:  account,
		Amount:   amount,
		Detail:   detail,
	})
}

func SumBucketList(values []AmountBucket) Amount {
	total := Amount(0)
	for _, value := range values {
		total += value.Amount
	}
	return total
}
