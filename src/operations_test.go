package main

import "testing"

func TestStatusReconcilesNetworkAndShardBalances(t *testing.T) {
	run, err := RunScenario("cross-shard")
	if err != nil {
		t.Fatal(err)
	}
	status := BuildStatusReport(run)
	if status.Protocol != "RiftDTL" || status.Rounds != 1 || status.TransferCount != 1 {
		t.Fatalf("unexpected status header: %+v", status)
	}
	if status.Transfers.Finalized != 1 || status.Transfers.Failed != 0 {
		t.Fatalf("unexpected transfer counters: %+v", status.Transfers)
	}
	for _, asset := range status.Assets {
		available := Amount(0)
		for _, shard := range status.Shards {
			available += bucketAmount(shard.Available, asset.Asset)
		}
		if available != asset.Available {
			t.Fatalf("asset %s: shards=%d network=%d", asset.Asset, available, asset.Available)
		}
	}
}

func TestStatusHandlesEmptyTransferSet(t *testing.T) {
	run, err := RunScenario("baseline")
	if err != nil {
		t.Fatal(err)
	}
	status := BuildStatusReport(run)
	if status.TransferCount != 0 || status.Transfers.Finalized != 0 || status.Rounds != 0 {
		t.Fatalf("unexpected baseline counters: %+v", status)
	}
	if len(status.Shards) != 5 || len(status.Assets) != 3 || status.StateDigest == "" {
		t.Fatalf("incomplete status: %+v", status)
	}
}
