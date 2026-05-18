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

// #1 End-to-end: owner defines an edition, approves a market, the market
// mints it to several buyers up to maxSupply, and the over-max mint aborts.
func TestE2EMarketDefineThenDelegatedMintToMax(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	// Owner defines the edition (no supply minted).
	CallContract(t, ct, "mint",
		[]byte(`{"id":"drop1","amount":0,"maxSupply":5,"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	// Owner approves the market as operator.
	CallContract(t, ct, "setApprovalForAll",
		[]byte(`{"operator":"hive:market","approved":true}`),
		nil, ownerAddress, true, uint(150_000_000), "")

	// Market mints to two different buyers, total == maxSupply.
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer1","id":"drop1","amount":3,"data":""}`),
		nil, "hive:market", true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer2","id":"drop1","amount":2,"data":""}`),
		nil, "hive:market", true, uint(150_000_000), "")

	CallContract(t, ct, "balanceOf",
		[]byte(`{"account":"hive:buyer1","id":"drop1"}`),
		nil, "hive:buyer1", true, uint(150_000_000), `{"balance":3}`)
	CallContract(t, ct, "balanceOf",
		[]byte(`{"account":"hive:buyer2","id":"drop1"}`),
		nil, "hive:buyer2", true, uint(150_000_000), `{"balance":2}`)
	CallContract(t, ct, "totalSupply",
		[]byte(`{"id":"drop1"}`),
		nil, ownerAddress, true, uint(150_000_000), `{"totalSupply":5}`)

	// One past maxSupply aborts.
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer1","id":"drop1","amount":1,"data":""}`),
		nil, "hive:market", false, uint(150_000_000), "")
}

// #1 End-to-end for a defined series: owner defines a series, market mints
// one of the series ids up to its per-id maxSupply, over-max aborts.
func TestE2EMarketDefineSeriesThenDelegatedMint(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	CallContract(t, ct, "mintSeries",
		[]byte(`{"idPrefix":"shop-","startNumber":1,"count":3,"amount":0,"maxSupply":2}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	// Every generated id exists with zero supply.
	CallContract(t, ct, "exists",
		[]byte(`{"id":"shop-1"}`), nil, ownerAddress, true, uint(150_000_000), `{"exists":true}`)
	CallContract(t, ct, "exists",
		[]byte(`{"id":"shop-3"}`), nil, ownerAddress, true, uint(150_000_000), `{"exists":true}`)
	CallContract(t, ct, "totalSupply",
		[]byte(`{"id":"shop-2"}`), nil, ownerAddress, true, uint(150_000_000), `{"totalSupply":0}`)

	CallContract(t, ct, "setApprovalForAll",
		[]byte(`{"operator":"hive:market","approved":true}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"shop-2","amount":2,"data":""}`),
		nil, "hive:market", true, uint(150_000_000), "")
	CallContract(t, ct, "balanceOf",
		[]byte(`{"account":"hive:buyer","id":"shop-2"}`),
		nil, "hive:buyer", true, uint(150_000_000), `{"balance":2}`)
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"shop-2","amount":1,"data":""}`),
		nil, "hive:market", false, uint(150_000_000), "")
}

// #2 Soulbound is carried from define: defined-soulbound tokens report
// soulbound and cannot be transferred after minting.
func TestDefineSoulboundEditionIsSoulboundAndNonTransferable(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	CallContract(t, ct, "mint",
		[]byte(`{"id":"sb1","amount":0,"maxSupply":3,"soulbound":true,"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "isSoulbound",
		[]byte(`{"id":"sb1"}`), nil, ownerAddress, true, uint(150_000_000), `{"soulbound":true}`)

	// Mint one to a holder, then a transfer must abort (soulbound).
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:holder","id":"sb1","amount":1,"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "safeTransferFrom",
		[]byte(`{"from":"hive:holder","to":"hive:other","id":"sb1","amount":1,"data":""}`),
		nil, "hive:holder", false, uint(150_000_000), "")
}

// #3 Properties are carried from define and persist through minting.
func TestDefinePropertiesPersistThroughMint(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	CallContract(t, ct, "mint",
		[]byte(`{"id":"p1","amount":0,"maxSupply":2,"properties":{"rarity":"gold"},"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	// Readable before any mint.
	CallContract(t, ct, "getProperties",
		[]byte(`{"id":"p1"}`), nil, ownerAddress, true, uint(150_000_000), `{"properties":{"rarity":"gold"}}`)
	// Still readable after minting.
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"p1","amount":1,"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "getProperties",
		[]byte(`{"id":"p1"}`), nil, ownerAddress, true, uint(150_000_000), `{"properties":{"rarity":"gold"}}`)
}

// #4 Define a unique (maxSupply == 1) edition; one mint succeeds, the next aborts.
func TestDefineUniqueEditionThenMintOnce(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	CallContract(t, ct, "mint",
		[]byte(`{"id":"u1","amount":0,"maxSupply":1,"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"u1","amount":1,"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"u1","amount":1,"data":""}`),
		nil, ownerAddress, false, uint(150_000_000), "")
}

// #5 A stranger (neither owner nor approved operator) cannot define an edition.
func TestDefineByStrangerAborts(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"id":"drop1","amount":0,"maxSupply":5,"data":""}`),
		nil, "hive:stranger", false, uint(150_000_000), "")
}

// #6 After a define, a mint that passes a mismatching maxSupply aborts.
func TestDefineThenMintMaxSupplyMismatchAborts(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"id":"m1","amount":0,"maxSupply":5,"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"m1","amount":2,"maxSupply":9,"data":""}`),
		nil, ownerAddress, false, uint(150_000_000), "")
}

// #7 mintSeries define-only with a propertiesTemplate: only the template id
// stores properties; copies report null; ids exist at zero supply and mint.
func TestDefineSeriesWithPropertiesTemplate(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	CallContract(t, ct, "mintSeries",
		[]byte(`{"idPrefix":"tpl-","startNumber":1,"count":3,"amount":0,"maxSupply":4,"properties":{"set":"alpha"},"propertiesTemplate":"tpl-1"}`),
		nil, ownerAddress, true, uint(200_000_000), "")

	CallContract(t, ct, "getProperties",
		[]byte(`{"id":"tpl-1"}`), nil, ownerAddress, true, uint(150_000_000), `{"properties":{"set":"alpha"}}`)
	CallContract(t, ct, "getProperties",
		[]byte(`{"id":"tpl-2"}`), nil, ownerAddress, true, uint(150_000_000), `{"properties":null}`)
	CallContract(t, ct, "exists",
		[]byte(`{"id":"tpl-2"}`), nil, ownerAddress, true, uint(150_000_000), `{"exists":true}`)
	CallContract(t, ct, "totalSupply",
		[]byte(`{"id":"tpl-2"}`), nil, ownerAddress, true, uint(150_000_000), `{"totalSupply":0}`)

	// Defined series id mints up to maxSupply, then aborts.
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"tpl-2","amount":4,"data":""}`),
		nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "mint",
		[]byte(`{"to":"hive:buyer","id":"tpl-2","amount":1,"data":""}`),
		nil, ownerAddress, false, uint(150_000_000), "")
}
