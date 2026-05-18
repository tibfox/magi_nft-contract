package main

import (
	"magi_nft/sdk"

	"github.com/CosmWasm/tinyjson/jlexer"
)

// ===================================
// MAGI NFT - ERC-1155 Exported WASM Functions
// ===================================

// ===================================
// Initialization
// ===================================

// Init initializes the NFT contract.
// Can only be called once by the contract owner (deployment account).
// Payload: {"name": "Collection Name", "symbol": "NFT", "baseUri": "https://api.example.com/metadata/"}
//
//go:wasmexport init
func Init(payload *string) *string {
	if isInit() {
		sdk.Abort("Already initialized")
	}

	// Only contract owner can initialize
	owner := sdk.GetEnvKey("contract.owner")
	if owner == nil {
		sdk.Abort("Contract owner not set")
	}
	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	if *caller != *owner {
		sdk.Abort("Only contract owner can initialize")
	}

	// Parse payload
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}
	var p InitPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	// Validate payload
	if p.Name == "" {
		sdk.Abort("Name required")
	}
	if len(p.Name) > maxNameLen {
		sdk.Abort("Name exceeds maximum length")
	}
	if p.Symbol == "" {
		sdk.Abort("Symbol required")
	}
	if len(p.Symbol) > maxSymbolLen {
		sdk.Abort("Symbol exceeds maximum length")
	}
	validateBaseURI(p.BaseURI)

	// Store contract properties
	sdk.StateSetObject("contract_name", p.Name)
	sdk.StateSetObject("contract_symbol", p.Symbol)
	sdk.StateSetObject("base_uri", p.BaseURI)
	if p.TrackMinted {
		sdk.StateSetObject("track_minted", "1")
	}
	if p.Metadata != "" {
		setCollectionMetadata(p.Metadata)
	}

	// Initialize contract state
	sdk.StateSetObject("isInit", "1")
	sdk.StateSetObject("owner", *owner)

	emitInit(*owner, p.Name, p.Symbol, p.BaseURI)
	return jsonResponse(SuccessResponse{Success: true})
}

// ===================================
// ERC-1155 Core Transfer Functions
// ===================================

// SafeTransferFrom transfers tokens from one address to another.
// Caller must be owner or approved operator.
// Payload: {"from": "hive:owner", "to": "hive:recipient", "id": "1", "amount": 1, "data": ""}
//
//go:wasmexport safeTransferFrom
func SafeTransferFrom(payload *string) *string {
	assertInit()
	assertNotPaused()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p SafeTransferFromPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.From == "" {
		sdk.Abort("From address required")
	}
	validateAddress(p.From)
	if p.To == "" {
		sdk.Abort("To address required")
	}
	validateAddress(p.To)
	if p.Id == "" {
		sdk.Abort("Token ID required")
	}
	validateTokenId(p.Id)
	if p.Amount == 0 {
		sdk.Abort("Amount must be greater than 0")
	}
	if p.From == p.To {
		sdk.Abort("Cannot transfer to self")
	}

	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	operator := *caller

	// Check authorization: owner, operator (blanket), or per-token allowance
	if !isApprovedOrOwner(operator, p.From) {
		// Check per-token allowance (ERC-6909)
		allowed := getAllowance(p.From, operator, p.Id)
		if allowed < p.Amount {
			sdk.Abort("Not authorized")
		}
		// Decrement allowance
		setAllowance(p.From, operator, p.Id, allowed-p.Amount)
	}

	// Check soulbound - contract owner can transfer, recipients cannot
	ownerAddr := getOwnerAddress()
	if isSoulbound(p.Id) && p.From != ownerAddr {
		sdk.Abort("Token is soulbound")
	}

	// Transfer
	decBalance(p.From, p.Id, p.Amount)
	incBalance(p.To, p.Id, p.Amount)
	emitTransferSingle(operator, p.From, p.To, p.Id, p.Amount)

	return jsonResponse(SuccessResponse{Success: true})
}

// SafeBatchTransferFrom transfers multiple token types from one address to another.
// Caller must be owner or approved operator.
// Payload: {"from": "hive:owner", "to": "hive:recipient", "ids": ["1", "2"], "amounts": [1, 5], "data": ""}
//
//go:wasmexport safeBatchTransferFrom
func SafeBatchTransferFrom(payload *string) *string {
	assertInit()
	assertNotPaused()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p SafeBatchTransferFromPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.From == "" {
		sdk.Abort("From address required")
	}
	validateAddress(p.From)
	if p.To == "" {
		sdk.Abort("To address required")
	}
	validateAddress(p.To)
	if len(p.Ids) == 0 {
		sdk.Abort("Token IDs required")
	}
	if len(p.Ids) != len(p.Amounts) {
		sdk.Abort("IDs and amounts length mismatch")
	}
	if p.From == p.To {
		sdk.Abort("Cannot transfer to self")
	}
	for _, id := range p.Ids {
		validateTokenId(id)
	}

	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	operator := *caller

	// Check authorization: owner, operator (blanket), or per-token allowance
	useAllowance := !isApprovedOrOwner(operator, p.From)

	// Check soulbound and transfer each token type
	ownerAddr := getOwnerAddress()
	for i := 0; i < len(p.Ids); i++ {
		if p.Amounts[i] == 0 {
			sdk.Abort("Amount must be greater than 0")
		}
		if useAllowance {
			allowed := getAllowance(p.From, operator, p.Ids[i])
			if allowed < p.Amounts[i] {
				sdk.Abort("Not authorized")
			}
			setAllowance(p.From, operator, p.Ids[i], allowed-p.Amounts[i])
		}
		if isSoulbound(p.Ids[i]) && p.From != ownerAddr {
			sdk.Abort("Token is soulbound")
		}
		decBalance(p.From, p.Ids[i], p.Amounts[i])
		incBalance(p.To, p.Ids[i], p.Amounts[i])
	}
	emitTransferBatch(operator, p.From, p.To, p.Ids, p.Amounts)

	return jsonResponse(SuccessResponse{Success: true})
}

// ===================================
// ERC-1155 Approval Functions
// ===================================

// SetApprovalForAll approves or revokes an operator for all tokens.
// Payload: {"operator": "hive:operator", "approved": true}
//
//go:wasmexport setApprovalForAll
func SetApprovalForAll(payload *string) *string {
	assertInit()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p SetApprovalForAllPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Operator == "" {
		sdk.Abort("Operator required")
	}
	validateAddress(p.Operator)

	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	account := *caller

	if account == p.Operator {
		sdk.Abort("Cannot approve self")
	}

	setApprovalForAllInternal(account, p.Operator, p.Approved)
	emitApprovalForAll(account, p.Operator, p.Approved)
	return jsonResponse(SuccessResponse{Success: true})
}

// ===================================
// ERC-6909 Per-Token Approval Functions
// ===================================

// Approve sets a per-token allowance for a spender (ERC-6909 pattern).
// Payload: {"spender": "hive:marketplace", "id": "card-1", "amount": 1}
//
//go:wasmexport approve
func Approve(payload *string) *string {
	assertInit()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p ApprovePayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Spender == "" {
		sdk.Abort("Spender required")
	}
	validateAddress(p.Spender)
	if p.Id == "" {
		sdk.Abort("Token ID required")
	}
	validateTokenId(p.Id)

	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	account := *caller

	if account == p.Spender {
		sdk.Abort("Cannot approve self")
	}

	setAllowance(account, p.Spender, p.Id, p.Amount)
	emitApproval(account, p.Spender, p.Id, p.Amount)
	return jsonResponse(SuccessResponse{Success: true})
}

// Allowance returns the per-token allowance for a spender (ERC-6909 pattern).
// Payload: {"owner": "hive:tibfox", "spender": "hive:marketplace", "id": "card-1"}
//
//go:wasmexport allowance
func Allowance(payload *string) *string {
	assertInit()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p AllowancePayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Owner == "" {
		sdk.Abort("Owner required")
	}
	if p.Spender == "" {
		sdk.Abort("Spender required")
	}
	if p.Id == "" {
		sdk.Abort("Token ID required")
	}

	amount := getAllowance(p.Owner, p.Spender, p.Id)
	return jsonResponse(AllowanceResponse{Amount: amount})
}

// ===================================
// Mint Functions (Owner Only)
// ===================================

// Mint creates new tokens and assigns them to an address.
// Payload: {"to": "hive:recipient", "id": "1", "amount": 1, "maxSupply": 100, "properties": {"color": "red"}, "data": ""}
// maxSupply is required on first mint (1 = unique, >1 = editioned), optional on subsequent mints.
// properties is optional arbitrary JSON, set on first mint only.
// Only the contract owner can mint.
//
//go:wasmexport mint
func Mint(payload *string) *string {
	assertInit()
	assertNotPaused()
	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	operator := *caller
	ownerAddr := getOwnerAddress()
	if !isApprovedOrOwner(operator, ownerAddr) {
		sdk.Abort("Must be owner or approved operator to mint")
	}
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p MintPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Amount != 0 {
		if p.To == "" {
			sdk.Abort("To address required")
		}
		validateAddress(p.To)
	}
	if p.Id == "" {
		sdk.Abort("Token ID required")
	}
	validateTokenId(p.Id)

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
	var maxSupply uint64
	if existingMax == 0 {
		// First mint - maxSupply is required
		if p.MaxSupply == 0 {
			sdk.Abort("MaxSupply required for new token (1 = unique, >1 = editioned)")
		}
		setMaxSupply(p.Id, p.MaxSupply)
		maxSupply = p.MaxSupply
		// Set soulbound on first mint if requested
		if p.Soulbound {
			setSoulbound(p.Id)
		}
		// Set properties on first mint if provided
		if p.Properties != "" {
			setTokenProperties(p.Id, p.Properties)
			emitPropertiesSet(p.Id)
		}
	} else {
		// Subsequent mint - use existing maxSupply, but validate if provided
		if p.MaxSupply != 0 && p.MaxSupply != existingMax {
			sdk.Abort("MaxSupply mismatch with existing token")
		}
		maxSupply = existingMax
	}

	// Check supply limits
	if isTrackMintedEnabled() {
		// Use totalMinted for supply check (burned tokens cannot be re-minted)
		currentMinted := getTotalMinted(p.Id)
		newMinted := safeAdd(currentMinted, p.Amount)
		if newMinted > maxSupply {
			sdk.Abort("Would exceed max supply")
		}
		incTotalMinted(p.Id, p.Amount)
	} else {
		// Use totalSupply for supply check (burned tokens can be re-minted)
		currentTotal := getTotalSupply(p.Id)
		newTotal := safeAdd(currentTotal, p.Amount)
		if newTotal > maxSupply {
			sdk.Abort("Would exceed max supply")
		}
	}

	// Emit tokenCreated on first mint
	if existingMax == 0 {
		emitTokenCreated(p.Id, maxSupply, p.Soulbound)
	}

	incBalance(p.To, p.Id, p.Amount)
	incTotalSupply(p.Id, p.Amount)
	emitTransferSingle(operator, "", p.To, p.Id, p.Amount) // Mint: from is zero address, operator is caller
	return jsonResponse(SuccessResponse{Success: true})
}

// MintBatch creates multiple token types and assigns them to an address.
// Payload: {"to": "hive:recipient", "ids": ["1", "2"], "amounts": [1, 5], "maxSupplies": [1, 100], "properties": [{"color": "red"}, {"size": 42}], "data": ""}
// maxSupplies required on first mint per token (1 = unique, >1 = editioned), optional/omittable for existing tokens.
// properties is optional per-token arbitrary JSON array, set on first mint only.
// Only the contract owner can mint.
//
//go:wasmexport mintBatch
func MintBatch(payload *string) *string {
	assertInit()
	assertNotPaused()
	owner, isOwner := getOwner()
	if !isOwner {
		sdk.Abort("Must be owner to mint")
	}
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p MintBatchPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.To == "" {
		sdk.Abort("To address required")
	}
	validateAddress(p.To)
	if len(p.Ids) == 0 {
		sdk.Abort("Token IDs required")
	}
	if len(p.Ids) != len(p.Amounts) {
		sdk.Abort("IDs and amounts length mismatch")
	}
	if len(p.MaxSupplies) > 0 && len(p.Ids) != len(p.MaxSupplies) {
		sdk.Abort("IDs and maxSupplies length mismatch")
	}
	for _, id := range p.Ids {
		validateTokenId(id)
	}

	// Validate propertiesTemplate before minting
	if p.PropertiesTemplate != "" && len(p.Ids) > 1 {
		found := false
		for _, id := range p.Ids {
			if id == p.PropertiesTemplate {
				found = true
				break
			}
		}
		if !found {
			if getMaxSupply(p.PropertiesTemplate) == 0 {
				sdk.Abort("propertiesTemplate must be one of the batch IDs or an existing NFT")
			}
		}
	}

	for i := 0; i < len(p.Ids); i++ {
		if p.Amounts[i] == 0 {
			sdk.Abort("Amount must be greater than 0")
		}

		// Get maxSupply from payload (0 if not provided)
		var payloadMax uint64
		if i < len(p.MaxSupplies) {
			payloadMax = p.MaxSupplies[i]
		}

		// Get soulbound from payload (false if not provided)
		var payloadSoulbound bool
		if i < len(p.Soulbound) {
			payloadSoulbound = p.Soulbound[i]
		}

		// Get properties from payload (empty if not provided)
		var payloadProperties string
		if i < len(p.Properties) {
			payloadProperties = p.Properties[i]
		}

		// Check/set max supply for this token
		existingMax := getMaxSupply(p.Ids[i])
		var maxSupply uint64
		if existingMax == 0 {
			// First mint - maxSupply is required
			if payloadMax == 0 {
				sdk.Abort("MaxSupply required for new token")
			}
			setMaxSupply(p.Ids[i], payloadMax)
			maxSupply = payloadMax
			// Set soulbound on first mint if requested
			if payloadSoulbound {
				setSoulbound(p.Ids[i])
			}
			// Set properties on first mint if provided
			if payloadProperties != "" {
				setTokenProperties(p.Ids[i], payloadProperties)
				emitPropertiesSet(p.Ids[i])
			}
		} else {
			// Subsequent mint - use existing maxSupply, but validate if provided
			if payloadMax != 0 && payloadMax != existingMax {
				sdk.Abort("MaxSupply mismatch with existing token")
			}
			maxSupply = existingMax
		}

		// Check supply limits
		if isTrackMintedEnabled() {
			// Use totalMinted for supply check (burned tokens cannot be re-minted)
			currentMinted := getTotalMinted(p.Ids[i])
			newMinted := safeAdd(currentMinted, p.Amounts[i])
			if newMinted > maxSupply {
				sdk.Abort("Would exceed max supply")
			}
			incTotalMinted(p.Ids[i], p.Amounts[i])
		} else {
			// Use totalSupply for supply check (burned tokens can be re-minted)
			currentTotal := getTotalSupply(p.Ids[i])
			newTotal := safeAdd(currentTotal, p.Amounts[i])
			if newTotal > maxSupply {
				sdk.Abort("Would exceed max supply")
			}
		}

		// Emit tokenCreated on first mint
		if existingMax == 0 {
			emitTokenCreated(p.Ids[i], maxSupply, payloadSoulbound)
		}

		incBalance(p.To, p.Ids[i], p.Amounts[i])
		incTotalSupply(p.Ids[i], p.Amounts[i])
	}
	emitTransferBatch(owner, "", p.To, p.Ids, p.Amounts) // Mint: from is zero address

	// Emit template relationship if propertiesTemplate is set
	if p.PropertiesTemplate != "" && len(p.Ids) > 1 {
		copyIds := make([]string, 0, len(p.Ids)-1)
		for _, id := range p.Ids {
			if id != p.PropertiesTemplate {
				copyIds = append(copyIds, id)
			}
		}
		emitTemplateMint(p.PropertiesTemplate, copyIds)
	}

	return jsonResponse(SuccessResponse{Success: true})
}

// MintSeries creates a series of token IDs with a shared configuration.
// Token IDs are generated as: idPrefix + (startNumber + i) + idSuffix for i in [0, count).
// All tokens share the same amount, maxSupply, soulbound, and properties settings.
// This avoids the large payload size of mintBatch when minting many identical tokens.
// Payload: {"to": "hive:recipient", "idPrefix": "card-", "idSuffix": "-rare", "startNumber": 1, "count": 100, "amount": 1, "maxSupply": 1, "soulbound": false}
// Only the contract owner can mint.
//
//go:wasmexport mintSeries
func MintSeries(payload *string) *string {
	assertInit()
	assertNotPaused()
	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	operator := *caller
	ownerAddr := getOwnerAddress()
	if !isApprovedOrOwner(operator, ownerAddr) {
		sdk.Abort("Must be owner or approved operator to mint")
	}
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p MintSeriesPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Amount != 0 {
		if p.To == "" {
			sdk.Abort("To address required")
		}
		validateAddress(p.To)
	}
	if p.Count == 0 {
		sdk.Abort("Count must be greater than 0")
	}
	defineOnly := p.Amount == 0
	if defineOnly && operator != ownerAddr {
		sdk.Abort("Only owner can define an edition")
	}
	if p.MaxSupply == 0 {
		sdk.Abort("MaxSupply required (1 = unique, >1 = editioned)")
	}
	// Check startNumber + count doesn't overflow uint64
	if p.StartNumber+p.Count < p.StartNumber {
		sdk.Abort("StartNumber + Count overflows")
	}
	// Validate prefix and suffix for pipe characters (generated IDs use prefix + number + suffix)
	for i := 0; i < len(p.IdPrefix); i++ {
		if p.IdPrefix[i] == '|' {
			sdk.Abort("Invalid character in ID prefix")
		}
	}
	for i := 0; i < len(p.IdSuffix); i++ {
		if p.IdSuffix[i] == '|' {
			sdk.Abort("Invalid character in ID suffix")
		}
	}
	// Validate propertiesTemplate: must be within the generated range OR already exist as an NFT
	if p.PropertiesTemplate != "" {
		found := false
		for j := uint64(0); j < p.Count; j++ {
			if p.IdPrefix+uint64ToStr(p.StartNumber+j)+p.IdSuffix == p.PropertiesTemplate {
				found = true
				break
			}
		}
		if !found {
			// Check if it already exists as a minted NFT
			if getMaxSupply(p.PropertiesTemplate) == 0 {
				sdk.Abort("propertiesTemplate must be one of the generated IDs or an existing NFT")
			}
			// Cannot set properties on an existing external template
			if p.Properties != "" {
				sdk.Abort("Cannot set properties when using an existing NFT as template")
			}
		}
	}

	ids := make([]string, p.Count)
	amounts := make([]uint64, p.Count)

	// Cache loop-invariant state reads
	trackMinted := isTrackMintedEnabled()
	setProps := p.Properties != "" && p.PropertiesTemplate == ""
	setTemplateProps := p.Properties != "" && p.PropertiesTemplate != ""

	for i := uint64(0); i < p.Count; i++ {
		id := p.IdPrefix + uint64ToStr(p.StartNumber+i) + p.IdSuffix
		// Skip full validateTokenId — prefix and suffix already checked for '|' and digits can't contain '|'.
		// Only need to verify length.
		if len(id) > maxTokenIdLen {
			sdk.Abort("Token ID exceeds maximum length")
		}
		ids[i] = id
		amounts[i] = p.Amount

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
			// Subsequent mint — need to read existing state
			if p.MaxSupply != existingMax {
				sdk.Abort("MaxSupply mismatch with existing token")
			}
			if trackMinted {
				currentMinted := getTotalMinted(id)
				newMinted := safeAdd(currentMinted, p.Amount)
				if newMinted > existingMax {
					sdk.Abort("Would exceed max supply")
				}
				incTotalMinted(id, p.Amount)
			} else {
				currentTotal := getTotalSupply(id)
				newTotal := safeAdd(currentTotal, p.Amount)
				if newTotal > existingMax {
					sdk.Abort("Would exceed max supply")
				}
			}
			incBalance(p.To, id, p.Amount)
			incTotalSupply(id, p.Amount)
		}
	}
	if !defineOnly {
		emitTransferBatch(operator, "", p.To, ids, amounts) // Mint: from is zero address, operator is caller
	}

	// Emit template relationship if propertiesTemplate is set
	if p.PropertiesTemplate != "" {
		templateId := p.PropertiesTemplate
		// All generated IDs that aren't the template are copies
		copyIds := make([]string, 0, len(ids))
		for _, id := range ids {
			if id != templateId {
				copyIds = append(copyIds, id)
			}
		}
		if len(copyIds) > 0 {
			emitTemplateMint(templateId, copyIds)
		}
	}

	return jsonResponse(SuccessResponse{Success: true})
}

// ===================================
// Burn Functions
// ===================================

// Burn destroys tokens from an account.
// Caller must be owner or approved operator.
// Payload: {"from": "hive:owner", "id": "1", "amount": 1}
//
//go:wasmexport burn
func Burn(payload *string) *string {
	assertInit()
	assertNotPaused()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p BurnPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.From == "" {
		sdk.Abort("From address required")
	}
	validateAddress(p.From)
	if p.Id == "" {
		sdk.Abort("Token ID required")
	}
	validateTokenId(p.Id)
	if p.Amount == 0 {
		sdk.Abort("Amount must be greater than 0")
	}

	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	operator := *caller

	// Check authorization: owner, operator (blanket), or per-token allowance
	if !isApprovedOrOwner(operator, p.From) {
		// Check per-token allowance (ERC-6909)
		allowed := getAllowance(p.From, operator, p.Id)
		if allowed < p.Amount {
			sdk.Abort("Not authorized")
		}
		// Decrement allowance
		setAllowance(p.From, operator, p.Id, allowed-p.Amount)
	}

	decBalance(p.From, p.Id, p.Amount)
	decTotalSupply(p.Id, p.Amount)
	emitTransferSingle(operator, p.From, "", p.Id, p.Amount) // Burn: to is zero address
	return jsonResponse(SuccessResponse{Success: true})
}

// BurnBatch destroys multiple token types from an account.
// Caller must be owner or approved operator.
// Payload: {"from": "hive:owner", "ids": ["1", "2"], "amounts": [1, 5]}
//
//go:wasmexport burnBatch
func BurnBatch(payload *string) *string {
	assertInit()
	assertNotPaused()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p BurnBatchPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.From == "" {
		sdk.Abort("From address required")
	}
	validateAddress(p.From)
	if len(p.Ids) == 0 {
		sdk.Abort("Token IDs required")
	}
	if len(p.Ids) != len(p.Amounts) {
		sdk.Abort("IDs and amounts length mismatch")
	}
	for _, id := range p.Ids {
		validateTokenId(id)
	}

	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	operator := *caller

	// Check authorization: owner, operator (blanket), or per-token allowance
	useAllowance := !isApprovedOrOwner(operator, p.From)

	for i := 0; i < len(p.Ids); i++ {
		if p.Amounts[i] == 0 {
			sdk.Abort("Amount must be greater than 0")
		}
		if useAllowance {
			allowed := getAllowance(p.From, operator, p.Ids[i])
			if allowed < p.Amounts[i] {
				sdk.Abort("Not authorized")
			}
			setAllowance(p.From, operator, p.Ids[i], allowed-p.Amounts[i])
		}
		decBalance(p.From, p.Ids[i], p.Amounts[i])
		decTotalSupply(p.Ids[i], p.Amounts[i])
	}
	emitTransferBatch(operator, p.From, "", p.Ids, p.Amounts) // Burn: to is zero address
	return jsonResponse(SuccessResponse{Success: true})
}

// ===================================
// URI Management
// ===================================

// SetURI sets the URI for a specific token ID.
// Payload: {"id": "1", "uri": "https://example.com/metadata/1.json"}
// Only the contract owner can set URIs.
//
//go:wasmexport setURI
func SetURI(payload *string) *string {
	assertInit()
	_, isOwner := getOwner()
	if !isOwner {
		sdk.Abort("Must be owner to set URI")
	}
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p SetURIPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Id == "" {
		sdk.Abort("Token ID required")
	}
	validateTokenId(p.Id)
	validateURI(p.Uri)

	setTokenURI(p.Id, p.Uri)
	emitURI(p.Uri, p.Id)
	return jsonResponse(SuccessResponse{Success: true})
}

// SetBaseURI updates the base URI for all tokens.
// Payload: {"baseUri": "https://newapi.example.com/metadata/"}
// Only the contract owner can set the base URI.
//
//go:wasmexport setBaseURI
func SetBaseURI(payload *string) *string {
	assertInit()
	_, isOwner := getOwner()
	if !isOwner {
		sdk.Abort("Must be owner to set base URI")
	}
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p SetBaseURIPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	validateBaseURI(p.BaseURI)
	previousURI := getBaseURI()
	sdk.StateSetObject("base_uri", p.BaseURI)
	emitBaseURIChange(previousURI, p.BaseURI)
	return jsonResponse(SuccessResponse{Success: true})
}

// ===================================
// Contract Management Actions
// ===================================

// ChangeOwner transfers contract ownership to a new address.
// Payload: {"newOwner": "hive:newowner"}
//
//go:wasmexport changeOwner
func ChangeOwner(payload *string) *string {
	assertInit()
	previousOwner, isOwner := getOwner()
	if !isOwner {
		sdk.Abort("Not owner")
	}
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p ChangeOwnerPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.NewOwner == "" {
		sdk.Abort("New owner required")
	}
	validateAddress(p.NewOwner)
	if p.NewOwner == previousOwner {
		sdk.Abort("Already the owner")
	}

	sdk.StateSetObject("owner", p.NewOwner)
	emitOwnerChange(previousOwner, p.NewOwner)
	return jsonResponse(SuccessResponse{Success: true})
}

// Pause pauses all token transfers. Only owner can pause.
//
//go:wasmexport pause
func Pause(_ *string) *string {
	assertInit()
	owner, isOwner := getOwner()
	if !isOwner {
		sdk.Abort("Not owner")
	}
	if isPaused() {
		sdk.Abort("Already paused")
	}
	sdk.StateSetObject("paused", "1")
	emitPaused(owner)
	return jsonResponse(SuccessResponse{Success: true})
}

// Unpause unpauses all token transfers. Only owner can unpause.
//
//go:wasmexport unpause
func Unpause(_ *string) *string {
	assertInit()
	owner, isOwner := getOwner()
	if !isOwner {
		sdk.Abort("Not owner")
	}
	if !isPaused() {
		sdk.Abort("Not paused")
	}
	sdk.StateDeleteObject("paused")
	emitUnpaused(owner)
	return jsonResponse(SuccessResponse{Success: true})
}

// ===================================
// ERC-1155 Read-Only Queries
// ===================================

// BalanceOf returns the token balance of an address for a specific token ID.
// Payload: {"account": "hive:user", "id": "1"}
//
//go:wasmexport balanceOf
func BalanceOf(payload *string) *string {
	assertInit()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p BalanceOfPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Account == "" {
		sdk.Abort("Account required")
	}
	if p.Id == "" {
		sdk.Abort("Token ID required")
	}

	bal := getBalance(p.Account, p.Id)
	return jsonResponse(BalanceResponse{Balance: bal})
}

// BalanceOfBatch returns token balances for multiple account/id pairs.
// Payload: {"accounts": ["hive:user1", "hive:user2"], "ids": ["1", "2"]}
//
//go:wasmexport balanceOfBatch
func BalanceOfBatch(payload *string) *string {
	assertInit()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p BalanceOfBatchPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if len(p.Accounts) == 0 {
		sdk.Abort("Accounts required")
	}
	if len(p.Accounts) != len(p.Ids) {
		sdk.Abort("Accounts and IDs length mismatch")
	}

	balances := make([]uint64, len(p.Accounts))
	for i := 0; i < len(p.Accounts); i++ {
		balances[i] = getBalance(p.Accounts[i], p.Ids[i])
	}
	return jsonResponse(BalanceBatchResponse{Balances: balances})
}

// IsApprovedForAll returns whether an operator is approved for all tokens of an account.
// Payload: {"account": "hive:owner", "operator": "hive:operator"}
//
//go:wasmexport isApprovedForAll
func IsApprovedForAll(payload *string) *string {
	assertInit()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p IsApprovedForAllPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Account == "" {
		sdk.Abort("Account required")
	}
	if p.Operator == "" {
		sdk.Abort("Operator required")
	}

	approved := isApprovedForAllInternal(p.Account, p.Operator)
	return jsonResponse(IsApprovedResponse{Approved: approved})
}

// URI returns the metadata URI for a token ID.
// Payload: {"id": "1"}
//
//go:wasmexport uri
func URI(payload *string) *string {
	assertInit()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p URIPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Id == "" {
		sdk.Abort("Token ID required")
	}

	uri := getTokenURI(p.Id)
	return jsonResponse(URIResponse{Uri: uri})
}

// GetOwnerExport returns the current contract owner.
//
//go:wasmexport getOwner
func GetOwnerExport(_ *string) *string {
	assertInit()
	owner := getOwnerAddress()
	return jsonResponse(OwnerResponse{Owner: owner})
}

// GetInfo returns contract metadata.
//
//go:wasmexport getInfo
func GetInfo(_ *string) *string {
	assertInit()
	return jsonResponse(InfoResponse{
		Name:        getContractName(),
		Symbol:      getContractSymbol(),
		BaseURI:     getBaseURI(),
		TrackMinted: isTrackMintedEnabled(),
	})
}

// IsPausedExport returns whether the contract is paused.
//
//go:wasmexport isPaused
func IsPausedExport(_ *string) *string {
	assertInit()
	return jsonResponse(PausedResponse{Paused: isPaused()})
}

// TotalSupply returns the total minted supply for a token ID.
// Payload: {"id": "1"}
//
//go:wasmexport totalSupply
func TotalSupply(payload *string) *string {
	assertInit()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p TotalSupplyPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Id == "" {
		sdk.Abort("Token ID required")
	}

	supply := getTotalSupply(p.Id)
	return jsonResponse(TotalSupplyResponse{TotalSupply: supply})
}

// MaxSupplyQuery returns the max supply for a token ID.
// Payload: {"id": "1"}
//
//go:wasmexport maxSupply
func MaxSupplyQuery(payload *string) *string {
	assertInit()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p MaxSupplyPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Id == "" {
		sdk.Abort("Token ID required")
	}

	max := getMaxSupply(p.Id)
	return jsonResponse(MaxSupplyResponse{MaxSupply: max})
}

// TotalMinted returns the total ever minted for a token ID (only tracked if trackMinted enabled).
// Payload: {"id": "1"}
//
//go:wasmexport totalMinted
func TotalMintedQuery(payload *string) *string {
	assertInit()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p TotalMintedPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Id == "" {
		sdk.Abort("Token ID required")
	}

	minted := getTotalMinted(p.Id)
	return jsonResponse(TotalMintedResponse{TotalMinted: minted})
}

// Exists returns whether a token ID has been minted (maxSupply > 0).
// Payload: {"id": "1"}
//
//go:wasmexport exists
func Exists(payload *string) *string {
	assertInit()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p ExistsPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Id == "" {
		sdk.Abort("Token ID required")
	}

	// A token exists if it has a maxSupply set (meaning it was minted at least once)
	exists := getMaxSupply(p.Id) > 0
	return jsonResponse(ExistsResponse{Exists: exists})
}

// IsSoulbound returns whether a token ID is soulbound (non-transferable).
// Payload: {"id": "1"}
//
//go:wasmexport isSoulbound
func IsSoulbound(payload *string) *string {
	assertInit()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p IsSoulboundPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Id == "" {
		sdk.Abort("Token ID required")
	}

	return jsonResponse(IsSoulboundResponse{Soulbound: isSoulbound(p.Id)})
}

// ===================================
// Token Properties Management
// ===================================

// SetProperties sets or updates the properties for a token ID.
// Payload: {"id": "1", "properties": {"color": "red", "rarity": "legendary"}}
// Only the contract owner can set properties.
//
//go:wasmexport setProperties
func SetProperties(payload *string) *string {
	assertInit()
	_, isOwner := getOwner()
	if !isOwner {
		sdk.Abort("Must be owner to set properties")
	}
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p SetPropertiesPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Id == "" {
		sdk.Abort("Token ID required")
	}
	if p.Properties == "" {
		sdk.Abort("Properties required")
	}

	setTokenProperties(p.Id, p.Properties)
	emitPropertiesSet(p.Id)
	return jsonResponse(SuccessResponse{Success: true})
}

// GetProperties returns the properties for a token ID.
// Payload: {"id": "1"}
//
//go:wasmexport getProperties
func GetProperties(payload *string) *string {
	assertInit()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p GetPropertiesPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Id == "" {
		sdk.Abort("Token ID required")
	}

	props := getTokenProperties(p.Id)
	return jsonResponse(PropertiesResponse{Properties: props})
}

// ===================================
// Collection Metadata Management
// ===================================

// SetCollectionMetadata sets or updates the collection-level metadata JSON.
// Payload: {"metadata": {"description": "My collection", "image": "https://..."}}
// Only the contract owner can set collection metadata.
//
//go:wasmexport setCollectionMetadata
func SetCollectionMetadata(payload *string) *string {
	assertInit()
	_, isOwner := getOwner()
	if !isOwner {
		sdk.Abort("Must be owner to set collection metadata")
	}
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p SetCollectionMetadataPayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.Metadata == "" {
		sdk.Abort("Metadata required")
	}

	setCollectionMetadata(p.Metadata)
	return jsonResponse(SuccessResponse{Success: true})
}

// GetCollectionMetadata returns the collection-level metadata JSON.
//
//go:wasmexport getCollectionMetadata
func GetCollectionMetadata(_ *string) *string {
	assertInit()
	metadata := getCollectionMetadata()
	return jsonResponse(CollectionMetadataResponse{Metadata: metadata})
}

// ===================================
// ERC-165 Interface Detection
// ===================================

// ERC-165 interface IDs
const (
	InterfaceIdERC165  = "0x01ffc9a7" // ERC-165 itself
	InterfaceIdERC1155 = "0xd9b67a26" // ERC-1155 Multi Token Standard
)

// SupportsInterface returns whether this contract implements a given interface (ERC-165).
// Payload: {"interfaceId": "0xd9b67a26"}
//
//go:wasmexport supportsInterface
func SupportsInterface(payload *string) *string {
	assertInit()
	if payload == nil || *payload == "" {
		sdk.Abort("Payload required")
	}

	var p SupportsInterfacePayload
	r := jlexer.Lexer{Data: []byte(*payload)}
	p.UnmarshalTinyJSON(&r)
	if r.Error() != nil {
		sdk.Abort("Invalid payload")
	}

	if p.InterfaceId == "" {
		sdk.Abort("Interface ID required")
	}

	// Check supported interfaces
	supported := p.InterfaceId == InterfaceIdERC165 || p.InterfaceId == InterfaceIdERC1155
	return jsonResponse(SupportsInterfaceResponse{Supported: supported})
}
