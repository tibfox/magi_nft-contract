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
	if p.Symbol == "" {
		sdk.Abort("Symbol required")
	}

	// Store contract properties
	sdk.StateSetObject("contract_name", p.Name)
	sdk.StateSetObject("contract_symbol", p.Symbol)
	sdk.StateSetObject("base_uri", p.BaseURI)
	if p.TrackMinted {
		sdk.StateSetObject("track_minted", "1")
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
	if p.To == "" {
		sdk.Abort("To address required")
	}
	if p.Id == "" {
		sdk.Abort("Token ID required")
	}

	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	operator := *caller

	// Check authorization
	if !isApprovedOrOwner(operator, p.From) {
		sdk.Abort("Not authorized")
	}

	// Check soulbound - owner can transfer, recipients cannot
	owner, _ := getOwner()
	if isSoulbound(p.Id) && p.From != owner {
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
	if p.To == "" {
		sdk.Abort("To address required")
	}
	if len(p.Ids) == 0 {
		sdk.Abort("Token IDs required")
	}
	if len(p.Ids) != len(p.Amounts) {
		sdk.Abort("IDs and amounts length mismatch")
	}

	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	operator := *caller

	// Check authorization
	if !isApprovedOrOwner(operator, p.From) {
		sdk.Abort("Not authorized")
	}

	// Check soulbound and transfer each token type
	owner, _ := getOwner()
	for i := 0; i < len(p.Ids); i++ {
		if isSoulbound(p.Ids[i]) && p.From != owner {
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
	owner, isOwner := getOwner()
	if !isOwner {
		sdk.Abort("Must be owner to mint")
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

	if p.To == "" {
		sdk.Abort("To address required")
	}
	if p.Id == "" {
		sdk.Abort("Token ID required")
	}
	if p.Amount == 0 {
		sdk.Abort("Amount must be greater than 0")
	}

	// Check/set max supply for this token
	existingMax := getMaxSupply(p.Id)
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

	incBalance(p.To, p.Id, p.Amount)
	incTotalSupply(p.Id, p.Amount)
	emitTransferSingle(owner, "", p.To, p.Id, p.Amount) // Mint: from is zero address
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
	if len(p.Ids) == 0 {
		sdk.Abort("Token IDs required")
	}
	if len(p.Ids) != len(p.Amounts) {
		sdk.Abort("IDs and amounts length mismatch")
	}
	if len(p.MaxSupplies) > 0 && len(p.Ids) != len(p.MaxSupplies) {
		sdk.Abort("IDs and maxSupplies length mismatch")
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

		incBalance(p.To, p.Ids[i], p.Amounts[i])
		incTotalSupply(p.Ids[i], p.Amounts[i])
	}
	emitTransferBatch(owner, "", p.To, p.Ids, p.Amounts) // Mint: from is zero address
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
	if p.Id == "" {
		sdk.Abort("Token ID required")
	}
	if p.Amount == 0 {
		sdk.Abort("Amount must be greater than 0")
	}

	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	operator := *caller

	// Check authorization
	if !isApprovedOrOwner(operator, p.From) {
		sdk.Abort("Not authorized")
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
	if len(p.Ids) == 0 {
		sdk.Abort("Token IDs required")
	}
	if len(p.Ids) != len(p.Amounts) {
		sdk.Abort("IDs and amounts length mismatch")
	}

	caller := sdk.GetEnvKey("msg.caller")
	if caller == nil {
		sdk.Abort("Caller required")
	}
	operator := *caller

	// Check authorization
	if !isApprovedOrOwner(operator, p.From) {
		sdk.Abort("Not authorized")
	}

	for i := 0; i < len(p.Ids); i++ {
		if p.Amounts[i] == 0 {
			sdk.Abort("Amount must be greater than 0")
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
	sdk.StateSetObject("paused", "0")
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
	owner, _ := getOwner()
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
