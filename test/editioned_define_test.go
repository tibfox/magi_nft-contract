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
