package main

// ===================================
// MAGI NFT - ERC-1155 JSON Types (tinyjson)
// ===================================

// ===================================
// Payload Types (Input)
// ===================================

// InitPayload for init action
type InitPayload struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BaseURI     string `json:"baseUri"`
	TrackMinted bool   `json:"trackMinted"` // Optional: if true, burned tokens can't be re-minted
}

// SafeTransferFromPayload for safeTransferFrom action
type SafeTransferFromPayload struct {
	From   string `json:"from"`
	To     string `json:"to"`
	Id     string `json:"id"`
	Amount uint64 `json:"amount"`
	Data   string `json:"data"`
}

// SafeBatchTransferFromPayload for safeBatchTransferFrom action
type SafeBatchTransferFromPayload struct {
	From    string   `json:"from"`
	To      string   `json:"to"`
	Ids     []string `json:"ids"`
	Amounts []uint64 `json:"amounts"`
	Data    string   `json:"data"`
}

// SetApprovalForAllPayload for setApprovalForAll action
type SetApprovalForAllPayload struct {
	Operator string `json:"operator"`
	Approved bool   `json:"approved"`
}

// BalanceOfPayload for balanceOf query
type BalanceOfPayload struct {
	Account string `json:"account"`
	Id      string `json:"id"`
}

// BalanceOfBatchPayload for balanceOfBatch query
type BalanceOfBatchPayload struct {
	Accounts []string `json:"accounts"`
	Ids      []string `json:"ids"`
}

// IsApprovedForAllPayload for isApprovedForAll query
type IsApprovedForAllPayload struct {
	Account  string `json:"account"`
	Operator string `json:"operator"`
}

// MintPayload for mint action
type MintPayload struct {
	To                 string `json:"to"`
	Id                 string `json:"id"`
	Amount             uint64 `json:"amount"`
	MaxSupply          uint64 `json:"maxSupply"`          // Required: 1 = unique, >1 = editioned
	Soulbound          bool   `json:"soulbound"`          // Optional: if true, token cannot be transferred
	Properties         string `json:"properties"`         // Optional: arbitrary JSON stored as raw string
	PropertiesTemplate string `json:"propertiesTemplate"` // Optional: token ID to inherit properties from
	Data               string `json:"data"`
}

// MintBatchPayload for mintBatch action
type MintBatchPayload struct {
	To                 string   `json:"to"`
	Ids                []string `json:"ids"`
	Amounts            []uint64 `json:"amounts"`
	MaxSupplies        []uint64 `json:"maxSupplies"`       // Required: per-token max supply
	Soulbound          []bool   `json:"soulbound"`         // Optional: per-token soulbound flags
	Properties         []string `json:"properties"`        // Optional: per-token arbitrary JSON properties
	PropertiesTemplate string   `json:"propertiesTemplate"` // Optional: token ID to inherit properties from (tokens without explicit properties get this template)
	Data               string   `json:"data"`
}

// TotalSupplyPayload for totalSupply query
type TotalSupplyPayload struct {
	Id string `json:"id"`
}

// MaxSupplyPayload for maxSupply query
type MaxSupplyPayload struct {
	Id string `json:"id"`
}

// TotalMintedPayload for totalMinted query
type TotalMintedPayload struct {
	Id string `json:"id"`
}

// ExistsPayload for exists query
type ExistsPayload struct {
	Id string `json:"id"`
}

// BurnPayload for burn action
type BurnPayload struct {
	From   string `json:"from"`
	Id     string `json:"id"`
	Amount uint64 `json:"amount"`
}

// BurnBatchPayload for burnBatch action
type BurnBatchPayload struct {
	From    string   `json:"from"`
	Ids     []string `json:"ids"`
	Amounts []uint64 `json:"amounts"`
}

// URIPayload for uri query
type URIPayload struct {
	Id string `json:"id"`
}

// SetURIPayload for setURI action
type SetURIPayload struct {
	Id  string `json:"id"`
	Uri string `json:"uri"`
}

// SetBaseURIPayload for setBaseURI action
type SetBaseURIPayload struct {
	BaseURI string `json:"baseUri"`
}

// ChangeOwnerPayload for changeOwner action
type ChangeOwnerPayload struct {
	NewOwner string `json:"newOwner"`
}

// SetPropertiesPayload for setProperties action
type SetPropertiesPayload struct {
	Id         string `json:"id"`
	Properties string `json:"properties"` // Arbitrary JSON stored as raw string
}

// GetPropertiesPayload for getProperties query
type GetPropertiesPayload struct {
	Id string `json:"id"`
}

// SupportsInterfacePayload for supportsInterface query (ERC-165)
type SupportsInterfacePayload struct {
	InterfaceId string `json:"interfaceId"`
}

// ===================================
// Response Types (Output)
// ===================================

// BalanceResponse for balance queries
type BalanceResponse struct {
	Balance uint64 `json:"balance"`
}

// BalanceBatchResponse for balanceOfBatch query
type BalanceBatchResponse struct {
	Balances []uint64 `json:"balances"`
}

// TotalSupplyResponse for totalSupply query
type TotalSupplyResponse struct {
	TotalSupply uint64 `json:"totalSupply"`
}

// MaxSupplyResponse for maxSupply query
type MaxSupplyResponse struct {
	MaxSupply uint64 `json:"maxSupply"`
}

// TotalMintedResponse for totalMinted query
type TotalMintedResponse struct {
	TotalMinted uint64 `json:"totalMinted"`
}

// ExistsResponse for exists query
type ExistsResponse struct {
	Exists bool `json:"exists"`
}

// IsSoulboundPayload for isSoulbound query
type IsSoulboundPayload struct {
	Id string `json:"id"`
}

// IsSoulboundResponse for isSoulbound query
type IsSoulboundResponse struct {
	Soulbound bool `json:"soulbound"`
}

// IsApprovedResponse for isApprovedForAll query
type IsApprovedResponse struct {
	Approved bool `json:"approved"`
}

// URIResponse for uri query
type URIResponse struct {
	Uri string `json:"uri"`
}

// OwnerResponse for owner queries
type OwnerResponse struct {
	Owner string `json:"owner"`
}

// InfoResponse for contract info queries
type InfoResponse struct {
	Name        string `json:"name"`
	Symbol      string `json:"symbol"`
	BaseURI     string `json:"baseUri"`
	TrackMinted bool   `json:"trackMinted"`
}

// PausedResponse for isPaused queries
type PausedResponse struct {
	Paused bool `json:"paused"`
}

// SuccessResponse for mutation operations
type SuccessResponse struct {
	Success bool `json:"success"`
}

// PropertiesResponse for getProperties query
type PropertiesResponse struct {
	Properties string `json:"properties"` // Raw JSON string
}

// SupportsInterfaceResponse for supportsInterface query (ERC-165)
type SupportsInterfaceResponse struct {
	Supported bool `json:"supported"`
}

// ===================================
// Event Types
// ===================================

// InitEvent for contract initialization
type InitEvent struct {
	Type       string         `json:"type"`
	Attributes InitAttributes `json:"attributes"`
	Tx         string         `json:"tx"`
}

type InitAttributes struct {
	Owner   string `json:"owner"`
	Name    string `json:"name"`
	Symbol  string `json:"symbol"`
	BaseURI string `json:"baseUri"`
}

// TransferSingleEvent for single token transfers (ERC-1155)
type TransferSingleEvent struct {
	Type       string                   `json:"type"`
	Attributes TransferSingleAttributes `json:"attributes"`
	Tx         string                   `json:"tx"`
}

type TransferSingleAttributes struct {
	Operator string `json:"operator"`
	From     string `json:"from"`
	To       string `json:"to"`
	Id       string `json:"id"`
	Value    uint64 `json:"value"`
}

// TransferBatchEvent for batch token transfers (ERC-1155)
type TransferBatchEvent struct {
	Type       string                  `json:"type"`
	Attributes TransferBatchAttributes `json:"attributes"`
	Tx         string                  `json:"tx"`
}

type TransferBatchAttributes struct {
	Operator string   `json:"operator"`
	From     string   `json:"from"`
	To       string   `json:"to"`
	Ids      []string `json:"ids"`
	Values   []uint64 `json:"values"`
}

// ApprovalForAllEvent for operator approval (ERC-1155)
type ApprovalForAllEvent struct {
	Type       string                   `json:"type"`
	Attributes ApprovalForAllAttributes `json:"attributes"`
	Tx         string                   `json:"tx"`
}

type ApprovalForAllAttributes struct {
	Account  string `json:"account"`
	Operator string `json:"operator"`
	Approved bool   `json:"approved"`
}

// URIEvent for URI changes (ERC-1155)
type URIEvent struct {
	Type       string        `json:"type"`
	Attributes URIAttributes `json:"attributes"`
	Tx         string        `json:"tx"`
}

type URIAttributes struct {
	Value string `json:"value"`
	Id    string `json:"id"`
}

// OwnerChangeEvent for ownership transfers
type OwnerChangeEvent struct {
	Type       string                `json:"type"`
	Attributes OwnerChangeAttributes `json:"attributes"`
	Tx         string                `json:"tx"`
}

type OwnerChangeAttributes struct {
	PreviousOwner string `json:"previousOwner"`
	NewOwner      string `json:"newOwner"`
}

// PausedEvent for pause action
type PausedEvent struct {
	Type       string           `json:"type"`
	Attributes PausedAttributes `json:"attributes"`
	Tx         string           `json:"tx"`
}

type PausedAttributes struct {
	By string `json:"by"`
}

// UnpausedEvent for unpause action
type UnpausedEvent struct {
	Type       string             `json:"type"`
	Attributes UnpausedAttributes `json:"attributes"`
	Tx         string             `json:"tx"`
}

type UnpausedAttributes struct {
	By string `json:"by"`
}

// BaseURIChangeEvent for base URI updates
type BaseURIChangeEvent struct {
	Type       string                  `json:"type"`
	Attributes BaseURIChangeAttributes `json:"attributes"`
	Tx         string                  `json:"tx"`
}

type BaseURIChangeAttributes struct {
	PreviousURI string `json:"previousUri"`
	NewURI      string `json:"newUri"`
}

// PropertiesSetEvent for property updates
type PropertiesSetEvent struct {
	Type       string                  `json:"type"`
	Attributes PropertiesSetAttributes `json:"attributes"`
}

type PropertiesSetAttributes struct {
	Id string `json:"id"`
}
