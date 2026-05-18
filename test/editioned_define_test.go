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
