import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, bucket, runScenario } from "../helpers/rift.ts";

test("reconciler groups destination credits before applying debits by arrival", () => {
  const report = runScenario("reconciliation");
  assertCommon(report, "reconciliation");
  assert.equal(report.reconciliations.length, 1);

  const result = report.reconciliations[0];
  assert.equal(result.published_checkpoint, true);
  assert.equal(result.credit_before_debit, true);
  assert.equal(result.credit_groups.length, 2);
  assert.equal(result.debits.map((debit) => debit.arrival).join(","), "2,5,8");

  const borealGroup = result.credit_groups.find((group) => group.key === "shard-boreal:boreal-bob:usdq");
  assert.ok(borealGroup, "missing grouped Boreal credit");
  assert.equal(borealGroup.amount, 180_000);
  assert.equal(borealGroup.transfers.length, 2);

  const firstCredit = result.trace.find((entry) => entry.phase === "credit" && entry.action === "apply-group");
  const firstDebit = result.trace.find((entry) => entry.phase === "debit");
  assert.ok(firstCredit && firstDebit, "missing credit/debit trace");
  assert.ok(firstCredit.step < firstDebit.step, "credit phase should be visible before debit phase");
  assert.equal(bucket(report.risk.credits_applied, "usdq"), 305_000);
  assert.equal(bucket(report.risk.debits_applied, "usdq"), 305_000);
  assert.equal(report.invariants.grouped_credit_checkpoint, true);
});
