package main

import "fmt"

type ScenarioRun struct {
	Name            string            `json:"name"`
	Ledger          *Ledger           `json:"-"`
	Policy          ScenarioPolicy    `json:"policy"`
	Reconciliations []ReconcileResult `json:"reconciliations"`
	Submitted       []string          `json:"submitted"`
	Notes           []string          `json:"notes"`
}

func AvailableScenarios() []string {
	return []string{
		"baseline",
		"cross-shard",
		"shard-close",
		"reconciliation",
		"withdrawal",
		"exploit-window",
	}
}

func RunScenario(name string) (*ScenarioRun, error) {
	switch name {
	case "baseline":
		return scenarioBaseline(), nil
	case "cross-shard":
		return scenarioCrossShard(), nil
	case "shard-close":
		return scenarioShardClose(), nil
	case "reconciliation":
		return scenarioReconciliation(), nil
	case "withdrawal":
		return scenarioWithdrawal(), nil
	case "exploit-window":
		return scenarioExploitWindow(), nil
	default:
		return nil, fmt.Errorf("unknown scenario %q", name)
	}
}

func NewScenarioRun(name string, ledger *Ledger, policy ScenarioPolicy) *ScenarioRun {
	return &ScenarioRun{
		Name:            name,
		Ledger:          ledger,
		Policy:          policy,
		Reconciliations: []ReconcileResult{},
		Submitted:       []string{},
		Notes:           []string{},
	}
}

func (s *ScenarioRun) AddReconciliation(result ReconcileResult) {
	s.Reconciliations = append(s.Reconciliations, result)
}

func (s *ScenarioRun) AddSubmitted(transfer *Transfer) {
	if transfer == nil {
		return
	}
	s.Submitted = append(s.Submitted, transfer.ID)
}

func (s *ScenarioRun) Note(value string) {
	if value == "" {
		return
	}
	s.Notes = append(s.Notes, value)
	s.Ledger.Note(value)
}

func BuildSeedLedger() *Ledger {
	ledger := NewLedger()
	seedAssets(ledger)
	seedShards(ledger)
	seedAccounts(ledger)
	ledger.CaptureOpeningSupply()
	ledger.Note("seeded five economic shards with deterministic USDQ, EURQ and RIFT balances")
	return ledger
}

func seedAssets(ledger *Ledger) {
	usdq := NewAsset("usdq", "USDQ", 6, AssetStable)
	usdq.Name = "Rift USD Quote"
	usdq.SettlementBps = 0
	usdq.WithdrawalBps = 0
	usdq.Tags.Set("rail", "primary")
	eurq := NewAsset("eurq", "EURQ", 6, AssetStable)
	eurq.Name = "Rift EUR Quote"
	eurq.SettlementBps = 0
	eurq.WithdrawalBps = 0
	eurq.Tags.Set("rail", "secondary")
	rift := NewAsset("rift", "RIFT", 8, AssetUtility)
	rift.Name = "Rift Coordinator Token"
	rift.SettlementBps = 2
	rift.WithdrawalBps = 5
	rift.RiskWeightBps = 8_500
	rift.Tags.Set("rail", "governance")
	ledger.AddAsset(usdq)
	ledger.AddAsset(eurq)
	ledger.AddAsset(rift)
}

func seedShards(ledger *Ledger) {
	shards := []*Shard{
		NewShard("shard-atlas", "Atlas", "us-east", 10),
		NewShard("shard-boreal", "Boreal", "eu-west", 8),
		NewShard("shard-cygnus", "Cygnus", "ap-south", 7),
		NewShard("shard-delta", "Delta", "us-west", 6),
		NewShard("shard-ember", "Ember", "eu-central", 4),
	}
	for _, shard := range shards {
		shard.Tags.Set("domain", "economic")
		ledger.AddShard(shard)
	}
}

func seedAccounts(ledger *Ledger) {
	addSeedAccount(ledger, "shard-atlas", "atlas-market", "Atlas Market Maker", "market-maker", map[string]Amount{
		"usdq": 6_000_000,
		"eurq": 500_000,
	})
	addSeedAccount(ledger, "shard-atlas", "atlas-alice", "Alice", "retail", map[string]Amount{
		"usdq": 1_000_000,
	})
	addSeedAccount(ledger, "shard-atlas", "atlas-attacker-origin", "Attacker Origin", "retail", map[string]Amount{
		"usdq": 40_000,
	})
	addSeedAccount(ledger, "shard-boreal", "boreal-merchant", "Boreal Merchant", "merchant", map[string]Amount{
		"usdq": 250_000,
		"eurq": 750_000,
	})
	addSeedAccount(ledger, "shard-boreal", "boreal-bob", "Bob", "retail", map[string]Amount{
		"usdq": 50_000,
	})
	addSeedAccount(ledger, "shard-cygnus", "cygnus-attacker-dest", "Attacker Destination", "retail", map[string]Amount{
		"usdq": 0,
	})
	addSeedAccount(ledger, "shard-cygnus", "cygnus-bridge", "Cygnus Bridge", "bridge", map[string]Amount{
		"usdq": 900_000,
		"rift": 50_000_000,
	})
	addSeedAccount(ledger, "shard-delta", "delta-treasury", "Delta Treasury", "treasury", map[string]Amount{
		"usdq": 3_000_000,
		"eurq": 2_000_000,
	})
	addSeedAccount(ledger, "shard-delta", "delta-ops", "Delta Ops", "operator", map[string]Amount{
		"usdq": 800_000,
	})
	addSeedAccount(ledger, "shard-ember", "ember-escrow", "Ember Escrow", "escrow", map[string]Amount{
		"usdq": 500_000,
	})
}

func addSeedAccount(ledger *Ledger, shardID string, id string, owner string, role string, balances map[string]Amount) {
	account := NewAccount(id, owner, role, shardID)
	account.Tags.Set("seed", "true")
	for asset, amount := range balances {
		if amount > 0 {
			if err := account.Credit(asset, amount); err != nil {
				panic(err)
			}
		} else {
			account.balance(asset)
		}
	}
	ledger.AddAccount(shardID, account)
}

func submitOrPanic(ledger *Ledger, intent TransferIntent) *Transfer {
	transfer, err := ledger.SubmitTransfer(intent)
	if err != nil {
		panic(err)
	}
	return transfer
}

func vulnerableReconciler(ledger *Ledger) *Reconciler {
	return NewReconciler(ledger, VulnerablePolicy())
}
