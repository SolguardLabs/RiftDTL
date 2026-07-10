import assert from "node:assert/strict";
import { spawnSync } from "node:child_process";
import { existsSync } from "node:fs";
import { join, resolve } from "node:path";
import { fileURLToPath } from "node:url";

export const root = resolve(fileURLToPath(new URL("../..", import.meta.url)));
const exeName = process.platform === "win32" ? "riftdtl.exe" : "riftdtl";
const defaultBin = join(root, "out", exeName);

export type AmountBucket = {
  asset: string;
  amount: number;
};

export type BalanceReport = {
  asset: string;
  available: number;
  held: number;
  pending_in: number;
  pending_out: number;
  withdrawn: number;
  externalized: number;
};

export type AccountReport = {
  id: string;
  owner: string;
  role: string;
  shard_id: string;
  frozen: boolean;
  sequence: number;
  balances: BalanceReport[];
  available: AmountBucket[];
  withdrawn: AmountBucket[];
  externalized: AmountBucket[];
};

export type ShardReport = {
  id: string;
  name: string;
  region: string;
  status: string;
  priority: number;
  account_count: number;
  available: AmountBucket[];
  pending_credit: AmountBucket[];
  pending_debit: AmountBucket[];
  withdrawn: AmountBucket[];
  externalized: AmountBucket[];
  certificate?: string;
};

export type TransferReport = {
  id: string;
  source_shard: string;
  dest_shard: string;
  from_account: string;
  to_account: string;
  asset: string;
  amount: number;
  fee: number;
  arrival: number;
  credit_applied: boolean;
  debit_applied: boolean;
  withdrawn: number;
  status: string;
  reason?: string;
  credit_group: string;
};

export type WithdrawalReport = {
  id: string;
  shard_id: string;
  account_id: string;
  asset: string;
  amount: number;
  status: string;
  window?: string;
  source_transfer?: string;
  before_debit_final: boolean;
};

export type CreditGroupReport = {
  key: string;
  shard_id: string;
  account_id: string;
  asset: string;
  amount: number;
  transfers: string[];
};

export type DebitReport = {
  transfer: string;
  shard_id: string;
  account: string;
  asset: string;
  amount: number;
  fee: number;
  arrival: number;
  applied: boolean;
  reason?: string;
};

export type TraceReport = {
  step: number;
  phase: string;
  action: string;
  subject: string;
  transfer?: string;
  group?: string;
  shard_id?: string;
  account?: string;
  asset?: string;
  amount?: number;
  detail?: string;
};

export type ReconcileReport = {
  name: string;
  round: number;
  window: string;
  credit_groups: CreditGroupReport[];
  debits: DebitReport[];
  withdrawals: string[];
  completed: string[];
  failed: string[];
  phantom_credits: AmountBucket[];
  published_checkpoint: boolean;
  credit_before_debit: boolean;
  trace: TraceReport[];
};

export type RiftReport = {
  lab: string;
  scenario: string;
  network_id: string;
  clock: number;
  state_digest: string;
  assets: Array<Record<string, unknown>>;
  shards: ShardReport[];
  accounts: AccountReport[];
  transfers: TransferReport[];
  withdrawals: WithdrawalReport[];
  closure_certificates: Array<Record<string, any>>;
  reconciliations: ReconcileReport[];
  totals: Record<string, AmountBucket[]>;
  risk: {
    opening_supply: AmountBucket[];
    current_supply: AmountBucket[];
    supply_delta: AmountBucket[];
    credits_applied: AmountBucket[];
    debits_applied: AmountBucket[];
    unbacked_credits: AmountBucket[];
    withdrawn_before_debit: AmountBucket[];
    withdrawn_unbacked: AmountBucket[];
    externalized: AmountBucket[];
    failed_debit_credits: number;
    checkpoint_withdrawals: number;
    credit_before_debit_rounds: number;
    phantom_liquidity: number;
    retirable_phantom: number;
    max_single_failed_debit: number;
  };
  invariants: {
    accounts_non_negative: boolean;
    shard_links_valid: boolean;
    transfer_links_valid: boolean;
    closure_links_valid: boolean;
    no_unbacked_credit: boolean;
    withdrawals_backed_by_final_debit: boolean;
    no_supply_expansion: boolean;
    grouped_credit_checkpoint: boolean;
    exploit_reachable: boolean;
  };
  events: Array<Record<string, any>>;
  notes: string[];
};

export function binaryPath(): string {
  return process.env.RIFT_BIN ?? defaultBin;
}

export function ensureBuilt(): void {
  if (process.env.RIFT_BIN) return;
  if (existsSync(defaultBin)) return;
  const result = spawnSync(process.execPath, ["scripts/build.mjs"], {
    cwd: root,
    encoding: "utf8",
    stdio: "pipe",
  });
  if (result.status !== 0) {
    throw new Error(["build failed", result.stdout.trim(), result.stderr.trim()].filter(Boolean).join("\n"));
  }
}

export function runBinary(args: string[], expectSuccess = true) {
  ensureBuilt();
  const result = spawnSync(binaryPath(), args, {
    cwd: root,
    encoding: "utf8",
    stdio: "pipe",
  });
  if (expectSuccess && result.status !== 0) {
    throw new Error(
      [`riftdtl ${args.join(" ")} failed`, result.stdout.trim(), result.stderr.trim()].filter(Boolean).join("\n"),
    );
  }
  return result;
}

export function listScenarios(): string[] {
  return runBinary(["--list"]).stdout.trim().split(/\r?\n/).filter(Boolean);
}

export function runScenario(name: string): RiftReport {
  const result = runBinary(["scenario", name]);
  return JSON.parse(result.stdout) as RiftReport;
}

export function validateScenario(name: string): string {
  return runBinary(["validate", name]).stdout.trim();
}

export function byId<T extends { id: string }>(items: T[], id: string): T {
  const found = items.find((item) => item.id === id);
  assert.ok(found, `missing id ${id}`);
  return found;
}

export function bucket(entries: AmountBucket[], asset: string): number {
  const found = entries.find((entry) => entry.asset === asset);
  assert.ok(found, `missing asset ${asset}`);
  return found.amount;
}

export function events(report: RiftReport, kind: string): Array<Record<string, any>> {
  return report.events.filter((event) => event.kind === kind);
}

export function assertDigest(value: unknown): void {
  assert.equal(typeof value, "string");
  assert.match(value as string, /^[0-9a-f]{32}$/);
}

export function assertCommon(report: RiftReport, scenario: string): void {
  assert.equal(report.lab, "RiftDTL");
  assert.equal(report.scenario, scenario);
  assert.equal(report.network_id, "rift-local-shardnet");
  assertDigest(report.state_digest);
  assert.equal(report.assets.length, 3);
  assert.equal(report.shards.length, 5);
  assert.ok(report.accounts.length >= 10);
  assert.equal(report.invariants.accounts_non_negative, true);
  assert.equal(report.invariants.shard_links_valid, true);
  assert.equal(report.invariants.transfer_links_valid, true);
  assert.equal(report.invariants.closure_links_valid, true);
  assert.ok(Array.isArray(report.reconciliations));
  assert.ok(Array.isArray(report.events));
}
