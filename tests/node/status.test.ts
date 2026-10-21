import test from "node:test";
import assert from "node:assert/strict";
import { runBinary } from "../helpers/rift.ts";

test("status command reconciles shard and network balances", () => {
  const output = runBinary(["status", "cross-shard"]);
  assert.equal(output.status, 0);
  const report = JSON.parse(output.stdout);
  assert.equal(report.protocol, "RiftDTL");
  assert.equal(report.scenario, "cross-shard");
  assert.equal(report.shards.length, 5);
  assert.equal(report.assets.length, 3);
  assert.equal(report.transfer_count, 1);
  assert.equal(report.transfers.finalized, 1);
  for (const asset of report.assets) {
    const shardAvailable = report.shards.reduce(
      (sum: number, shard: any) =>
        sum + (shard.available.find((bucket: any) => bucket.asset === asset.asset)?.amount ?? 0),
      0,
    );
    assert.equal(asset.available, shardAvailable);
  }
});

test("status command rejects unknown scenarios", () => {
  const output = runBinary(["status", "unknown"], false);
  assert.notEqual(output.status, 0);
  assert.match(output.stderr, /unknown scenario/);
});
