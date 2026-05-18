package contract_test

import (
	"testing"
)

// A per-token approve allowance authorizes minting and is decremented per mint,
// independent of setApprovalForAll (ERC-6909, mirrors safeTransferFrom).
func TestPerTokenAllowanceAuthorizesMintAndDecrements(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	// Owner defines an edition, then grants the market a capped allowance of 3.
	CallContract(t, ct, "mint",
		[]byte(`{"id":"drop1","amount":0,"maxSupply":10,"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "approve",
		[]byte(`{"spender":"hive:market","id":"drop1","amount":3}`),
		nil, ownerAddress, true, uint(150_000_000), "")

	// Market mints 2 within allowance.
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"drop1","amount":2,"data":""}`),
		nil, "hive:market", true, uint(150_000_000), "")
	CallContract(t, ct, "allowance",
		[]byte(`{"owner":"hive:tibfox","spender":"hive:market","id":"drop1"}`),
		nil, ownerAddress, true, uint(150_000_000), `{"amount":1}`)

	// Over remaining allowance aborts.
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"drop1","amount":2,"data":""}`),
		nil, "hive:market", false, uint(150_000_000), "")

	// Exactly the remaining 1 succeeds, exhausting the allowance.
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"drop1","amount":1,"data":""}`),
		nil, "hive:market", true, uint(150_000_000), "")
	CallContract(t, ct, "allowance",
		[]byte(`{"owner":"hive:tibfox","spender":"hive:market","id":"drop1"}`),
		nil, ownerAddress, true, uint(150_000_000), `{"amount":0}`)

	// Exhausted allowance aborts further mints.
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"drop1","amount":1,"data":""}`),
		nil, "hive:market", false, uint(150_000_000), "")

	CallContract(t, ct, "balanceOf",
		[]byte(`{"account":"hive:buyer","id":"drop1"}`),
		nil, "hive:buyer", true, uint(150_000_000), `{"balance":3}`)
}

// A per-token allowance only authorizes the exact token id it was granted for.
func TestPerTokenAllowanceScopedToId(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	CallContract(t, ct, "mint",
		[]byte(`{"id":"drop1","amount":0,"maxSupply":5,"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"id":"drop2","amount":0,"maxSupply":5,"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "approve",
		[]byte(`{"spender":"hive:market","id":"drop1","amount":5}`),
		nil, ownerAddress, true, uint(150_000_000), "")

	// Allowed on drop1.
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"drop1","amount":1,"data":""}`),
		nil, "hive:market", true, uint(150_000_000), "")
	// Not allowed on drop2 (no allowance, not an operator).
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"drop2","amount":1,"data":""}`),
		nil, "hive:market", false, uint(150_000_000), "")
}

// mintSeries via per-token allowance: allowance is checked and decremented
// per generated id (mirrors safeBatchTransferFrom).
func TestPerTokenAllowanceMintSeriesPerId(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	// Owner defines a 2-id series, grants the market allowance of 1 on each id.
	CallContract(t, ct, "mintSeries",
		[]byte(`{"idPrefix":"s-","startNumber":1,"count":2,"amount":0,"maxSupply":5}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "approve",
		[]byte(`{"spender":"hive:market","id":"s-1","amount":1}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "approve",
		[]byte(`{"spender":"hive:market","id":"s-2","amount":1}`),
		nil, ownerAddress, true, uint(150_000_000), "")

	// Market mints the whole series (1 each) — succeeds, allowances drop to 0.
	CallContract(t, ct, "mintSeries",
		[]byte(`{"to":"hive:buyer","idPrefix":"s-","startNumber":1,"count":2,"amount":1,"maxSupply":5}`),
		nil, "hive:market", true, uint(150_000_000), "")
	CallContract(t, ct, "balanceOf",
		[]byte(`{"account":"hive:buyer","id":"s-2"}`),
		nil, "hive:buyer", true, uint(150_000_000), `{"balance":1}`)

	// Second pass aborts — allowances exhausted.
	CallContract(t, ct, "mintSeries",
		[]byte(`{"to":"hive:buyer","idPrefix":"s-","startNumber":1,"count":2,"amount":1,"maxSupply":5}`),
		nil, "hive:market", false, uint(150_000_000), "")
}
