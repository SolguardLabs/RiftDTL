import test from "node:test";
import assert from "node:assert/strict";
import { assertCommon, bucket, byId, events, runScenario } from "../helpers/rift.ts";

test("shard closure certifies reconciled local state and rejects new outbound transfer", () => {
  const report = runScenario("shard-close");
  assertCommon(report, "shard-close");
  assert.equal(report.closure_certificates.length, 1);

  const ember = byId(report.shards, "shard-ember");
  assert.equal(ember.status, "closed");
  assert.ok(ember.certificate?.startsWith("close-shard-ember-"));
  assert.equal(bucket(ember.available, "usdq"), 620_000);
  assert.equal(report.transfers[0].status, "finalized");
  assert.match(report.notes.join("\n"), /closed shard rejected new outbound transfer/);
  assert.ok(events(report, "shard.closed").length, "missing shard close event");
});
