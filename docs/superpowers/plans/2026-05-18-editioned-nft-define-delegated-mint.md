# Editioned NFT — Define-Without-Mint + Delegated Mint Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Let an owner define an editioned NFT with zero supply and let an owner-approved market mint it to buyers up to `maxSupply`.

**Architecture:** Two surgical edits to `contract/token.go`. (1) Replace the owner-only mint gate in `Mint`/`MintSeries` with the existing `isApprovedOrOwner(caller, ownerAddr)` check and stamp the real caller as the Transfer `operator`. (2) Treat `amount == 0` as owner-only "define edition" (set `maxSupply`/`soulbound`/`properties`, emit existing `tokenCreated`, no balance/supply/Transfer). No new exports, events, state keys, or approval mechanism.

**Tech Stack:** Go (TinyGo → `wasm-unknown`), `vsc-node` contract test harness, badger-backed state.

Spec: `docs/superpowers/specs/2026-05-18-editioned-nft-define-delegated-mint-design.md`
Branch: `feat/editioned-define-delegated-mint` (already created off latest upstream `main`).

---

## Build & Test Commands (used by every task)

Rebuild the wasm artifact after **every** contract edit — tests embed `test/artifacts/main.wasm`:

```bash
cd /home/dockeruser/magi/magi_nft-contract
tinygo build -gc=custom -scheduler=none -panic=trap -no-debug -target=wasm-unknown -o test/artifacts/main.wasm ./contract
```

Run a single test:

```bash
cd /home/dockeruser/magi/magi_nft-contract && go test ./test/... -run TestName -v
```

Full regression:

```bash
cd /home/dockeruser/magi/magi_nft-contract && go test ./test/...
```

Commit style: multi-line subject + bullet body, no Co-Authored-By.

---

## File Structure

- Modify: `contract/token.go` — `Mint` (~358-443), `MintSeries` (~603-752), `Exists` comment (~1252)
- Create: `test/editioned_define_test.go` — all new behavior tests for this feature
- Rebuild: `test/artifacts/main.wasm` (build output, committed because tests embed it)

No payload struct changes → `contract/types_tinyjson.go` is **not** touched (it is hand-maintained; leave it).

---

### Task 1: Delegated mint authorization for `Mint`

Change `Mint` so an operator approved by the owner via the existing `setApprovalForAll` can mint, and the Transfer event records the real caller.

**Files:**
- Modify: `contract/token.go` — `Mint`, lines ~362-365 (auth gate) and ~441 (Transfer event)
- Test: `test/editioned_define_test.go`

- [ ] **Step 1: Write the failing test**

Create `test/editioned_define_test.go`:

```go
package contract_test

import (
	"testing"
)

// Owner approves a market operator; the market mints to a buyer.
func TestDelegatedMintByApprovedOperator(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	// Owner approves the market as operator for all tokens.
	CallContract(t, ct, "setApprovalForAll",
		[]byte(`{"operator":"hive:market","approved":true}`),
		nil, ownerAddress, true, uint(150_000_000), "")

	// Market (caller != owner) mints to a buyer.
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"ed1","amount":3,"maxSupply":100,"data":""}`),
		nil, "hive:market", true, uint(150_000_000), "")

	// Buyer holds the tokens.
	CallContract(t, ct, "balanceOf",
		[]byte(`{"account":"hive:buyer","id":"ed1"}`),
		nil, "hive:buyer", true, uint(150_000_000), `{"balance":3}`)
}

// A caller that is neither owner nor approved cannot mint.
func TestDelegatedMintUnauthorizedCallerFails(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"ed1","amount":1,"maxSupply":10,"data":""}`),
		nil, "hive:stranger", false, uint(150_000_000), "")
}

// Revoking operator approval stops the market from minting.
func TestDelegatedMintAfterRevokeFails(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "setApprovalForAll",
		[]byte(`{"operator":"hive:market","approved":true}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"ed1","amount":1,"maxSupply":10,"data":""}`),
		nil, "hive:market", true, uint(150_000_000), "")
	CallContract(t, ct, "setApprovalForAll",
		[]byte(`{"operator":"hive:market","approved":false}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"ed1","amount":1,"maxSupply":10,"data":""}`),
		nil, "hive:market", false, uint(150_000_000), "")
}
```

- [ ] **Step 2: Build wasm and run the test to verify it fails**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
tinygo build -gc=custom -scheduler=none -panic=trap -no-debug -target=wasm-unknown -o test/artifacts/main.wasm ./contract && \
go test ./test/... -run 'TestDelegatedMint' -v
```

Expected: `TestDelegatedMintByApprovedOperator` and `TestDelegatedMintAfterRevokeFails` FAIL — the market call aborts with `Must be owner to mint` (gate is still owner-only). `TestDelegatedMintUnauthorizedCallerFails` already passes.

- [ ] **Step 3: Replace the auth gate in `Mint`**

In `contract/token.go`, in `func Mint`, replace:

```go
	owner, isOwner := getOwner()
	if !isOwner {
		sdk.Abort("Must be owner to mint")
	}
```

with:

```go
	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	operator := *caller
	ownerAddr := getOwnerAddress()
	if !isApprovedOrOwner(operator, ownerAddr) {
		sdk.Abort("Must be owner or approved operator to mint")
	}
```

- [ ] **Step 4: Update the mint Transfer event to record the real caller**

In `func Mint`, replace:

```go
	emitTransferSingle(owner, "", p.To, p.Id, p.Amount) // Mint: from is zero address
```

with:

```go
	emitTransferSingle(operator, "", p.To, p.Id, p.Amount) // Mint: from is zero address, operator is caller
```

- [ ] **Step 5: Build wasm and run the tests to verify they pass**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
tinygo build -gc=custom -scheduler=none -panic=trap -no-debug -target=wasm-unknown -o test/artifacts/main.wasm ./contract && \
go test ./test/... -run 'TestDelegatedMint' -v
```

Expected: all three PASS.

- [ ] **Step 6: Run mint regression to confirm owner path unbroken**

```bash
cd /home/dockeruser/magi/magi_nft-contract && go test ./test/... -run 'TestMint' -v
```

Expected: all existing `TestMint*` PASS (owner is a subset of the new gate).

- [ ] **Step 7: Commit**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
git add contract/token.go test/editioned_define_test.go test/artifacts/main.wasm && \
git commit -m "feat: allow owner-approved operator to call mint

- Replace Mint owner-only gate with isApprovedOrOwner(caller, owner)
  so a market the owner approved via setApprovalForAll can mint to a
  buyer (caller != owner, recipient = buyer).
- Stamp the real caller as the TransferSingle operator for provenance.
- Add delegated-mint tests (approved op mints, unauthorized fails,
  revoked operator fails)."
```

---

### Task 2: Delegated mint authorization for `MintSeries`

Apply the same gate + provenance change to `MintSeries`.

**Files:**
- Modify: `contract/token.go` — `MintSeries`, lines ~606-609 (auth gate) and ~734 (Transfer event)
- Test: `test/editioned_define_test.go`

- [ ] **Step 1: Write the failing test**

Append to `test/editioned_define_test.go`:

```go
// Owner-approved market mints a generated series to a buyer.
func TestDelegatedMintSeriesByApprovedOperator(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "setApprovalForAll",
		[]byte(`{"operator":"hive:market","approved":true}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mintSeries",
		[]byte(`{"to":"hive:buyer","idPrefix":"card-","startNumber":1,"count":3,"amount":1,"maxSupply":1}`),
		nil, "hive:market", true, uint(150_000_000), "")
	CallContract(t, ct, "balanceOf",
		[]byte(`{"account":"hive:buyer","id":"card-2"}`),
		nil, "hive:buyer", true, uint(150_000_000), `{"balance":1}`)
}

func TestDelegatedMintSeriesUnauthorizedFails(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mintSeries",
		[]byte(`{"to":"hive:buyer","idPrefix":"card-","startNumber":1,"count":2,"amount":1,"maxSupply":1}`),
		nil, "hive:stranger", false, uint(150_000_000), "")
}
```

- [ ] **Step 2: Build wasm and run the test to verify it fails**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
tinygo build -gc=custom -scheduler=none -panic=trap -no-debug -target=wasm-unknown -o test/artifacts/main.wasm ./contract && \
go test ./test/... -run 'TestDelegatedMintSeries' -v
```

Expected: `TestDelegatedMintSeriesByApprovedOperator` FAILS (`Must be owner to mint`); `TestDelegatedMintSeriesUnauthorizedFails` passes.

- [ ] **Step 3: Replace the auth gate in `MintSeries`**

In `func MintSeries`, replace:

```go
	owner, isOwner := getOwner()
	if !isOwner {
		sdk.Abort("Must be owner to mint")
	}
```

with:

```go
	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	operator := *caller
	ownerAddr := getOwnerAddress()
	if !isApprovedOrOwner(operator, ownerAddr) {
		sdk.Abort("Must be owner or approved operator to mint")
	}
```

- [ ] **Step 4: Update the series Transfer event to record the real caller**

In `func MintSeries`, replace:

```go
	emitTransferBatch(owner, "", p.To, ids, amounts) // Mint: from is zero address
```

with:

```go
	emitTransferBatch(operator, "", p.To, ids, amounts) // Mint: from is zero address, operator is caller
```

- [ ] **Step 5: Build wasm and run the tests to verify they pass**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
tinygo build -gc=custom -scheduler=none -panic=trap -no-debug -target=wasm-unknown -o test/artifacts/main.wasm ./contract && \
go test ./test/... -run 'TestDelegatedMintSeries' -v
```

Expected: both PASS.

- [ ] **Step 6: Run mintseries regression**

```bash
cd /home/dockeruser/magi/magi_nft-contract && go test ./test/... -run 'TestMintSeries' -v
```

Expected: existing `TestMintSeries*` PASS.

- [ ] **Step 7: Commit**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
git add contract/token.go test/editioned_define_test.go test/artifacts/main.wasm && \
git commit -m "feat: allow owner-approved operator to call mintSeries

- Apply the owner-or-approved-operator gate to MintSeries.
- Stamp the real caller as the TransferBatch operator.
- Add delegated mintSeries tests (approved op, unauthorized fails)."
```

---

### Task 3: Define edition without minting in `Mint` (`amount == 0`)

`amount == 0` becomes owner-only "define edition": set `maxSupply`/`soulbound`/`properties`, emit `tokenCreated`, mint nothing.

**Files:**
- Modify: `contract/token.go` — `Mint`, the `To`-required check (~377-379) and the `amount == 0` check (~385-387), insert define-only branch before the supply logic
- Test: `test/editioned_define_test.go`

- [ ] **Step 1: Write the failing test**

Append to `test/editioned_define_test.go`:

```go
// Owner defines an edition with zero supply; it exists; then it is mintable up to maxSupply.
func TestDefineEditionThenMintUpToMax(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	// Define: amount 0, no recipient needed.
	CallContract(t, ct, "mint",
		[]byte(`{"id":"drop1","amount":0,"maxSupply":5,"properties":"{\"rarity\":\"gold\"}","data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")

	// Defined edition exists with zero supply.
	CallContract(t, ct, "exists",
		[]byte(`{"id":"drop1"}`), nil, ownerAddress, true, uint(150_000_000), `{"exists":true}`)
	CallContract(t, ct, "totalSupply",
		[]byte(`{"id":"drop1"}`), nil, ownerAddress, true, uint(150_000_000), `{"totalSupply":0}`)

	// Mint up to maxSupply succeeds.
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"drop1","amount":5,"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")

	// One more exceeds max supply.
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"drop1","amount":1,"data":""}`),
		nil, ownerAddress, false, uint(150_000_000), "")
}

// Defining requires a maxSupply.
func TestDefineEditionRequiresMaxSupply(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"id":"drop1","amount":0,"data":""}`),
		nil, ownerAddress, false, uint(150_000_000), "")
}

// An already-defined (or minted) edition cannot be redefined.
func TestDefineEditionRedefineFails(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"id":"drop1","amount":0,"maxSupply":5,"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"id":"drop1","amount":0,"maxSupply":9,"data":""}`),
		nil, ownerAddress, false, uint(150_000_000), "")
}

// Defining is owner-only even for an approved operator.
func TestDefineEditionOperatorCannotDefine(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "setApprovalForAll",
		[]byte(`{"operator":"hive:market","approved":true}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"id":"drop1","amount":0,"maxSupply":5,"data":""}`),
		nil, "hive:market", false, uint(150_000_000), "")
}
```

- [ ] **Step 2: Build wasm and run the test to verify it fails**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
tinygo build -gc=custom -scheduler=none -panic=trap -no-debug -target=wasm-unknown -o test/artifacts/main.wasm ./contract && \
go test ./test/... -run 'TestDefineEdition' -v
```

Expected: `TestDefineEditionThenMintUpToMax`, `TestDefineEditionRedefineFails`, `TestDefineEditionOperatorCannotDefine` FAIL (define call aborts with `Amount must be greater than 0`). `TestDefineEditionRequiresMaxSupply` passes for the wrong reason (still fix it via the real path below).

- [ ] **Step 3: Make the `To`-required check skip define-only calls**

In `func Mint`, replace:

```go
	if p.To == "" {
		sdk.Abort("To address required")
	}
	validateAddress(p.To)
```

with:

```go
	if p.Amount != 0 {
		if p.To == "" {
			sdk.Abort("To address required")
		}
		validateAddress(p.To)
	}
```

- [ ] **Step 4: Replace the `amount == 0` abort with the define-only branch**

In `func Mint`, replace:

```go
	if p.Amount == 0 {
		sdk.Abort("Amount must be greater than 0")
	}

	// Check/set max supply for this token
	existingMax := getMaxSupply(p.Id)
```

with:

```go
	existingMax := getMaxSupply(p.Id)

	// Define-only mode: amount == 0 registers an edition without minting.
	if p.Amount == 0 {
		if operator != ownerAddr {
			sdk.Abort("Only owner can define an edition")
		}
		if existingMax != 0 {
			sdk.Abort("Edition already defined")
		}
		if p.MaxSupply == 0 {
			sdk.Abort("MaxSupply required for new token (1 = unique, >1 = editioned)")
		}
		setMaxSupply(p.Id, p.MaxSupply)
		if p.Soulbound {
			setSoulbound(p.Id)
		}
		if p.Properties != "" {
			setTokenProperties(p.Id, p.Properties)
			emitPropertiesSet(p.Id)
		}
		emitTokenCreated(p.Id, p.MaxSupply, p.Soulbound)
		return jsonResponse(SuccessResponse{Success: true})
	}

	// Check/set max supply for this token
```

Note: this introduces `existingMax` once at the top; the original `existingMax := getMaxSupply(p.Id)` line that followed the old `amount == 0` block is now removed by this replacement (it is included verbatim in the replaced text above, so the declaration is not duplicated).

- [ ] **Step 5: Build wasm and run the tests to verify they pass**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
tinygo build -gc=custom -scheduler=none -panic=trap -no-debug -target=wasm-unknown -o test/artifacts/main.wasm ./contract && \
go test ./test/... -run 'TestDefineEdition' -v
```

Expected: all four PASS.

- [ ] **Step 6: Full regression**

```bash
cd /home/dockeruser/magi/magi_nft-contract && go test ./test/...
```

Expected: entire suite PASS (the `amount == 0` path was previously a hard abort; all existing tests use `amount > 0`).

- [ ] **Step 7: Commit**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
git add contract/token.go test/editioned_define_test.go test/artifacts/main.wasm && \
git commit -m "feat: define an edition without minting via mint amount=0

- mint with amount==0 registers maxSupply/soulbound/properties and
  emits tokenCreated, minting nothing (owner-only, no recipient).
- Guard against redefine (existingMax != 0) and missing maxSupply.
- Skip the To-required check for define-only calls.
- Pre-defined editions are then mintable up to maxSupply via the
  existing subsequent-mint path.
- Add define-edition tests (define+mint-to-max, requires maxSupply,
  redefine fails, operator cannot define)."
```

---

### Task 4: Define a series without minting in `MintSeries` (`amount == 0`)

`amount == 0` in `MintSeries` defines every generated id (owner-only) with zero supply.

**Files:**
- Modify: `contract/token.go` — `MintSeries`: the `To`-required check (~621-624), the `amount == 0` check (~628-630), and the per-id loop body (~688-732)
- Test: `test/editioned_define_test.go`

- [ ] **Step 1: Write the failing test**

Append to `test/editioned_define_test.go`:

```go
// Owner defines a whole series with zero supply; each id exists and is then mintable.
func TestDefineEditionSeriesThenMint(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	CallContract(t, ct, "mintSeries",
		[]byte(`{"idPrefix":"set-","startNumber":1,"count":3,"amount":0,"maxSupply":10}`),
		nil, ownerAddress, true, uint(150_000_000), "")

	CallContract(t, ct, "exists",
		[]byte(`{"id":"set-2"}`), nil, ownerAddress, true, uint(150_000_000), `{"exists":true}`)
	CallContract(t, ct, "totalSupply",
		[]byte(`{"id":"set-2"}`), nil, ownerAddress, true, uint(150_000_000), `{"totalSupply":0}`)

	// Defined series id is mintable up to maxSupply.
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"set-2","amount":10,"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"set-2","amount":1,"data":""}`),
		nil, ownerAddress, false, uint(150_000_000), "")
}

// Defining a series is owner-only.
func TestDefineEditionSeriesOperatorCannotDefine(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "setApprovalForAll",
		[]byte(`{"operator":"hive:market","approved":true}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mintSeries",
		[]byte(`{"idPrefix":"set-","startNumber":1,"count":2,"amount":0,"maxSupply":10}`),
		nil, "hive:market", false, uint(150_000_000), "")
}
```

- [ ] **Step 2: Build wasm and run the test to verify it fails**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
tinygo build -gc=custom -scheduler=none -panic=trap -no-debug -target=wasm-unknown -o test/artifacts/main.wasm ./contract && \
go test ./test/... -run 'TestDefineEditionSeries' -v
```

Expected: `TestDefineEditionSeriesThenMint` FAILS (`Amount must be greater than 0`); `TestDefineEditionSeriesOperatorCannotDefine` FAILS for the wrong reason (also `Amount must be greater than 0`, not the owner check) — Step 5 fixes both via the real path.

- [ ] **Step 3: Make the `To`-required check skip define-only series**

In `func MintSeries`, replace:

```go
	if p.To == "" {
		sdk.Abort("To address required")
	}
	validateAddress(p.To)
```

with:

```go
	if p.Amount != 0 {
		if p.To == "" {
			sdk.Abort("To address required")
		}
		validateAddress(p.To)
	}
```

- [ ] **Step 4: Replace the `amount == 0` abort with an owner-only define gate**

In `func MintSeries`, replace:

```go
	if p.Amount == 0 {
		sdk.Abort("Amount must be greater than 0")
	}
```

with:

```go
	defineOnly := p.Amount == 0
	if defineOnly && operator != ownerAddr {
		sdk.Abort("Only owner can define an edition")
	}
```

- [ ] **Step 5: Handle define-only in the per-id loop**

In `func MintSeries`, the per-id loop has a first-mint branch `if existingMax == 0 { ... }`. Replace that whole first-mint block:

```go
		existingMax := getMaxSupply(id)
		if existingMax == 0 {
			// First mint — we know balance, totalSupply, totalMinted are all 0.
			// Skip reads and write directly.
			if p.Amount > p.MaxSupply {
				sdk.Abort("Would exceed max supply")
			}
			setMaxSupply(id, p.MaxSupply)
			if p.Soulbound {
				setSoulbound(id)
			}
			// When using propertiesTemplate, only the template token gets properties stored;
			// copies inherit via the template relationship event.
			if setProps || (setTemplateProps && id == p.PropertiesTemplate) {
				setTokenProperties(id, p.Properties)
				emitPropertiesSet(id)
			}
			if trackMinted {
				sdk.StateSetObject(totalMintedKey(id), string(u64ToBytes(p.Amount)))
			}
			setBalance(p.To, id, p.Amount)
			sdk.StateSetObject(totalSupplyKey(id), string(u64ToBytes(p.Amount)))
			emitTokenCreated(id, p.MaxSupply, p.Soulbound)
		} else {
```

with:

```go
		existingMax := getMaxSupply(id)
		if existingMax == 0 {
			// First touch — balance, totalSupply, totalMinted are all 0.
			setMaxSupply(id, p.MaxSupply)
			if p.Soulbound {
				setSoulbound(id)
			}
			// When using propertiesTemplate, only the template token gets properties stored;
			// copies inherit via the template relationship event.
			if setProps || (setTemplateProps && id == p.PropertiesTemplate) {
				setTokenProperties(id, p.Properties)
				emitPropertiesSet(id)
			}
			emitTokenCreated(id, p.MaxSupply, p.Soulbound)
			if !defineOnly {
				if p.Amount > p.MaxSupply {
					sdk.Abort("Would exceed max supply")
				}
				if trackMinted {
					sdk.StateSetObject(totalMintedKey(id), string(u64ToBytes(p.Amount)))
				}
				setBalance(p.To, id, p.Amount)
				sdk.StateSetObject(totalSupplyKey(id), string(u64ToBytes(p.Amount)))
			}
		} else if defineOnly {
			sdk.Abort("Edition already defined")
		} else {
```

- [ ] **Step 6: Skip the batch Transfer event for define-only series**

In `func MintSeries`, replace:

```go
	emitTransferBatch(operator, "", p.To, ids, amounts) // Mint: from is zero address, operator is caller
```

with:

```go
	if !defineOnly {
		emitTransferBatch(operator, "", p.To, ids, amounts) // Mint: from is zero address, operator is caller
	}
```

(If Task 2 has not been applied, the line still reads `emitTransferBatch(owner, ...)`; wrap whichever form is present in the same `if !defineOnly { ... }` guard.)

- [ ] **Step 7: Build wasm and run the tests to verify they pass**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
tinygo build -gc=custom -scheduler=none -panic=trap -no-debug -target=wasm-unknown -o test/artifacts/main.wasm ./contract && \
go test ./test/... -run 'TestDefineEditionSeries' -v
```

Expected: both PASS.

- [ ] **Step 8: Full regression**

```bash
cd /home/dockeruser/magi/magi_nft-contract && go test ./test/...
```

Expected: entire suite PASS. Pay attention to `TestMintSeries*` and `editions_benchmark`/`template_benchmark` tests (the first-touch block was restructured but the `amount > 0` behavior is preserved: same writes, same overflow guard).

- [ ] **Step 9: Commit**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
git add contract/token.go test/editioned_define_test.go test/artifacts/main.wasm && \
git commit -m "feat: define a series without minting via mintSeries amount=0

- mintSeries with amount==0 defines every generated id
  (maxSupply/soulbound/properties + tokenCreated) with zero supply,
  owner-only, no recipient, no TransferBatch.
- Restructure the first-touch loop branch so define-only skips
  balance/supply writes; redefine of an existing id aborts.
- amount>0 behavior unchanged. Add series define tests."
```

---

### Task 5: Update stale `Exists` comment + final verification

**Files:**
- Modify: `contract/token.go` — `Exists`, comment at ~1252

- [ ] **Step 1: Fix the stale comment**

In `func Exists`, replace:

```go
	// A token exists if it has a maxSupply set (meaning it was minted at least once)
	exists := getMaxSupply(p.Id) > 0
```

with:

```go
	// A token exists once its maxSupply is set — either by a mint or by a
	// define-only call (mint/mintSeries with amount == 0), even at zero supply.
	exists := getMaxSupply(p.Id) > 0
```

- [ ] **Step 2: Full regression**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
tinygo build -gc=custom -scheduler=none -panic=trap -no-debug -target=wasm-unknown -o test/artifacts/main.wasm ./contract && \
go test ./test/...
```

Expected: entire suite PASS, including all new `TestDelegatedMint*` and `TestDefineEdition*` tests.

- [ ] **Step 3: Commit**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
git add contract/token.go test/artifacts/main.wasm && \
git commit -m "docs: clarify Exists covers define-only editions

- A token exists once maxSupply is set, including via amount==0
  define-only calls at zero supply."
```

- [ ] **Step 4: Push the branch to origin (no PR)**

```bash
cd /home/dockeruser/magi/magi_nft-contract && \
git push -u origin feat/editioned-define-delegated-mint
```

Do **not** open a PR. Report the pushed branch back to the user. The pending
`origin/main` force-push (fork main → upstream) is a separate item — raise it with the
user; do not perform it as part of this plan.

---

## Self-Review

**Spec coverage:**
- "Define edition without minting" → Task 3 (`Mint`), Task 4 (`MintSeries`). ✓
- "Mint up to maxSupply" → existing subsequent-mint path; asserted in Task 3 Step 1 and Task 4 Step 1. ✓
- "Market mints on behalf, caller != owner" → Task 1, Task 2 (owner-or-approved-operator gate + caller-as-operator provenance). ✓
- "Owner-only define" → Task 3 Step 4 / Task 4 Step 4 owner check; tested in Task 3 & 4. ✓
- "Reuse ERC-6909 operator approval, no per-token mint cap" → Task 1/2 use `isApprovedOrOwner`; no allowance decrement added. ✓
- "MintBatch + payment out of scope" → not touched. ✓
- "Exists reports defined editions" → Task 5 comment; behavior already correct, asserted in Task 3/4. ✓
- "tinyjson not regenerated" → no payload struct change; called out in File Structure. ✓

**Placeholder scan:** No TBD/TODO; every code step shows full code and exact commands. ✓

**Type/name consistency:** `operator`, `ownerAddr`, `caller`, `defineOnly`, `existingMax` used consistently across Tasks 1–4. `getOwnerAddress()`, `isApprovedOrOwner(caller, from)`, `getMaxSupply`, `setMaxSupply`, `setSoulbound`, `setTokenProperties`, `emitPropertiesSet`, `emitTokenCreated`, `emitTransferSingle`, `emitTransferBatch` match the signatures verified in `contract/internal.go`/`contract/events.go`. Task 4 Step 6 notes the Task-2 ordering dependency on the `emitTransferBatch` line form. ✓
