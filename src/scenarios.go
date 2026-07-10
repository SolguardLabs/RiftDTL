package main

func scenarioBaseline() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := NewScenarioRun("baseline", ledger, DefaultScenarioPolicy("baseline"))
	run.Note("baseline exposes seeded shards without transfer execution")
	return run
}

func scenarioCrossShard() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := NewScenarioRun("cross-shard", ledger, DefaultScenarioPolicy("cross-shard"))
	ledger.Advance(1)
	transfer := submitOrPanic(ledger, TransferIntent{
		SourceShard: "shard-atlas",
		DestShard:   "shard-boreal",
		FromAccount: "atlas-market",
		ToAccount:   "boreal-merchant",
		Asset:       "usdq",
		Amount:      300_000,
		Arrival:     3,
		Tags:        Tags("scenario", "cross-shard"),
	})
	run.AddSubmitted(transfer)
	result := vulnerableReconciler(ledger).Reconcile("cross-shard-netting", nil)
	run.AddReconciliation(result)
	run.Note("honest cross-shard transfer finalizes because the origin account is solvent")
	return run
}

func scenarioShardClose() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := NewScenarioRun("shard-close", ledger, DefaultScenarioPolicy("shard-close"))
	ledger.Advance(1)
	transfer := submitOrPanic(ledger, TransferIntent{
		SourceShard: "shard-delta",
		DestShard:   "shard-ember",
		FromAccount: "delta-treasury",
		ToAccount:   "ember-escrow",
		Asset:       "usdq",
		Amount:      120_000,
		Arrival:     2,
		Tags:        Tags("scenario", "shard-close"),
	})
	run.AddSubmitted(transfer)
	result := vulnerableReconciler(ledger).Reconcile("pre-close-netting", nil)
	run.AddReconciliation(result)
	closure := NewClosureEngine(ledger)
	ledger.Advance(1)
	if err := closure.Begin("shard-ember", "operator requested drain after reconciled transfer"); err != nil {
		panic(err)
	}
	ledger.Advance(1)
	if _, err := closure.Certify("shard-ember", "pending activity snapshotted for final certificate"); err != nil {
		panic(err)
	}
	_, err := ledger.SubmitTransfer(TransferIntent{
		SourceShard: "shard-ember",
		DestShard:   "shard-atlas",
		FromAccount: "ember-escrow",
		ToAccount:   "atlas-alice",
		Asset:       "usdq",
		Amount:      10_000,
		Arrival:     5,
	})
	if err != nil {
		run.Note("closed shard rejected new outbound transfer")
	}
	return run
}

func scenarioReconciliation() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := NewScenarioRun("reconciliation", ledger, DefaultScenarioPolicy("reconciliation"))
	ledger.Advance(1)
	t1 := submitOrPanic(ledger, TransferIntent{
		SourceShard: "shard-atlas",
		DestShard:   "shard-boreal",
		FromAccount: "atlas-market",
		ToAccount:   "boreal-bob",
		Asset:       "usdq",
		Amount:      100_000,
		Arrival:     8,
		Tags:        Tags("batch", "priority"),
	})
	t2 := submitOrPanic(ledger, TransferIntent{
		SourceShard: "shard-delta",
		DestShard:   "shard-boreal",
		FromAccount: "delta-treasury",
		ToAccount:   "boreal-bob",
		Asset:       "usdq",
		Amount:      80_000,
		Arrival:     2,
		Tags:        Tags("batch", "priority"),
	})
	t3 := submitOrPanic(ledger, TransferIntent{
		SourceShard: "shard-atlas",
		DestShard:   "shard-cygnus",
		FromAccount: "atlas-market",
		ToAccount:   "cygnus-bridge",
		Asset:       "usdq",
		Amount:      125_000,
		Arrival:     5,
		Tags:        Tags("batch", "priority"),
	})
	run.AddSubmitted(t1)
	run.AddSubmitted(t2)
	run.AddSubmitted(t3)
	result := vulnerableReconciler(ledger).Reconcile("grouped-priority", nil)
	run.AddReconciliation(result)
	run.Note("credits are grouped by destination before debits are applied by arrival")
	return run
}

func scenarioWithdrawal() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := NewScenarioRun("withdrawal", ledger, DefaultScenarioPolicy("withdrawal"))
	ledger.Advance(1)
	transfer := submitOrPanic(ledger, TransferIntent{
		SourceShard: "shard-atlas",
		DestShard:   "shard-boreal",
		FromAccount: "atlas-market",
		ToAccount:   "boreal-merchant",
		Asset:       "usdq",
		Amount:      220_000,
		Arrival:     4,
		Tags:        Tags("scenario", "withdrawal"),
	})
	run.AddSubmitted(transfer)
	result := vulnerableReconciler(ledger).Reconcile("settle-before-withdrawal", nil)
	run.AddReconciliation(result)
	ledger.Advance(1)
	engine := NewWithdrawalEngine(ledger)
	if _, err := engine.RequestAndFinalize(WithdrawalIntent{
		ShardID:        "shard-boreal",
		AccountID:      "boreal-merchant",
		Asset:          "usdq",
		Amount:         220_000,
		SourceTransfer: transfer.ID,
		Tags:           Tags("scenario", "withdrawal"),
	}, false); err != nil {
		panic(err)
	}
	run.Note("withdrawal after final debit is backed by completed origin settlement")
	return run
}

func scenarioExploitWindow() *ScenarioRun {
	ledger := BuildSeedLedger()
	run := NewScenarioRun("exploit-window", ledger, ExploitScenarioPolicy())
	ledger.Advance(1)
	transfer := submitOrPanic(ledger, TransferIntent{
		SourceShard: "shard-atlas",
		DestShard:   "shard-cygnus",
		FromAccount: "atlas-attacker-origin",
		ToAccount:   "cygnus-attacker-dest",
		Asset:       "usdq",
		Amount:      700_000,
		Arrival:     1,
		Tags:        Tags("scenario", "exploit", "attacker", "true"),
	})
	run.AddSubmitted(transfer)
	reconciler := vulnerableReconciler(ledger)
	result := reconciler.Start("exploit-credit-checkpoint")
	reconciler.ApplyCredits(result)
	reconciler.ProcessCheckpointWithdrawals(result, []WithdrawalIntent{
		{
			ShardID:        "shard-cygnus",
			AccountID:      "cygnus-attacker-dest",
			Asset:          "usdq",
			Amount:         700_000,
			SourceTransfer: transfer.ID,
			Tags:           Tags("scenario", "exploit", "phase", "checkpoint"),
		},
	})
	reconciler.ApplyDebits(result)
	reconciler.Close(result)
	run.AddReconciliation(*result)
	run.Note("attacker withdrew grouped destination credit before the origin debit failed")
	return run
}
