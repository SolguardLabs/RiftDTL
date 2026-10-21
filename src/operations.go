package main

// StatusReport is a read-only operational projection of a completed scenario.
type StatusReport struct {
	Protocol      string             `json:"protocol"`
	Scenario      string             `json:"scenario"`
	Clock         int64              `json:"clock"`
	StateDigest   string             `json:"state_digest"`
	Rounds        int                `json:"rounds"`
	TransferCount int                `json:"transfer_count"`
	Transfers     TransferCounters   `json:"transfers"`
	Shards        []ShardLiquidity   `json:"shards"`
	Assets        []NetworkLiquidity `json:"assets"`
}

type TransferCounters struct {
	Queued    int `json:"queued"`
	Finalized int `json:"finalized"`
	Failed    int `json:"failed"`
	Rejected  int `json:"rejected"`
}

type ShardLiquidity struct {
	ShardID          string         `json:"shard_id"`
	Region           string         `json:"region"`
	Status           ShardStatus    `json:"status"`
	AccountCount     int            `json:"account_count"`
	Available        []AmountBucket `json:"available"`
	Held             []AmountBucket `json:"held"`
	PendingCredit    []AmountBucket `json:"pending_credit"`
	PendingDebit     []AmountBucket `json:"pending_debit"`
	SettlementBuffer []AmountBucket `json:"settlement_buffer"`
}

type NetworkLiquidity struct {
	Asset        string `json:"asset"`
	Available    Amount `json:"available"`
	Held         Amount `json:"held"`
	PendingIn    Amount `json:"pending_in"`
	PendingOut   Amount `json:"pending_out"`
	Externalized Amount `json:"externalized"`
}

func BuildStatusReport(run *ScenarioRun) StatusReport {
	report := BuildReport(run)
	status := StatusReport{
		Protocol:      report.Protocol,
		Scenario:      report.Scenario,
		Clock:         report.Clock,
		StateDigest:   report.StateDigest,
		Rounds:        len(report.Reconciliations),
		TransferCount: len(report.Transfers),
		Shards:        make([]ShardLiquidity, 0, len(report.Shards)),
		Assets:        make([]NetworkLiquidity, 0, len(report.Assets)),
	}
	for _, transfer := range report.Transfers {
		switch transfer.Status {
		case TransferFinalized:
			status.Transfers.Finalized++
		case TransferDebitFailed:
			status.Transfers.Failed++
		case TransferRejected:
			status.Transfers.Rejected++
		default:
			status.Transfers.Queued++
		}
	}
	for _, shard := range report.Shards {
		status.Shards = append(status.Shards, ShardLiquidity{
			ShardID:          shard.ID,
			Region:           shard.Region,
			Status:           shard.Status,
			AccountCount:     shard.AccountCount,
			Available:        shard.Available,
			Held:             shard.Held,
			PendingCredit:    shard.PendingCredit,
			PendingDebit:     shard.PendingDebit,
			SettlementBuffer: shard.SettlementBuffer,
		})
	}
	for _, asset := range report.Assets {
		status.Assets = append(status.Assets, NetworkLiquidity{
			Asset:        asset.ID,
			Available:    bucketAmount(report.Totals.Available, asset.ID),
			Held:         bucketAmount(report.Totals.Held, asset.ID),
			PendingIn:    bucketAmount(report.Totals.PendingIn, asset.ID),
			PendingOut:   bucketAmount(report.Totals.PendingOut, asset.ID),
			Externalized: bucketAmount(report.Totals.Externalized, asset.ID),
		})
	}
	return status
}

func bucketAmount(values []AmountBucket, asset string) Amount {
	for _, value := range values {
		if value.Asset == asset {
			return value.Amount
		}
	}
	return 0
}
