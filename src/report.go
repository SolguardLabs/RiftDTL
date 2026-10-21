package main

type TotalsReport struct {
	OpeningSupply  []AmountBucket `json:"opening_supply"`
	Available      []AmountBucket `json:"available"`
	Held           []AmountBucket `json:"held"`
	PendingIn      []AmountBucket `json:"pending_in"`
	PendingOut     []AmountBucket `json:"pending_out"`
	Withdrawn      []AmountBucket `json:"withdrawn"`
	Externalized   []AmountBucket `json:"externalized"`
	Credits        []AmountBucket `json:"credits"`
	Debits         []AmountBucket `json:"debits"`
	CreditVariance []AmountBucket `json:"credit_variance"`
}

type Report struct {
	Protocol            string               `json:"protocol"`
	Scenario            string               `json:"scenario"`
	NetworkID           string               `json:"network_id"`
	Clock               int64                `json:"clock"`
	StateDigest         string               `json:"state_digest"`
	Policy              ScenarioPolicy       `json:"policy"`
	Assets              []Asset              `json:"assets"`
	Shards              []ShardReport        `json:"shards"`
	Accounts            []AccountReport      `json:"accounts"`
	Transfers           []Transfer           `json:"transfers"`
	Withdrawals         []Withdrawal         `json:"withdrawals"`
	ClosureCertificates []ClosureCertificate `json:"closure_certificates"`
	Reconciliations     []ReconcileResult    `json:"reconciliations"`
	Totals              TotalsReport         `json:"totals"`
	Risk                RiskMetrics          `json:"risk"`
	Invariants          InvariantReport      `json:"invariants"`
	Events              []Event              `json:"events"`
	Notes               []string             `json:"notes"`
}

func BuildReport(run *ScenarioRun) Report {
	ledger := run.Ledger
	risk := NewRiskEngine(ledger, run.Reconciliations)
	metrics := risk.Metrics()
	invariants := risk.Invariants()
	notes := append([]string{}, ledger.Notes...)
	notes = append(notes, run.Notes...)
	return Report{
		Protocol:            ledger.Protocol,
		Scenario:            run.Name,
		NetworkID:           ledger.NetworkID,
		Clock:               ledger.Clock.Snapshot(),
		StateDigest:         ledger.Digest(),
		Policy:              run.Policy,
		Assets:              ledger.Assets.List(),
		Shards:              ledger.Shards.Reports(),
		Accounts:            ledger.Shards.Accounts(),
		Transfers:           ledger.Transfers.List(),
		Withdrawals:         ledger.Withdrawals.List(),
		ClosureCertificates: ledger.Closures.List(),
		Reconciliations:     run.Reconciliations,
		Totals:              buildTotals(ledger, metrics),
		Risk:                metrics,
		Invariants:          invariants,
		Events:              ledger.Journal.List(),
		Notes:               notes,
	}
}

func buildTotals(ledger *Ledger, metrics RiskMetrics) TotalsReport {
	available := map[string]Amount{}
	held := map[string]Amount{}
	pendingIn := map[string]Amount{}
	pendingOut := map[string]Amount{}
	withdrawn := map[string]Amount{}
	externalized := map[string]Amount{}
	for _, asset := range ledger.Assets.IDs() {
		available[asset] = ledger.Shards.TotalAvailable(asset)
		held[asset] = ledger.Shards.TotalHeld(asset)
		pendingIn[asset] = ledger.Shards.TotalPendingIn(asset)
		pendingOut[asset] = ledger.Shards.TotalPendingOut(asset)
		withdrawn[asset] = ledger.Shards.TotalWithdrawn(asset)
		externalized[asset] = ledger.Shards.TotalExternalized(asset)
	}
	return TotalsReport{
		OpeningSupply:  metrics.OpeningSupply,
		Available:      BucketsFromMap(available),
		Held:           BucketsFromMap(held),
		PendingIn:      BucketsFromMap(pendingIn),
		PendingOut:     BucketsFromMap(pendingOut),
		Withdrawn:      BucketsFromMap(withdrawn),
		Externalized:   BucketsFromMap(externalized),
		Credits:        metrics.CreditsApplied,
		Debits:         metrics.DebitsApplied,
		CreditVariance: metrics.CreditSettlementVariance,
	}
}

func ValidateReport(report Report) bool {
	return report.Protocol == "RiftDTL" &&
		report.NetworkID == "rift-local-shardnet" &&
		report.Scenario != "" &&
		report.StateDigest != "" &&
		report.Invariants.AccountsNonNegative &&
		report.Invariants.ShardLinksValid &&
		report.Invariants.TransferLinksValid &&
		report.Invariants.ClosureLinksValid
}
