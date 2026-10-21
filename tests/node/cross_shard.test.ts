import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, bucket, byId, events, runScenario } from "../helpers/rift.ts";

test("honest cross-shard transfer finalizes credit and debit", () => {
  const report = runScenario("cross-shard");
  assertCommon(report, "cross-shard");
  assert.equal(report.transfers.length, 1);
  assert.equal(report.reconciliations.length, 1);

  const transfer = report.transfers[0];
  assert.equal(transfer.status, "finalized");
  assert.equal(transfer.credit_applied, true);
  assert.equal(transfer.debit_applied, true);
  assert.equal(transfer.amount, 300_000);

  const merchant = byId(report.accounts, "boreal-merchant");
  const market = byId(report.accounts, "atlas-market");
  assert.equal(bucket(merchant.available, "usdq"), 550_000);
  assert.equal(bucket(market.available, "usdq"), 5_700_000);
  assert.equal(bucket(report.risk.supply_delta, "usdq"), 0);
  assert.equal(report.invariants.credits_balanced, true);
  assert.equal(report.invariants.withdrawals_reconciled, true);
  assert.ok(events(report, "credit.applied").length >= 1);
  assert.ok(events(report, "debit.applied").length >= 1);
});
