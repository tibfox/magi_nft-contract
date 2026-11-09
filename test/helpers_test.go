package contract_test

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"vsc-node/lib/test_utils"
	contract_session "vsc-node/modules/contract/session"
	"vsc-node/modules/db/vsc/contracts"
	ledgerDb "vsc-node/modules/db/vsc/ledger"
	stateEngine "vsc-node/modules/state-processing"

	"github.com/stretchr/testify/assert"
)

// rcFundingHbd is the hbd balance each test account is created with. 1 RC == 1
// hbd milliunit, so this is 1,000 HBD == 1,000,000 RC: comfortably above the
// helper's RcLimit of 100,000, which keeps the RC limit rather than the account
// balance as the gas ceiling, with room for a whole suite of calls.
const rcFundingHbd int64 = 1_000_000

// CallResult is an alias for the test call result type
type CallResult = test_utils.ContractTestCallResult

var _ = embed.FS{} // just so "embed" can be imported

const ContractID = "vsctestcontract"
const ownerAddress = "hive:tibfox"

// Default init payload for tests (ERC-1155)
var DefaultInitPayload = []byte(`{"name":"Magi NFT","symbol":"MNFT","baseUri":"https://api.magi.network/metadata/"}`)

//go:embed artifacts/main.wasm
var ContractWasm []byte

// testAccounts is every account this suite calls as. SetupContractTest funds each
// one, because the node derives a contract call's WASM gas budget from the
// caller's RC — transactions.go computes gas = min(availableRC, RcLimit), and
// rc-system derives availableRC from the account's hbd balance plus the free
// allowance for Hive accounts (params.RC_HIVE_FREE_AMOUNT, 10_000). RC is charged
// cumulatively, so on the free allowance alone a caller runs out part-way through
// a multi-call test and the rest of its calls fail on budget rather than on
// anything the contract did. Funded callers keep RcLimit (and each test's maxGas
// assertion) the binding limit, which is what these tests mean to measure.
var testAccounts = []string{
	"hive:alice", "hive:attacker", "hive:bob", "hive:buyer", "hive:buyer1",
	"hive:buyer2", "hive:collector", "hive:contractx", "hive:contracty",
	"hive:holder", "hive:market", "hive:marketplace", "hive:newowner",
	"hive:operator", "hive:operator1", "hive:operator2", "hive:other",
	"hive:othercontract", "hive:owner2", "hive:owner3", "hive:recipient",
	"hive:regularuser", "hive:stranger", "hive:third", "hive:tibfox",
	"hive:unknown", "hive:user1",
}

// Setup an Instance of a test
func SetupContractTest() *test_utils.ContractTest {
	CleanBadgerDB()
	ct := test_utils.NewContractTest()
	for _, account := range testAccounts {
		ct.Deposit(account, rcFundingHbd, ledgerDb.AssetHbd)
	}
	ct.RegisterContract(ContractID, ownerAddress, ContractWasm)
	return &ct
}

// clean the db for multiple (sequential) tests
func CleanBadgerDB() {
	err := os.RemoveAll("data/badger")
	if err != nil {
		panic("failed to remove data/badger")
	}
}

// CallContract executes a contract action and asserts basic success
func CallContract(
	t *testing.T,
	ct *test_utils.ContractTest,
	action string,
	payload json.RawMessage,
	intents []contracts.Intent,
	authUser string,
	expectedResult bool,
	maxGas uint,
	expectedOutput string,

) CallResult {
	fmt.Println(action)
	fmt.Println(string(payload))
	result := ct.Call(stateEngine.TxVscCallContract{
		Caller: authUser,

		Self: stateEngine.TxSelf{
			TxId:                 fmt.Sprintf("%s-tx", action),
			BlockId:              "block1",
			Index:                0,
			OpIndex:              0,
			Timestamp:            "2025-09-03T00:00:00",
			RequiredAuths:        []string{authUser},
			RequiredPostingAuths: []string{},
		},
		ContractId: ContractID,
		Action:     action,
		Payload:    payload,
		RcLimit:    100000,
		Intents:    intents,
	})

	PrintLogs(result.Logs)
	PrintErrorIfFailed(result)
	fmt.Printf("return msg: %s\n", result.Ret)
	fmt.Printf("RC used: %d\n", result.RcUsed)
	fmt.Printf("gas used: %d\n", result.GasUsed)
	fmt.Printf("gas max : %d\n", maxGas)

	assert.LessOrEqual(t, result.GasUsed, maxGas, fmt.Sprintf("Gas %d exceeded limit %d", result.GasUsed, maxGas))

	if expectedResult {
		assert.True(t, result.Success, "Contract action failed with "+result.Ret)
	} else {
		assert.False(t, result.Success, "Contract action did not fail (as expected)")
	}
	if expectedOutput != "" {
		assert.True(t, startsWith(result.Ret, expectedOutput), true)
	}
	return result
}

// startsWith checks whether s begins with prefix, with no allocation.
func startsWith(s, prefix string) bool {
	if len(prefix) > len(s) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if s[i] != prefix[i] {
			return false
		}
	}
	return true
}

// PrintLogs prints all logs from a contract call
func PrintLogs(logs map[string]contract_session.LogOutput) {
	for key, values := range logs {
		for _, v := range values.Logs {
			fmt.Printf("[%s] %s\n", key, v)
		}
	}
}

// PrintErrorIfFailed prints error if the contract call failed
func PrintErrorIfFailed(result CallResult) {
	if !result.Success {
		fmt.Println(result.ErrMsg)
	}
}

// ToJSONRaw converts Go objects to json.RawMessage
func ToJSONRaw(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal JSON: %v", err))
	}
	return b
}

// PayloadToJSON safely converts payloads to json.RawMessage
func PayloadToJSON(v any) json.RawMessage {
	switch val := v.(type) {
	case string:
		return json.RawMessage([]byte(val)) // no quoting
	case json.RawMessage:
		return val
	default:
		return ToJSONRaw(val) // fallback to normal marshaling
	}
}

type ContractTestCase struct {
	Name          string
	Action        string
	Payload       any
	Intents       []contracts.Intent
	ExpectLogs    bool
	AuthUser      string
	ExpectSuccess bool
}

// for table-driven tests
func RunContractTests(t *testing.T, ct *test_utils.ContractTest, tests []ContractTestCase) {
	for _, tt := range tests {
		tt := tt // capture range variable
		t.Run(tt.Name, func(t *testing.T) {
			CallContract(t, ct, tt.Action, PayloadToJSON(tt.Payload), tt.Intents, tt.AuthUser, tt.ExpectSuccess, uint(100_000_000), "")
		})
	}
}

func toStringPtr(s string) *string {
	return &s
}
