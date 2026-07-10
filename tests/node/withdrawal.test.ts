import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, bucket, events, runScenario } from "../helpers/rift.ts";

test("withdrawal after debit finalization is backed", () => {
  const report = runScenario("withdrawal");
  assertCommon(report, "withdrawal");
  assert.equal(report.transfers[0].status, "finalized");
  assert.equal(report.withdrawals.length, 1);

  const withdrawal = report.withdrawals[0];
  assert.equal(withdrawal.status, "finalized");
  assert.equal(withdrawal.amount, 220_000);
  assert.equal(withdrawal.before_debit_final, false);
  assert.equal(withdrawal.source_transfer, report.transfers[0].id);
  assert.equal(bucket(report.risk.externalized, "usdq"), 220_000);
  assert.equal(bucket(report.risk.withdrawn_before_debit, "usdq"), 0);
  assert.equal(bucket(report.risk.withdrawn_unbacked, "usdq"), 0);
  assert.equal(report.invariants.withdrawals_backed_by_final_debit, true);
  assert.ok(events(report, "withdrawal.finalized").length, "missing withdrawal event");
});
