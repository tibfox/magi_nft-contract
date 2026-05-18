package contract_test

import (
	"fmt"
	"testing"
)

// ===================================
// MintSeries Tests
// ===================================

func TestMintSeriesSuccess(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"card-","startNumber":1,"count":10,"amount":1,"maxSupply":1}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, true, uint(500_000_000), "")
}

func TestMintSeriesProducesCorrectIds(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"card-","startNumber":5,"count":3,"amount":1,"maxSupply":1}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, true, uint(150_000_000), "")

	// IDs should be card-5, card-6, card-7
	for _, id := range []string{"card-5", "card-6", "card-7"} {
		balPayload := fmt.Sprintf(`{"account":"hive:tibfox","id":"%s"}`, id)
		result := CallContract(t, ct, "balanceOf", []byte(balPayload), nil, ownerAddress, true, uint(150_000_000), "")
		if result.Ret != `{"balance":1}` {
			t.Errorf("Expected balance 1 for %s, got %s", id, result.Ret)
		}
	}

	// card-4 should not exist
	result := CallContract(t, ct, "balanceOf", []byte(`{"account":"hive:tibfox","id":"card-4"}`), nil, ownerAddress, true, uint(150_000_000), "")
	if result.Ret != `{"balance":0}` {
		t.Errorf("Expected balance 0 for card-4, got %s", result.Ret)
	}
}

func TestMintSeriesWithStartNumberZero(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"item-","startNumber":0,"count":5,"amount":1,"maxSupply":1}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, true, uint(150_000_000), "")

	result := CallContract(t, ct, "balanceOf", []byte(`{"account":"hive:tibfox","id":"item-0"}`), nil, ownerAddress, true, uint(150_000_000), "")
	if result.Ret != `{"balance":1}` {
		t.Errorf("Expected balance 1 for item-0, got %s", result.Ret)
	}
}

func TestMintSeriesWithSoulbound(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"sb-","startNumber":1,"count":3,"amount":1,"maxSupply":1,"soulbound":true}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, true, uint(150_000_000), "")

	result := CallContract(t, ct, "isSoulbound", []byte(`{"id":"sb-1"}`), nil, ownerAddress, true, uint(150_000_000), "")
	if result.Ret != `{"soulbound":true}` {
		t.Errorf("Expected soulbound true for sb-1, got %s", result.Ret)
	}
}

func TestMintSeriesWithProperties(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"gem-","startNumber":1,"count":2,"amount":1,"maxSupply":1,"properties":{"rarity":"epic"}}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, true, uint(150_000_000), "")

	result := CallContract(t, ct, "getProperties", []byte(`{"id":"gem-1"}`), nil, ownerAddress, true, uint(150_000_000), "")
	if result.Ret != `{"properties":{"rarity":"epic"}}` {
		t.Errorf("Expected properties for gem-1, got %s", result.Ret)
	}
}

func TestMintSeriesEditionedTokens(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	// maxSupply=10, amount=5 per token
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"edition-","startNumber":1,"count":3,"amount":5,"maxSupply":10}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, true, uint(150_000_000), "")

	result := CallContract(t, ct, "balanceOf", []byte(`{"account":"hive:tibfox","id":"edition-1"}`), nil, ownerAddress, true, uint(150_000_000), "")
	if result.Ret != `{"balance":5}` {
		t.Errorf("Expected balance 5 for edition-1, got %s", result.Ret)
	}
}

func TestMintSeriesWithSuffix(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"card-","idSuffix":"-rare","startNumber":1,"count":3,"amount":1,"maxSupply":1}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, true, uint(150_000_000), "")

	// IDs should be card-1-rare, card-2-rare, card-3-rare
	for _, id := range []string{"card-1-rare", "card-2-rare", "card-3-rare"} {
		balPayload := fmt.Sprintf(`{"account":"hive:tibfox","id":"%s"}`, id)
		result := CallContract(t, ct, "balanceOf", []byte(balPayload), nil, ownerAddress, true, uint(150_000_000), "")
		if result.Ret != `{"balance":1}` {
			t.Errorf("Expected balance 1 for %s, got %s", id, result.Ret)
		}
	}

	// card-1 (without suffix) should NOT exist
	result := CallContract(t, ct, "balanceOf", []byte(`{"account":"hive:tibfox","id":"card-1"}`), nil, ownerAddress, true, uint(150_000_000), "")
	if result.Ret != `{"balance":0}` {
		t.Errorf("Expected balance 0 for card-1 (no suffix), got %s", result.Ret)
	}
}

func TestMintSeriesWithSuffixAndTemplate(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	// Template must match full generated ID including suffix
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"card-","idSuffix":"-epic","startNumber":1,"count":5,"amount":1,"maxSupply":1,"properties":{"rarity":"epic"},"propertiesTemplate":"card-1-epic"}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, true, uint(200_000_000), "")

	// Template token should have properties
	result := CallContract(t, ct, "getProperties", []byte(`{"id":"card-1-epic"}`), nil, ownerAddress, true, uint(150_000_000), "")
	if result.Ret != `{"properties":{"rarity":"epic"}}` {
		t.Errorf("Expected properties on template card-1-epic, got %s", result.Ret)
	}

	// Non-template tokens should NOT have properties stored
	result = CallContract(t, ct, "getProperties", []byte(`{"id":"card-2-epic"}`), nil, ownerAddress, true, uint(150_000_000), "")
	if result.Ret != `{"properties":null}` {
		t.Errorf("Expected no stored properties for card-2-epic, got %s", result.Ret)
	}
}

func TestMintSeriesWithSuffixAndProperties(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"gem-","idSuffix":"-v2","startNumber":1,"count":2,"amount":1,"maxSupply":1,"properties":{"type":"diamond"}}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, true, uint(150_000_000), "")

	result := CallContract(t, ct, "getProperties", []byte(`{"id":"gem-1-v2"}`), nil, ownerAddress, true, uint(150_000_000), "")
	if result.Ret != `{"properties":{"type":"diamond"}}` {
		t.Errorf("Expected properties for gem-1-v2, got %s", result.Ret)
	}
}

func TestMintSeriesFailsWithPipeInSuffix(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"card-","idSuffix":"-bad|suffix","startNumber":1,"count":5,"amount":1,"maxSupply":1}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintSeriesWithEmptySuffix(t *testing.T) {
	// Empty suffix should work the same as no suffix (backward compatible)
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"card-","idSuffix":"","startNumber":1,"count":3,"amount":1,"maxSupply":1}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, true, uint(150_000_000), "")

	// IDs should be card-1, card-2, card-3 (no suffix appended)
	for _, id := range []string{"card-1", "card-2", "card-3"} {
		balPayload := fmt.Sprintf(`{"account":"hive:tibfox","id":"%s"}`, id)
		result := CallContract(t, ct, "balanceOf", []byte(balPayload), nil, ownerAddress, true, uint(150_000_000), "")
		if result.Ret != `{"balance":1}` {
			t.Errorf("Expected balance 1 for %s, got %s", id, result.Ret)
		}
	}
}

func TestMintSeriesFailsIfNotOwner(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:other","idPrefix":"card-","startNumber":1,"count":5,"amount":1,"maxSupply":1}`)
	CallContract(t, ct, "mintSeries", payload, nil, "hive:other", false, uint(150_000_000), "")
}

func TestMintSeriesFailsIfPaused(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "pause", nil, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"card-","startNumber":1,"count":5,"amount":1,"maxSupply":1}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintSeriesFailsWithZeroCount(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"card-","startNumber":1,"count":0,"amount":1,"maxSupply":1}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, false, uint(150_000_000), "")
}


// amount:0 is no longer an error — it defines the series without minting
// (owner-only). See editioned_define_test.go for full define behavior.
func TestMintSeriesZeroAmountDefinesSeries(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"idPrefix":"card-","startNumber":1,"count":5,"amount":0,"maxSupply":1}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, true, uint(150_000_000), "")
	CallContract(t, ct, "totalSupply", []byte(`{"id":"card-3"}`), nil, ownerAddress, true, uint(150_000_000), `{"totalSupply":0}`)
}

func TestMintSeriesFailsWithZeroMaxSupply(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"card-","startNumber":1,"count":5,"amount":1,"maxSupply":0}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintSeriesFailsWithEmptyTo(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"","idPrefix":"card-","startNumber":1,"count":5,"amount":1,"maxSupply":1}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintSeriesFailsWithPipeInPrefix(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"bad|prefix-","startNumber":1,"count":5,"amount":1,"maxSupply":1}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintSeriesFailsWouldExceedMaxSupply(t *testing.T) {
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	// maxSupply=1 but amount=2
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"card-","startNumber":1,"count":5,"amount":2,"maxSupply":1}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintSeriesFailsIfNotInit(t *testing.T) {
	ct := SetupContractTest()
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"card-","startNumber":1,"count":5,"amount":1,"maxSupply":1}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

// ===================================
// MintSeries PropertiesTemplate Tests
// ===================================

func TestMintSeriesTemplateMiddleId(t *testing.T) {
	// Template ID is card-3 (not the first generated ID card-1)
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"card-","startNumber":1,"count":5,"amount":1,"maxSupply":1,"properties":{"rarity":"legendary"},"propertiesTemplate":"card-3"}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, true, uint(200_000_000), "")

	// Template token should have properties
	result := CallContract(t, ct, "getProperties", []byte(`{"id":"card-3"}`), nil, ownerAddress, true, uint(150_000_000), "")
	if result.Ret != `{"properties":{"rarity":"legendary"}}` {
		t.Errorf("Expected properties on template card-3, got %s", result.Ret)
	}

	// Non-template tokens should NOT have properties stored (they inherit via event)
	result = CallContract(t, ct, "getProperties", []byte(`{"id":"card-1"}`), nil, ownerAddress, true, uint(150_000_000), "")
	if result.Ret != `{"properties":null}` {
		t.Errorf("Expected no stored properties for card-1, got %s", result.Ret)
	}
}

func TestMintSeriesTemplateExternalNFT(t *testing.T) {
	// Template is an already-minted NFT, not in the generated range
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	// Mint template NFT first
	CallContract(t, ct, "mint", []byte(`{"to":"hive:tibfox","id":"template-master","amount":1,"maxSupply":1,"properties":{"rarity":"epic"},"data":""}`), nil, ownerAddress, true, uint(150_000_000), "")

	// Now mintSeries referencing external template
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"copy-","startNumber":1,"count":5,"amount":1,"maxSupply":1,"propertiesTemplate":"template-master"}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, true, uint(150_000_000), "")

	// All generated tokens should exist
	for _, id := range []string{"copy-1", "copy-2", "copy-3", "copy-4", "copy-5"} {
		balPayload := fmt.Sprintf(`{"account":"hive:tibfox","id":"%s"}`, id)
		result := CallContract(t, ct, "balanceOf", []byte(balPayload), nil, ownerAddress, true, uint(150_000_000), "")
		if result.Ret != `{"balance":1}` {
			t.Errorf("Expected balance 1 for %s, got %s", id, result.Ret)
		}
	}
}

func TestMintSeriesTemplateExternalNFTWithPropertiesFails(t *testing.T) {
	// External template + properties provided → should fail (cannot overwrite external template properties)
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")

	// Mint template NFT
	CallContract(t, ct, "mint", []byte(`{"to":"hive:tibfox","id":"base-card","amount":1,"maxSupply":1,"data":""}`), nil, ownerAddress, true, uint(150_000_000), "")

	// mintSeries referencing external template AND setting properties — should fail
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"derived-","startNumber":1,"count":3,"amount":1,"maxSupply":1,"properties":{"type":"warrior","power":99},"propertiesTemplate":"base-card"}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, false, uint(200_000_000), "")
}

func TestMintSeriesTemplateFailsNonExistentExternal(t *testing.T) {
	// Template ID not in generated range AND not an existing NFT → should fail
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"card-","startNumber":1,"count":5,"amount":1,"maxSupply":1,"properties":{"rarity":"epic"},"propertiesTemplate":"nonexistent-token"}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, false, uint(150_000_000), "")
}

func TestMintSeriesTemplateNoProperties(t *testing.T) {
	// propertiesTemplate set but no properties field — should still work (just emits template event, no props)
	ct := SetupContractTest()
	CallContract(t, ct, "init", DefaultInitPayload, nil, ownerAddress, true, uint(150_000_000), "")
	payload := []byte(`{"to":"hive:tibfox","idPrefix":"card-","startNumber":1,"count":3,"amount":1,"maxSupply":1,"propertiesTemplate":"card-1"}`)
	CallContract(t, ct, "mintSeries", payload, nil, ownerAddress, true, uint(150_000_000), "")

	// All tokens should be minted
	result := CallContract(t, ct, "balanceOf", []byte(`{"account":"hive:tibfox","id":"card-2"}`), nil, ownerAddress, true, uint(150_000_000), "")
	if result.Ret != `{"balance":1}` {
		t.Errorf("Expected balance 1 for card-2, got %s", result.Ret)
	}
}
