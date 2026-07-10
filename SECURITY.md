# Security Policy

This repository is intentionally vulnerable. Do not deploy the reconciler logic
as production settlement code.

The challenge is scoped to the ordering of reconciliation phases:

- destination credits are grouped by shard and account,
- the grouped credit checkpoint is visible to withdrawals,
- origin debits are only validated afterward in arrival order,
- failed debits do not atomically revert previously published credits.

The remediation is to make debit reservation and origin solvency validation the
commit barrier for every cross-shard credit, or to keep credits non-withdrawable
until the matching debit is final.
