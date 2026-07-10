package main

type ReconcilePolicy struct {
	Name                            string `json:"name"`
	GroupCreditsByDestination       bool   `json:"group_credits_by_destination"`
	ValidateCreditBeforeDebit       bool   `json:"validate_credit_before_debit"`
	PublishCreditCheckpoint         bool   `json:"publish_credit_checkpoint"`
	AllowWithdrawalDuringCheckpoint bool   `json:"allow_withdrawal_during_checkpoint"`
	AtomicCreditReversal            bool   `json:"atomic_credit_reversal"`
	DebitOrder                      string `json:"debit_order"`
}

func VulnerablePolicy() ReconcilePolicy {
	return ReconcilePolicy{
		Name:                            "grouped-credit-checkpoint",
		GroupCreditsByDestination:       true,
		ValidateCreditBeforeDebit:       false,
		PublishCreditCheckpoint:         true,
		AllowWithdrawalDuringCheckpoint: true,
		AtomicCreditReversal:            false,
		DebitOrder:                      "arrival",
	}
}

func StrictPolicy() ReconcilePolicy {
	return ReconcilePolicy{
		Name:                            "debit-first-atomic",
		GroupCreditsByDestination:       false,
		ValidateCreditBeforeDebit:       true,
		PublishCreditCheckpoint:         false,
		AllowWithdrawalDuringCheckpoint: false,
		AtomicCreditReversal:            true,
		DebitOrder:                      "arrival",
	}
}

type ScenarioPolicy struct {
	Name      string          `json:"name"`
	Policy    ReconcilePolicy `json:"policy"`
	Objective string          `json:"objective"`
}

func DefaultScenarioPolicy(name string) ScenarioPolicy {
	return ScenarioPolicy{
		Name:      name,
		Policy:    VulnerablePolicy(),
		Objective: "simulate production reconciler ordering",
	}
}

func ExploitScenarioPolicy() ScenarioPolicy {
	return ScenarioPolicy{
		Name:      "exploit-window",
		Policy:    VulnerablePolicy(),
		Objective: "withdraw grouped destination credit before origin debit finalization",
	}
}
