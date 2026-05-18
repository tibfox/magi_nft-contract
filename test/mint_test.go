package contract_test

import (
	"testing"
)

// ===================================
// Mint Tests
// ===================================

func TestMintSuccess(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","id":"1","amount":10,"maxSupply":100,"data":""}`)
	CallContract(t, ct, "mint", payload, nil, ownerAddress, true, uint(150_000_000), "")
}

func TestMintUniqueNFT(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	// maxSupply=1 means unique NFT
	payload := []byte(`{"to":"hive:tibfox","id":"unique-art-001","amount":1,"maxSupply":1,"data":""}`)
	CallContract(t, ct, "mint", payload, nil, ownerAddress, true, uint(150_000_000), "")

	// Can't mint more of a unique NFT
	payload2 := []byte(`{"to":"hive:tibfox","id":"unique-art-001","amount":1,"maxSupply":1,"data":""}`)
	CallContract(t, ct, "mint", payload2, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintEditionedNFT(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	// maxSupply=100 means 100 edition NFT
	payload := []byte(`{"to":"hive:tibfox","id":"edition-001","amount":50,"maxSupply":100,"data":""}`)
	CallContract(t, ct, "mint", payload, nil, ownerAddress, true, uint(150_000_000), "")

	// Can mint 50 more (total 100)
	payload2 := []byte(`{"to":"hive:other","id":"edition-001","amount":50,"maxSupply":100,"data":""}`)
	CallContract(t, ct, "mint", payload2, nil, ownerAddress, true, uint(150_000_000), "")

	// Can't mint more - would exceed max supply
	payload3 := []byte(`{"to":"hive:tibfox","id":"edition-001","amount":1,"maxSupply":100,"data":""}`)
	CallContract(t, ct, "mint", payload3, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintFailsIfNoMaxSupply(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	// Missing maxSupply should fail
	payload := []byte(`{"to":"hive:tibfox","id":"1","amount":10,"data":""}`)
	CallContract(t, ct, "mint", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintFailsIfMaxSupplyMismatch(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	// First mint sets maxSupply to 100
	payload := []byte(`{"to":"hive:tibfox","id":"1","amount":10,"maxSupply":100,"data":""}`)
	CallContract(t, ct, "mint", payload, nil, ownerAddress, true, uint(150_000_000), "")

	// Second mint with different maxSupply should fail
	payload2 := []byte(`{"to":"hive:tibfox","id":"1","amount":10,"maxSupply":200,"data":""}`)
	CallContract(t, ct, "mint", payload2, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintFailsIfNotOwner(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:other","id":"1","amount":10,"maxSupply":100,"data":""}`)
	CallContract(t, ct, "mint", payload, nil, "hive:other", false, uint(150_000_000), "")
}

func TestMintFailsIfNotInit(t *testing.T) {
	ct := SetupContractTest()
	payload := []byte(`{"to":"hive:tibfox","id":"1","amount":10,"maxSupply":100,"data":""}`)
	CallContract(t, ct, "mint", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintFailsIfPaused(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "pause", nil, nil, ownerAddress, true, uint(150_000_000), "")

	payload := []byte(`{"to":"hive:tibfox","id":"1","amount":10,"maxSupply":100,"data":""}`)
	CallContract(t, ct, "mint", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

// amount:0 is no longer an error — it defines an edition without minting
// (owner-only). See editioned_define_test.go for full define behavior.
func TestMintZeroAmountDefinesEdition(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"id":"1","amount":0,"maxSupply":100,"data":""}`)
	CallContract(t, ct, "mint", payload, nil, ownerAddress, true, uint(150_000_000), "")
	// Defined, but nothing minted yet.
	CallContract(t, ct, "totalSupply", []byte(`{"id":"1"}`), nil, ownerAddress, true, uint(150_000_000), `{"totalSupply":0}`)
}

func TestMintFailsWithZeroMaxSupply(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","id":"1","amount":10,"maxSupply":0,"data":""}`)
	CallContract(t, ct, "mint", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintFailsWithAmountGreaterThanMaxSupply(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","id":"1","amount":101,"maxSupply":100,"data":""}`)
	CallContract(t, ct, "mint", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintFailsWithEmptyRecipient(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"","id":"1","amount":10,"maxSupply":100,"data":""}`)
	CallContract(t, ct, "mint", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintFailsWithEmptyTokenId(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","id":"","amount":10,"maxSupply":100,"data":""}`)
	CallContract(t, ct, "mint", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintWithoutMaxSupplyOnExistingToken(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	// First mint with maxSupply
	payload := []byte(`{"to":"hive:tibfox","id":"edition-002","amount":50,"maxSupply":100,"data":""}`)
	CallContract(t, ct, "mint", payload, nil, ownerAddress, true, uint(150_000_000), "")

	// Second mint without maxSupply - should use existing
	payload2 := []byte(`{"to":"hive:other","id":"edition-002","amount":25,"data":""}`)
	CallContract(t, ct, "mint", payload2, nil, ownerAddress, true, uint(150_000_000), "")

	// Verify total supply is 75
	supplyPayload := []byte(`{"id":"edition-002"}`)
	result := CallContract(t, ct, "totalSupply", supplyPayload, nil, ownerAddress, true, uint(150_000_000), "")
	if result.Ret != `{"totalSupply":75}` {
		t.Errorf("Expected totalSupply 75, got %s", result.Ret)
	}
}

// ===================================
// MintBatch Tests
// ===================================

func TestMintBatchSuccess(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","ids":["1","2","3"],"amounts":[10,20,30],"maxSupplies":[100,200,300],"data":""}`)
	CallContract(t, ct, "mintBatch", payload, nil, ownerAddress, true, uint(150_000_000), "")
}

func TestMintBatchFailsIfNotOwner(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","ids":["1","2","3"],"amounts":[10,20,30],"maxSupplies":[100,200,300],"data":""}`)
	CallContract(t, ct, "mintBatch", payload, nil, "hive:other", false, uint(150_000_000), "")
}

func TestMintBatchFailsIfPaused(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "pause", nil, nil, ownerAddress, true, uint(150_000_000), "")

	payload := []byte(`{"to":"hive:tibfox","ids":["1","2","3"],"amounts":[10,20,30],"maxSupplies":[100,200,300],"data":""}`)
	CallContract(t, ct, "mintBatch", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintBatchFailsWithMismatchedLengths(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	// 3 ids, 2 amounts
	payload := []byte(`{"to":"hive:tibfox","ids":["1","2","3"],"amounts":[10,20],"maxSupplies":[100,200,300],"data":""}`)
	CallContract(t, ct, "mintBatch", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintBatchFailsWithEmptyArrays(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","ids":[],"amounts":[],"maxSupplies":[],"data":""}`)
	CallContract(t, ct, "mintBatch", payload, nil, ownerAddress, false, uint(150_000_000), "")
}
