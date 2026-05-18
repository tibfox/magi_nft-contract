# Editioned NFT — Define-Without-Mint + Delegated (Market) Mint

Date: 2026-05-18
Repo: `magi_nft-contract` (based on latest upstream `vsc-eco/magi_nft-contract` main, `88335da`)
Branch: `feat/editioned-define-delegated-mint`

## Problem

A creator wants to:

1. Set up an editioned NFT (fixed `maxSupply`, `properties`, `soulbound`) **without minting any tokens**.
2. Let authorized parties mint that edition afterward, up to `maxSupply`.
3. Have a **market contract** execute the mint on a buyer's behalf: `msg.caller` is the
   market, but the recipient is the buyer and the market is not the contract owner.

## What upstream already provides

Latest upstream main already implements editioned NFTs and approvals; this feature is a
small delta on top, not a rewrite:

- `Mint` / `MintBatch` / `MintSeries` with `maxSupply` (1 = unique, >1 = editioned).
- Supply accounting: `totalSupply`, `totalMinted`, `maxSupply`, optional `trackMinted`.
- ERC-6909 approvals: per-token `Approve`/`Allowance`, operator `SetApprovalForAll`,
  and the helper `isApprovedOrOwner(caller, from)` (`contract/internal.go:410`), which
  returns true when `caller == from` or `from` approved `caller` for all.
- `Exists` returns `getMaxSupply(id) > 0`.
- `tokenCreated` event (`contract/events.go:181`) emitted on first mint of a token.

### Gaps

- `maxSupply` is only ever set on the **first mint** — no way to define an edition with
  zero supply.
- `Mint` / `MintSeries` are **owner-only** (`if !isOwner { Abort("Must be owner to mint") }`);
  the ERC-6909 operator approval governs transfers/burns, not minting.

## Design

Two surgical changes to `contract/token.go`, no new exports, events, state keys, or
approval mechanism.

### Decisions (confirmed with user)

| Question | Decision |
|---|---|
| Define-without-mint API | Generalize existing `Mint`/`MintSeries` with `amount == 0`; no new exports |
| Who may mint a defined edition | Owner; an operator approved via `SetApprovalForAll` (uncapped, up to `maxSupply`); **or** a spender the owner gave a per-token `approve` allowance (capped, decremented per mint) |
| Mint cap for approved markets | Operator approval = uncapped (up to `maxSupply`). Per-token `approve` allowance = capped at the granted amount, decremented per mint (ERC-6909). *(Revised 2026-05-18: per-token allowance now also authorizes minting.)* |
| Who may *define* an edition | **Owner only** (operators may mint, not define) |
| Branch base | Local `main` hard-reset to `upstream/main`; feature branched from it |

### Component 1 — Delegated mint authorization

**Files:** `contract/token.go` — `Mint`, `MintSeries`.

Replace the owner-only gate:

```go
owner, isOwner := getOwner()
if !isOwner { sdk.Abort("Must be owner to mint") }
```

with an owner-or-approved-operator gate:

```go
ownerAddr := getOwnerAddress()
caller := *sdk.GetEnvKey("msg.caller")
if !isApprovedOrOwner(caller, ownerAddr) {
    sdk.Abort("Must be owner or approved operator to mint")
}
```

- `getOwnerAddress()` (`contract/main.go:52`) returns the owner without a redundant
  `msg.caller` read; `caller` is read once and reused.
- The owner authorizes a market either with `setApprovalForAll(market, true)`
  (uncapped) or with a per-token `approve(spender=market, id, amount=N)` allowance
  (capped at N). The mint authorization mirrors `safeTransferFrom`: if the caller is
  not owner and not an approved operator, fall back to the per-token allowance
  `getAllowance(ownerAddr, operator, id)`; require `allowance >= amount` and
  `setAllowance(... allowance-amount)` (decrement per mint). For `mintSeries` the
  allowance is checked and decremented per generated id (mirrors
  `safeBatchTransferFrom`). Define-only (`amount == 0`) remains owner-only and never
  consults allowance.
- The mint cap is unchanged: the existing "subsequent mint" path enforces `maxSupply`
  (via `totalMinted` when `trackMinted` is on, else `totalSupply`).
- **Provenance fix:** the mint `TransferSingle` / `TransferBatch` `operator` field becomes
  the actual `caller` (the market) instead of `owner`. `from` stays the zero address.

### Component 2 — Define edition without minting (`amount == 0`)

**Files:** `contract/token.go` — `Mint`, `MintSeries`; comment fix in `Exists`.

Remove the blanket `if p.Amount == 0 { sdk.Abort("Amount must be greater than 0") }`
and branch on it instead.

**Define-only mode — `p.Amount == 0`:**

- **Owner-only**: require `caller == ownerAddr`, else abort
  `"Only owner can define an edition"`. (Approved operators may mint but not define.)
- Require `existingMax == 0` (edition not yet defined), else abort
  `"Edition already defined"`.
- Require `p.MaxSupply != 0`, else abort
  `"MaxSupply required (1 = unique, >1 = editioned)"`.
- `p.To` is optional and ignored in this mode (no recipient when nothing is minted);
  the `"To address required"` check is skipped when `p.Amount == 0`.
- Effects: `setMaxSupply(id, maxSupply)`; if `p.Soulbound` → `setSoulbound(id)`;
  if `p.Properties != ""` → `setTokenProperties(id, p.Properties)` + `emitPropertiesSet(id)`.
- Emit `emitTokenCreated(id, maxSupply, p.Soulbound)` as the "edition exists" signal.
- Do **not** modify balances or `totalSupply`; emit **no** `Transfer` event.
- Return `SuccessResponse{Success: true}`.

**Mint mode — `p.Amount > 0`:** unchanged logic. Because `maxSupply` was pre-set by a
prior define, `existingMax != 0`, so the existing "subsequent mint" path runs: it
enforces the `maxSupply` cap and does **not** re-emit `tokenCreated` (it is guarded by
`existingMax == 0`). A first `Mint` with `amount > 0` on an undefined token still
defines-and-mints exactly as today.

**`MintSeries` define-only:** same `amount == 0` treatment applied per generated id —
defines every id in the `idPrefix + (startNumber+i) + idSuffix` range with `maxSupply` /
`soulbound` / `properties`, leaves `totalSupply` at 0, emits `tokenCreated` per id, and
emits no `TransferBatch`. The `propertiesTemplate` handling is preserved.

**`Exists`:** behavior already correct (`getMaxSupply(id) > 0`); update the stale
comment `// ... (meaning it was minted at least once)` to reflect that a defined-but-
unminted edition also exists.

### Out of scope (YAGNI)

- `MintBatch` keeps requiring `amount > 0`; mixed per-element define/mint is not needed.
- No new exports, events, state keys, or approval mechanism (per-token allowance reuses
  the existing `approve`/`allowance` ERC-6909 functions).
- No payment/pricing logic — the market handles selling; the contract only authorizes the mint.

## Edge cases

- Define then mint exactly `maxSupply` succeeds; one more aborts `"Would exceed max supply"`.
- Define with `maxSupply == 0` aborts.
- Re-defining an already-defined or already-minted edition aborts `"Edition already defined"`.
- Approved operator calling define (`amount == 0`) aborts `"Only owner can define an edition"`.
- Unapproved caller minting aborts; after `setApprovalForAll(market, false)` the market's
  mint aborts.
- `soulbound` / `properties` set at define time are observed by tokens minted later.
- `MintSeries` define-only with overlapping/duplicate ids in the generated range follows
  the existing per-id `existingMax` guard (second occurrence aborts `"Edition already defined"`).

## Testing (TDD)

New tests follow existing patterns (`test/mintseries_test.go`, `test/approval_test.go`,
`test/mint_test.go`):

1. **Define single edition** (`amount=0`): `totalSupply==0`, `maxSupply` set,
   `Exists==true`, `IsSoulbound`/`getProperties` reflect define-time values.
2. **Mint up to cap**: minting `maxSupply` total succeeds across calls; the next aborts.
3. **Delegated mint**: owner `setApprovalForAll(market,true)`; market mints to a buyer
   (recipient balance increments, market balance unchanged); unapproved caller aborts;
   after revoke, market mint aborts.
4. **Owner-only define**: approved operator calling define (`amount=0`) aborts.
5. **Redefine guard**: defining an existing edition aborts.
6. **MintSeries define-only**: whole generated range defined with 0 supply, each
   `Exists==true`, then mintable up to `maxSupply`.
7. **Transfer event provenance**: delegated mint's `TransferSingle.operator == market`.

All existing tests must remain green (the owner path is a strict subset of the new gate;
`amount > 0` behavior is unchanged).

## Risk notes

- `types_tinyjson.go` is hand-maintained and never regenerated — payload structs
  (`MintPayload`, `MintSeriesPayload`) are unchanged by this feature, so no tinyjson edits
  are required. If any field is added later, tinyjson must be hand-edited to match.
- The auth-gate change widens who can call `Mint`/`MintSeries`; the supply cap and
  owner-only define keep the blast radius bounded. Confirm no other code path assumed
  `Mint` implies caller-is-owner.
