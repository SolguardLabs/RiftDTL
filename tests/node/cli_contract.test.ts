import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, assertDigest, listScenarios, runBinary, runScenario, validateScenario } from "../helpers/rift.ts";

test("CLI lists stable RiftDTL scenarios", () => {
  assert.deepEqual(listScenarios(), [
    "baseline",
    "cross-shard",
    "shard-close",
    "reconciliation",
    "withdrawal",
    "exploit-window",
  ]);
});

test("baseline scenario emits the public JSON contract", () => {
  const report = runScenario("baseline");
  assertCommon(report, "baseline");
  assert.equal(report.transfers.length, 0);
  assert.equal(report.withdrawals.length, 0);
  assert.equal(report.invariants.no_unbacked_credit, true);
  assert.equal(report.invariants.no_supply_expansion, true);
  assertDigest(report.state_digest);
});

test("validate command returns scenario digest", () => {
  const output = validateScenario("baseline");
  assert.match(output, /^ok baseline [0-9a-f]{32}$/);
});

test("unknown scenarios fail without JSON output", () => {
  const result = runBinary(["scenario", "missing-scenario"], false);
  assert.notEqual(result.status, 0);
  assert.match(result.stderr, /unknown scenario/);
  assert.equal(result.stdout.trim(), "");
});
