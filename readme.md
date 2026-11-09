# MAGI NFT

ERC-1155 compliant multi-token contract for the Magi Network.

## Overview

MAGI NFT is a multi-token implementation following the ERC-1155 standard. It supports both fungible and non-fungible tokens within a single contract, with features for pausability, ownership management, batch operations, and enforced supply limits.

## Token Configuration

Token properties are configured at initialization via the `init` payload:

| Property    | Type   | Description                                                          |
|-------------|--------|----------------------------------------------------------------------|
| name        | string | Collection name (e.g., "Magi NFT") — max 64 characters              |
| symbol      | string | Collection symbol (e.g., "MNFT") — max 16 characters                |
| baseUri     | string | Base URI for token metadata — must end with `/` when non-empty       |
| trackMinted | bool   | Optional: if true, burned tokens cannot be re-minted                 |
| metadata    | any    | Optional: arbitrary JSON collection metadata (description, icon, …)  |

Example init payload:
```json
{"name": "Magi NFT", "symbol": "MNFT", "baseUri": "https://api.magi.network/metadata/"}
```

With collection metadata and permanent burn tracking:
```json
{
  "name": "Magi NFT",
  "symbol": "MNFT",
  "baseUri": "https://api.magi.network/metadata/",
  "trackMinted": true,
  "metadata": {"description": "My collection", "icon": "https://example.com/icon.png"}
}
```

## Features

- **ERC-1155 Compliant**: Multi-token standard interface supporting both fungible and non-fungible tokens
- **Supply Enforcement**: Required `maxSupply` on mint (1 = unique NFT, >1 = editioned)
- **Soulbound Tokens**: Non-transferable tokens for badges, credentials, memberships
- **Batch Operations**: Efficient batch minting, burning, and transfers
- **Series Minting**: Compact single-payload mint for large runs of sequential IDs (`mintSeries`)
- **Operator Approval**: Approve operators to manage all your tokens (ERC-1155)
- **Per-Token Approval**: Approve specific token amounts per spender (ERC-6909)
- **Token Properties**: Attach arbitrary JSON metadata to tokens at mint time
- **Properties Templates**: Share properties across tokens with template inheritance (saves ~84% RC)
- **Custom URIs**: Set individual URIs per token or use baseUri fallback
- **Mintable**: Owner can mint single or batch tokens
- **Burnable**: Owner can burn single or batch tokens
- **Pausable**: Owner can pause/unpause all transfers
- **Ownership Transfer**: Owner can transfer contract ownership

## ERC-1155 Compliance

### Standard Functions (ERC-1155 Core)

| Function | Type | Standard |
|----------|------|----------|
| `balanceOf` | Query | ERC-1155 |
| `balanceOfBatch` | Query | ERC-1155 |
| `safeTransferFrom` | Action | ERC-1155 |
| `safeBatchTransferFrom` | Action | ERC-1155 |
| `setApprovalForAll` | Action | ERC-1155 |
| `isApprovedForAll` | Query | ERC-1155 |
| `uri` | Query | ERC-1155 |
| `TransferSingle` | Event | ERC-1155 |
| `TransferBatch` | Event | ERC-1155 |
| `ApprovalForAll` | Event | ERC-1155 |
| `URI` | Event | ERC-1155 |

### Extensions

| Feature | Source |
|---------|--------|
| `supportsInterface` | ERC-165 Standard |
| `totalSupply` / `exists` | ERC-1155 Supply Extension |
| `mint` / `mintBatch` / `mintSeries` | Common pattern (not in spec) |
| `burn` / `burnBatch` | Common pattern (not in spec) |
| `pause` / `unpause` | OpenZeppelin Pausable |
| `changeOwner` / `getOwner` | OpenZeppelin Ownable |
| `maxSupply` | Custom |
| `trackMinted` / `totalMinted` | Custom |
| `soulbound` / `isSoulbound` | Inspired by EIP-5192 |
| `approve` / `allowance` | ERC-6909 (per-token approval) |
| `setProperties` / `getProperties` | Custom |
| `propertiesTemplate` (on mint) | Custom |
| `setURI` / `setBaseURI` | Custom |
| `getInfo` | Custom |

## Supply Management

The first mint of a token requires `maxSupply` to define the token type:

### Unique NFTs (1/1)
```json
{"to": "hive:collector", "id": "art-001", "amount": 1, "maxSupply": 1, "data": ""}
```
Once minted, no more copies can ever be created.

### Editioned NFTs
```json
{"to": "hive:collector", "id": "art-002", "amount": 50, "maxSupply": 100, "data": ""}
```
Up to 100 total can be minted across multiple transactions.

### Subsequent Mints
For existing tokens, `maxSupply` is optional:
```json
{"to": "hive:collector", "id": "art-002", "amount": 25, "data": ""}
```
Uses the stored maxSupply. If provided, it must match.

### Supply Rules
- `maxSupply` is **required** on first mint (1 = unique, >1 = editioned)
- First mint locks the max supply for that token ID (cannot be changed)
- Subsequent mints can omit `maxSupply` (uses stored value)
- Minting fails if total would exceed max supply
- Burning decreases total supply (but max supply stays the same)

### Burn Behavior (trackMinted)

| Setting | Behavior | Use Case |
|---------|----------|----------|
| `trackMinted: false` (default) | Burned tokens can be re-minted up to maxSupply | Gaming items, editions |
| `trackMinted: true` | Burned tokens are gone forever | Collectibles, certificates |

When `trackMinted` is enabled:
- `totalMinted` tracks all tokens ever minted (never decreases)
- Supply check uses `totalMinted` instead of `totalSupply`
- After burning, those slots cannot be re-minted

## Soulbound Tokens

Soulbound tokens are non-transferable tokens that become bound to the recipient. The contract owner can distribute soulbound tokens, but once received, recipients cannot transfer them further. They can still be burned.

### Use Cases
- **Achievement Badges**: Gaming accomplishments bound to player
- **Certifications**: Educational credentials bound to recipient
- **Memberships**: Non-transferable membership tokens
- **Identity Tokens**: KYC/verification tokens
- **Proof of Attendance**: Event participation tokens (POAPs)

### Distribution Workflow

The owner can pre-mint soulbound tokens and distribute them later:

```json
// Step 1: Owner mints 10 soulbound badges to themselves
{"to": "hive:owner", "id": "badge-001", "amount": 10, "maxSupply": 100, "soulbound": true, "data": ""}

// Step 2: Owner distributes badges to recipients
{"from": "hive:owner", "to": "hive:player", "id": "badge-001", "amount": 1, "data": ""}

// Step 3: Recipient CANNOT transfer further (bound to them)
```

Or mint directly to recipients:
```json
{"to": "hive:player", "id": "badge-001", "amount": 1, "maxSupply": 1, "soulbound": true, "data": ""}
```

### Batch Minting with Mixed Soulbound

```json
{
  "to": "hive:owner",
  "ids": ["badge", "item"],
  "amounts": [1, 5],
  "maxSupplies": [1, 100],
  "soulbound": [true, false],
  "data": ""
}
```

### Soulbound Rules
- `soulbound` can only be set on the **first mint** of a token (like maxSupply)
- Once set, the soulbound status cannot be changed
- **Owner can transfer** soulbound tokens (for distribution)
- **Recipients cannot transfer** soulbound tokens (bound to them)
- Soulbound tokens **can be burned** (for revocation scenarios)
- Use `isSoulbound` query to check a token's status

### Soulbound vs Regular Tokens

| Workflow | Regular Token | Soulbound Token |
|----------|---------------|-----------------|
| Pre-mint to owner | ✅ Supported | ✅ Supported |
| Owner distributes | ✅ Supported | ✅ Supported |
| Recipient transfers | ✅ Supported | ❌ Blocked |
| Burn | ✅ Supported | ✅ Supported |

## Token Properties

Each token can have arbitrary JSON properties attached to it. Properties are set during the first mint and can be updated later by the contract owner.

### Setting Properties at Mint

Include `properties` in the mint payload (any valid JSON value):

```json
{"to": "hive:collector", "id": "art-001", "amount": 1, "maxSupply": 1, "properties": {"rarity": "legendary", "power": 42}, "data": ""}
```

Properties can be any valid JSON — objects, arrays, strings, numbers, or booleans:

```json
{"to": "hive:collector", "id": "tag-001", "amount": 1, "maxSupply": 1, "properties": "hello, world", "data": ""}
```

### Batch Minting with Properties

```json
{
  "to": "hive:owner",
  "ids": ["sword", "shield"],
  "amounts": [1, 1],
  "maxSupplies": [1, 1],
  "properties": [{"damage": 50}, {"defense": 30}],
  "data": ""
}
```

### Updating Properties

The contract owner can update properties after mint:

```json
{"id": "art-001", "properties": {"rarity": "legendary", "power": 99, "enchanted": true}}
```

### Reading Properties

Anyone can query token properties:

```json
{"id": "art-001"}
```

Returns `{"properties": ...}` with the stored JSON, or `{"properties": null}` if no properties are set.

### Properties Rules
- Properties can only be set on the **first mint** of a token (like maxSupply and soulbound)
- Subsequent mints of the same token ignore the `properties` field
- The contract **owner** can update properties at any time via `setProperties`
- Anyone can read properties via `getProperties`
- No length limitations (gas cost is the natural limit)
- Any valid JSON value is accepted

### Properties Templates

Instead of writing properties to every token individually, you can designate one token as a **template** and have other tokens inherit its properties. This saves ~84% RC when minting large batches of tokens with identical properties that might change later on (like statistics for game items for example). If the properties never change we highly recommend you to use editions instead.

#### How It Works

1. Mint a batch with `propertiesTemplate` pointing to one token ID
2. The first token gets explicit `properties`, the rest inherit from the template
3. `getProperties` on any child returns the template's properties
4. `setProperties` on a child overrides the template for that token only
5. `setProperties` on the template propagates to all children that haven't been overridden

#### Example: Minting 100 Game Cards

```json
{
  "to": "hive:owner",
  "ids": ["card-0", "card-1", "card-2", "..."],
  "amounts": [1, 1, 1, "..."],
  "maxSupplies": [1, 1, 1, "..."],
  "properties": [{"name": "Fire Mage", "hp": 80, "attack": 70}],
  "propertiesTemplate": "card-0",
  "data": ""
}
```

- `card-0` stores properties directly and becomes the template
- `card-1`, `card-2`, ... all reference `card-0` and inherit its properties
- Only 1 state write for properties instead of 100

#### Updating Individual Cards

```json
{"id": "card-5", "properties": {"name": "Fire Mage", "hp": 60, "attack": 90}}
```

Now `card-5` has custom stats, while all other cards still inherit from `card-0`.

#### Single Mint with Template

```json
{"to": "hive:owner", "id": "card-101", "amount": 1, "maxSupply": 1, "propertiesTemplate": "card-0", "data": ""}
```

#### Template Rules
- Template reference is set on **first mint** only (like properties)
- If both `properties` and `propertiesTemplate` are provided, explicit `properties` wins
- Template tokens are **non-transferable** (they serve as shared metadata anchors)
- Template tokens can still be burned
- `setProperties` on a child overrides the template for that token
- Updating the template's properties via `setProperties` affects all children immediately
- Only one level of template indirection is supported (no chaining)

## Mint Series

`mintSeries` is a compact alternative to `mintBatch` for minting a large run of tokens that share the same settings. Instead of sending an array of IDs, you provide a prefix, an optional suffix, a start number, and a count — the contract generates the IDs `idPrefix + (startNumber + i) + idSuffix`.

This solves the JSON payload size limit when minting hundreds of NFTs in one transaction where the IDs are the only thing that differs.

### Payload

```json
{
  "to": "hive:collector",
  "idPrefix": "card-",
  "idSuffix": "-rare",
  "startNumber": 1,
  "count": 100,
  "amount": 1,
  "maxSupply": 1,
  "soulbound": false,
  "properties": {"rarity": "common"}
}
```

This mints `card-1-rare` through `card-100-rare`, each unique (maxSupply=1), all with the same properties. The `idSuffix` field is optional — omit it to generate IDs like `card-1` through `card-100`.

### With Template Properties

`mintSeries` supports `propertiesTemplate` just like `mintBatch`. The template ID must be one of the generated IDs. Only the template token stores properties on-chain; all other tokens inherit via the `templateMint` event.

```json
{
  "to": "hive:collector",
  "idPrefix": "card-",
  "idSuffix": "-rare",
  "startNumber": 1,
  "count": 100,
  "amount": 1,
  "maxSupply": 1,
  "properties": {"rarity": "common", "power": 42},
  "propertiesTemplate": "card-1-rare"
}
```

Here `card-1-rare` stores the properties and serves as the template. `card-2-rare` through `card-100-rare` inherit from it. This saves ~5% RC compared to storing properties on every token.

### Rules

- `idPrefix` and `idSuffix` may not contain `|`
- Generated IDs are validated the same as any token ID (max 256 chars, no `|`)
- `idSuffix` is optional — when omitted, IDs are `idPrefix + number`
- `amount`, `maxSupply`, `soulbound`, and `properties` apply identically to every token in the series
- When `propertiesTemplate` is set, only the template token stores properties; copies inherit via `templateMint` event
- `propertiesTemplate` must be one of the generated IDs (e.g. `idPrefix + startNumber + idSuffix`) or an existing NFT
- Properties and soulbound status are set on **first mint** only (like `mint`/`mintBatch`)
- Emits a single `TransferBatch` event covering all minted tokens
- Count is unlimited — gas is the only constraint

### RC cost vs mintBatch

| Approach | What varies per token |
|----------|-----------------------|
| `mintBatch` | Full ID string in payload |
| `mintSeries` | Nothing — IDs are computed on-chain |
| `mintSeries` + template | Nothing — IDs computed on-chain, properties stored once |

For a 50-token run with template properties: `mintSeries` costs 12,056 RC vs `mintBatch` at 14,709 RC — **18% cheaper**. `mintSeries` skips redundant state reads on first mint (balance, totalSupply, totalMinted are known to be 0). Without template, `mintSeries` costs 12,057 RC. Both approaches avoid payload size limits, but `mintSeries` scales to any count in a single call.

## Functions

### Actions (State-Changing)

| Function               | Payload                                                        | Access |
|------------------------|----------------------------------------------------------------|--------|
| `init`                 | `{"name": string, "symbol": string, "baseUri": string, "trackMinted": bool}` | ContractOwner |
| `mint`                 | `{"to": string, "id": string, "amount": uint64, "maxSupply": uint64, "soulbound": bool, "properties": any, "propertiesTemplate": string, "data": string}` | Owner |
| `mintBatch`            | `{"to": string, "ids": []string, "amounts": []uint64, "maxSupplies": []uint64, "soulbound": []bool, "properties": []any, "propertiesTemplate": string, "data": string}` | Owner |
| `mintSeries`           | `{"to": string, "idPrefix": string, "idSuffix": string, "startNumber": uint64, "count": uint64, "amount": uint64, "maxSupply": uint64, "soulbound": bool, "properties": any, "propertiesTemplate": string}` | Owner |
| `burn`                 | `{"from": string, "id": string, "amount": uint64}`             | Owner |
| `burnBatch`            | `{"from": string, "ids": []string, "amounts": []uint64}`       | Owner |
| `safeTransferFrom`     | `{"from": string, "to": string, "id": string, "amount": uint64, "data": string}` | Owner/Operator |
| `safeBatchTransferFrom`| `{"from": string, "to": string, "ids": []string, "amounts": []uint64, "data": string}` | Owner/Operator |
| `setApprovalForAll`    | `{"operator": string, "approved": bool}`                       | Any |
| `approve`              | `{"spender": string, "id": string, "amount": uint64}`          | Any |
| `setProperties`        | `{"id": string, "properties": any}`                            | Owner |
| `setCollectionMetadata`| `{"metadata": any}`                                            | Owner |
| `setURI`               | `{"id": string, "uri": string}`                                | Owner |
| `setBaseURI`           | `{"baseUri": string}`                                          | Owner |
| `pause`                | -                                                              | Owner |
| `unpause`              | -                                                              | Owner |
| `changeOwner`          | `{"newOwner": string}`                                         | Owner |

### Queries (Read-Only)

| Function           | Payload                                      | Response                     |
|--------------------|----------------------------------------------|------------------------------|
| `balanceOf`        | `{"account": string, "id": string}`          | `{"balance": uint64}`        |
| `balanceOfBatch`   | `{"accounts": []string, "ids": []string}`    | `{"balances": []uint64}`     |
| `totalSupply`      | `{"id": string}`                             | `{"totalSupply": uint64}`    |
| `maxSupply`        | `{"id": string}`                             | `{"maxSupply": uint64}`      |
| `totalMinted`      | `{"id": string}`                             | `{"totalMinted": uint64}`    |
| `exists`           | `{"id": string}`                             | `{"exists": bool}`           |
| `isSoulbound`      | `{"id": string}`                             | `{"soulbound": bool}`        |
| `getProperties`         | `{"id": string}`                             | `{"properties": any\|null}`  |
| `getCollectionMetadata` | -                                            | `{"metadata": any\|null}`    |
| `isApprovedForAll`      | `{"account": string, "operator": string}`    | `{"approved": bool}`         |
| `allowance`             | `{"owner": string, "spender": string, "id": string}` | `{"amount": uint64}`  |
| `uri`              | `{"id": string}`                             | `{"uri": string}`            |
| `getOwner`         | -                                            | `{"owner": string}`          |
| `getInfo`          | -                                            | `{"name", "symbol", "baseUri", "trackMinted"}` |
| `isPaused`         | -                                            | `{"paused": bool}`           |
| `supportsInterface`| `{"interfaceId": string}`                    | `{"supported": bool}`        |

## Events

All events include `type` and `attributes`.

| Event Type       | Attributes                                    |
|------------------|-----------------------------------------------|
| `init_magi_nft`  | `owner`, `name`, `symbol`, `baseUri`          |
| `TransferSingle` | `operator`, `from`, `to`, `id`, `value`       |
| `TransferBatch`  | `operator`, `from`, `to`, `ids`, `values`     |
| `ApprovalForAll` | `account`, `operator`, `approved`             |
| `Approval`       | `owner`, `spender`, `id`, `amount`            |
| `URI`            | `value`, `id`                                 |
| `baseUriChange`  | `previousUri`, `newUri`                       |
| `ownerChange`    | `previousOwner`, `newOwner`                   |
| `paused`         | `by`                                          |
| `unpaused`       | `by`                                          |

### ERC-1155 Event Compliance

- **Mint**: Emits `TransferSingle` or `TransferBatch` with `from: ""`
- **Burn**: Emits `TransferSingle` or `TransferBatch` with `to: ""`
- **SetURI**: Emits `URI` event when token URI is updated
- **SetBaseURI**: Emits `baseUriChange` event when base URI is updated

## Operator Approval Pattern

Unlike ERC-20's per-token allowances, ERC-1155 uses operator approval. An approved operator can transfer ALL tokens of ALL types on behalf of the approver.

### How It Works

```
1. User approves operator for all tokens
2. Operator can transfer any token type/amount from user
3. Approval remains until explicitly revoked
```

### Marketplace Integration

**Step 1: User Approves Marketplace**

```json
{
  "action": "setApprovalForAll",
  "payload": {"operator": "hive:marketplace", "approved": true}
}
```

**Step 2: Marketplace Executes Transfer**

```json
{
  "action": "safeTransferFrom",
  "payload": {
    "from": "hive:seller",
    "to": "hive:buyer",
    "id": "42",
    "amount": 1,
    "data": ""
  }
}
```

**Step 3: Revoke Approval (Optional)**

```json
{
  "action": "setApprovalForAll",
  "payload": {"operator": "hive:marketplace", "approved": false}
}
```

## Per-Token Approval (ERC-6909)

`setApprovalForAll` is a blanket approval — the operator gets access to **all** tokens. For finer control, use `approve` to grant a spender access to a specific token ID with a specific amount. This is cherry-picked from the [ERC-6909](https://eips.ethereum.org/EIPS/eip-6909) standard.

### How It Works

```
1. User approves a spender for a specific token ID and amount
2. Spender can transfer up to the approved amount of that token
3. Allowance is decremented on each transfer
4. Setting amount to 0 revokes the approval
```

Transfer authorization checks (in order):
1. **Owner** — caller is the `from` address (always allowed)
2. **Operator** — caller has blanket `setApprovalForAll` approval (no allowance consumed)
3. **Allowance** — caller has sufficient per-token `approve` allowance (decremented on transfer)

### Example: Approve a Marketplace for 1 NFT

**Step 1: Approve**

```json
{
  "action": "approve",
  "payload": {"spender": "hive:marketplace", "id": "card-42", "amount": 1}
}
```

**Step 2: Marketplace Transfers**

```json
{
  "action": "safeTransferFrom",
  "payload": {"from": "hive:seller", "to": "hive:buyer", "id": "card-42", "amount": 1, "data": ""}
}
```

After the transfer, the allowance is 0 — the marketplace cannot transfer any more of this token.

### RC Cost

Per-token approval costs ~178 RC per call. Transfers via allowance cost ~408 RC (vs ~281 RC for owner transfers) due to the extra allowance read + write.

| Authorization | `approve` | `safeTransferFrom` | Total |
|---------------|---:|---:|---:|
| Owner (direct) | — | 281 | 281 |
| Operator (blanket) | 162 (one-time) | 281 | 281+ |
| Allowance (per-token) | 178 | 408 | 586 |

The per-token approach is more expensive per transfer but more secure — the spender can only touch exactly what was approved.

## Collection Metadata

Each collection can have arbitrary JSON metadata attached to it, readable on-chain via `getStateByKeys` (key: `collection_metadata`) or via the `getCollectionMetadata` query.

### Setting at Init

```json
{
  "name": "Magi NFT",
  "symbol": "MNFT",
  "baseUri": "https://api.magi.network/metadata/",
  "metadata": {"description": "My collection", "icon": "https://example.com/icon.png"}
}
```

### Updating Later

```json
{"metadata": {"description": "Updated description", "icon": "https://example.com/new-icon.png"}}
```

Only the contract owner can update. Any valid JSON value is accepted.

## Input Validation & Security

The contract enforces the following limits on all user-controlled inputs:

| Field       | Constraint                              |
|-------------|-----------------------------------------|
| `name`      | Max 64 characters                       |
| `symbol`    | Max 16 characters                       |
| `baseUri`   | Max 1024 characters, must end with `/`  |
| `uri`       | Max 1024 characters                     |
| `account`   | Max 256 characters, no `\|` character   |
| `tokenId`   | Max 256 characters, no `\|` character   |

The `\|` character is rejected in addresses and token IDs because it is used as a delimiter in internal state keys (e.g., `bal|account|tokenId`). Pipe injection could cause state key collisions.

## URI Management

### Default Behavior

URIs are constructed as `baseUri + tokenId`:
- baseUri: `https://api.magi.network/metadata/`
- tokenId: `123`
- Result: `https://api.magi.network/metadata/123`

### Custom URIs

Set a specific URI for individual tokens:

```json
{
  "action": "setURI",
  "payload": {"id": "1", "uri": "https://custom.example.com/token1.json"}
}
```

Custom URIs take precedence over the baseUri pattern.

## SDK

`sdk/` is vendored **verbatim** from
[vsc-eco/go-contract-template](https://github.com/vsc-eco/go-contract-template)
and is byte-identical to the copy in every other Magi contract repo. Do not
edit it here — change it in the template and re-sync:

```bash
/path/to/go-contract-template/scripts/sync-sdk.sh .
```

The SDK exposes the host functions the node publishes in
`modules/wasm/sdk/sdk.go`. Which of them this repo's tests can actually execute
is set by the `replace vsc-node` pin in `go.mod`; a wrapper whose host function
is absent from that node makes the module fail to instantiate, so bump the pin
before calling a newer one.

## Build

```bash
tinygo build -gc=custom -scheduler=none -panic=trap -no-debug -target=wasm-unknown -o test/artifacts/main.wasm ./contract
```

## Test

```bash
go test ./test/...
```

## Project Structure

```
magi_nft/
├── contract/
│   ├── main.go            # Entry point and state helpers
│   ├── token.go           # Exported WASM functions
│   ├── internal.go        # Internal helper functions
│   ├── types.go           # Type definitions
│   ├── types_tinyjson.go  # JSON serialization (tinyjson)
│   └── events.go          # Event emission
├── sdk/                   # VSC SDK bindings
├── test/
│   ├── helpers_test.go    # Test utilities
│   ├── init_test.go       # Initialization tests
│   ├── mint_test.go       # Mint/MintBatch tests
│   ├── mintseries_test.go # MintSeries tests
│   ├── burn_test.go       # Burn/BurnBatch tests
│   ├── transfer_test.go   # Transfer tests
│   ├── balance_test.go    # Balance query tests
│   ├── supply_test.go     # Supply management tests
│   ├── approval_test.go   # Operator approval tests (ERC-1155)
│   ├── approve_test.go    # Per-token approval tests (ERC-6909)
│   ├── uri_test.go        # URI management tests
│   ├── soulbound_test.go  # Soulbound token tests
│   ├── properties_test.go # Token properties tests
│   ├── trackminted_test.go# Track minted tests
│   ├── admin_test.go      # Admin/ownership tests
│   ├── lifecycle_test.go  # Lifecycle tests
│   ├── erc165_test.go     # ERC-165 interface tests
│   ├── benchmark_test.go  # RC consumption benchmark
│   ├── metadata_test.go   # Collection metadata tests
│   └── artifacts/         # Compiled WASM
└── readme.md
```

## RC Consumption

| Function               | Avg RC   |
|------------------------|----------|
| Queries                | 100      |
| unpause                | 100      |
| pause                  | 106      |
| setApprovalForAll      | 162      |
| approve                | 178      |
| changeOwner            | 183      |
| burn (unique)          | 202      |
| setBaseURI             | 272      |
| safeTransferFrom (owner) | 281–348 |
| safeTransferFrom (allowance) | 408 |
| mint (no properties)   | 418      |
| mint (with properties) | 427–1488 |
| setProperties          | 608      |
| setURI                 | 838      |
| init                   | 1,298    |
| mintSeries             | ~241 per token |
| mintSeries (template)  | ~241 per token |
| mintBatch              | ~285 per token (with template) |
| safeBatchTransferFrom  | ~139 per token |
| burnBatch              | ~87 per token  |

Multiply by batch size as usual: a full 256-entry `mintBatch` measures 68,833 RC
(6.88B gas), so batch callers need an hbd balance — RC is the hbd balance plus the
10,000 free allowance, and the gas budget is `min(availableRC, RcLimit)`.

## Benchmark: Real-World Scenario

RC consumption for realistic NFT use-cases (see `test/benchmark_test.go`). All values in RC (1000 RC = 1 HBD in the wallet).

### Minting

| Function | Scenario | RC |
|----------|----------|---:|
| `init` | Init contract | 1,298 |
| `mint` | 10 unique NFTs — total | 4,180 |
| `mint` | 10 unique NFTs — avg per call | 418 |
| `mint` | 10,000 editions of 1 NFT with properties | 1,755 |
| `mintSeries` | 10 unique NFTs | 2,371 |
| `mintSeries` | 50 unique NFTs | 12,057 |
| `mintSeries` | 50 unique NFTs with template properties | 12,056 |
| `mintBatch` | 50 unique NFTs with template — batch 1 (with properties) | 14,709 |
| `mintBatch` | 50 unique NFTs with template — batch 2 (no properties) | 13,823 |
| `mintBatch` | 100 unique NFTs (2×50) — total | 28,532 |
| `mintBatch` | 100 unique NFTs (2×50) — avg per batch | 14,266 |

### Transfers

| Function | Scenario | RC |
|----------|----------|---:|
| `safeTransferFrom` | 10 unique NFTs (owner) — total | 2,816 |
| `safeTransferFrom` | 10 unique NFTs (owner) — avg per call | 281 |
| `setApprovalForAll` | Approve operator (one-time) | 162 |
| `safeTransferFrom` | 10 unique NFTs (operator) — avg per call | 281 |
| `approve` | 10 unique NFTs — total | 1,780 |
| `approve` | 10 unique NFTs — avg per call | 178 |
| `safeTransferFrom` | 10 unique NFTs (allowance) — total | 4,080 |
| `safeTransferFrom` | 10 unique NFTs (allowance) — avg per call | 408 |
| `safeBatchTransferFrom` | 50 unique NFTs | 6,962 |
| `safeTransferFrom` | 1,000 editions | 348 |
| `safeTransferFrom` | 500 editions | 335 |

### Burns

| Function | Scenario | RC |
|----------|----------|---:|
| `burn` | 5 unique NFTs — total | 1,010 |
| `burn` | 5 unique NFTs — avg per call | 202 |
| `burn` | 100 editions | 240 |
| `burn` | 1,000 editions | 241 |
| `burnBatch` | 20 unique NFTs — total | 1,748 |
| `burnBatch` | 20 unique NFTs — avg per burn | 87 |