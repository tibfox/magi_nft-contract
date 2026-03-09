package main

import (
	"magi_nft/sdk"

	"github.com/CosmWasm/tinyjson/jwriter"
)

// ==========================================
// MAGI NFT - ERC-1155 Event Emission (tinyjson)
// ==========================================

// ==============
// Init Event
// ==============

func emitInit(owner, name, symbol, baseURI string) {
	txID := sdk.GetEnvKey("tx.id")
	event := InitEvent{
		Type: "init_magi_nft",
		Attributes: InitAttributes{
			Owner:   owner,
			Name:    name,
			Symbol:  symbol,
			BaseURI: baseURI,
		},
		Tx: *txID,
	}
	w := jwriter.Writer{}
	event.MarshalTinyJSON(&w)
	sdk.Log(string(w.Buffer.BuildBytes()))
}

// ==================
// TransferSingle Event (ERC-1155)
// ==================

func emitTransferSingle(operator, from, to, id string, value uint64) {
	txID := sdk.GetEnvKey("tx.id")
	event := TransferSingleEvent{
		Type: "TransferSingle",
		Attributes: TransferSingleAttributes{
			Operator: operator,
			From:     from,
			To:       to,
			Id:       id,
			Value:    value,
		},
		Tx: *txID,
	}
	w := jwriter.Writer{}
	event.MarshalTinyJSON(&w)
	sdk.Log(string(w.Buffer.BuildBytes()))
}

// ==================
// TransferBatch Event (ERC-1155)
// ==================

func emitTransferBatch(operator, from, to string, ids []string, values []uint64) {
	txID := sdk.GetEnvKey("tx.id")
	event := TransferBatchEvent{
		Type: "TransferBatch",
		Attributes: TransferBatchAttributes{
			Operator: operator,
			From:     from,
			To:       to,
			Ids:      ids,
			Values:   values,
		},
		Tx: *txID,
	}
	w := jwriter.Writer{}
	event.MarshalTinyJSON(&w)
	sdk.Log(string(w.Buffer.BuildBytes()))
}

// ======================
// ApprovalForAll Event (ERC-1155)
// ======================

func emitApprovalForAll(account, operator string, approved bool) {
	txID := sdk.GetEnvKey("tx.id")
	event := ApprovalForAllEvent{
		Type: "ApprovalForAll",
		Attributes: ApprovalForAllAttributes{
			Account:  account,
			Operator: operator,
			Approved: approved,
		},
		Tx: *txID,
	}
	w := jwriter.Writer{}
	event.MarshalTinyJSON(&w)
	sdk.Log(string(w.Buffer.BuildBytes()))
}

// ======================
// URI Event (ERC-1155)
// ======================

func emitURI(value, id string) {
	txID := sdk.GetEnvKey("tx.id")
	event := URIEvent{
		Type: "URI",
		Attributes: URIAttributes{
			Value: value,
			Id:    id,
		},
		Tx: *txID,
	}
	w := jwriter.Writer{}
	event.MarshalTinyJSON(&w)
	sdk.Log(string(w.Buffer.BuildBytes()))
}

// ======================
// Owner Change Event
// ======================

func emitOwnerChange(previousOwner, newOwner string) {
	txID := sdk.GetEnvKey("tx.id")
	event := OwnerChangeEvent{
		Type:       "ownerChange",
		Attributes: OwnerChangeAttributes{PreviousOwner: previousOwner, NewOwner: newOwner},
		Tx:         *txID,
	}
	w := jwriter.Writer{}
	event.MarshalTinyJSON(&w)
	sdk.Log(string(w.Buffer.BuildBytes()))
}

// ======================
// Pause Events
// ======================

func emitPaused(by string) {
	txID := sdk.GetEnvKey("tx.id")
	event := PausedEvent{
		Type:       "paused",
		Attributes: PausedAttributes{By: by},
		Tx:         *txID,
	}
	w := jwriter.Writer{}
	event.MarshalTinyJSON(&w)
	sdk.Log(string(w.Buffer.BuildBytes()))
}

func emitUnpaused(by string) {
	txID := sdk.GetEnvKey("tx.id")
	event := UnpausedEvent{
		Type:       "unpaused",
		Attributes: UnpausedAttributes{By: by},
		Tx:         *txID,
	}
	w := jwriter.Writer{}
	event.MarshalTinyJSON(&w)
	sdk.Log(string(w.Buffer.BuildBytes()))
}

// ======================
// BaseURI Change Event
// ======================

func emitBaseURIChange(previousURI, newURI string) {
	txID := sdk.GetEnvKey("tx.id")
	event := BaseURIChangeEvent{
		Type:       "baseUriChange",
		Attributes: BaseURIChangeAttributes{PreviousURI: previousURI, NewURI: newURI},
		Tx:         *txID,
	}
	w := jwriter.Writer{}
	event.MarshalTinyJSON(&w)
	sdk.Log(string(w.Buffer.BuildBytes()))
}

// ======================
// PropertiesSet Event
// ======================

func emitPropertiesSet(id string) {
	event := PropertiesSetEvent{
		Type:       "propertiesSet",
		Attributes: PropertiesSetAttributes{Id: id},
	}
	w := jwriter.Writer{}
	event.MarshalTinyJSON(&w)
	sdk.Log(string(w.Buffer.BuildBytes()))
}
