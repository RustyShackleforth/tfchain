// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package explorergraphql

import (
	"fmt"
	"io"
	"strconv"

	"github.com/threefoldtech/rivine/crypto"
	"github.com/threefoldtech/rivine/types"
)

type Contract interface {
	IsContract()
}

type Object interface {
	IsObject()
}

type OutputParent interface {
	IsOutputParent()
}

type UnlockCondition interface {
	IsUnlockCondition()
}

type UnlockFulfillment interface {
	IsUnlockFulfillment()
}

type Wallet interface {
	IsWallet()
}

type AtomicSwapCondition struct {
	Version      ByteVersion              `json:"Version"`
	UnlockHash   types.UnlockHash         `json:"UnlockHash"`
	Sender       *UnlockHashPublicKeyPair `json:"Sender"`
	Receiver     *UnlockHashPublicKeyPair `json:"Receiver"`
	HashedSecret BinaryData               `json:"HashedSecret"`
	TimeLock     LockTime                 `json:"TimeLock"`
}

func (AtomicSwapCondition) IsUnlockCondition() {}

type AtomicSwapFulfillment struct {
	Version         ByteVersion     `json:"Version"`
	ParentCondition UnlockCondition `json:"ParentCondition"`
	PublicKey       types.PublicKey `json:"PublicKey"`
	Signature       Signature       `json:"Signature"`
	Secret          *BinaryData     `json:"Secret"`
}

func (AtomicSwapFulfillment) IsUnlockFulfillment() {}

type Balance struct {
	Unlocked BigInt `json:"Unlocked"`
	Locked   BigInt `json:"Locked"`
}

type BlockChainSnapshotFacts struct {
	TotalCoins                 *BigInt `json:"TotalCoins"`
	TotalLockedCoins           *BigInt `json:"TotalLockedCoins"`
	TotalBlockStakes           *BigInt `json:"TotalBlockStakes"`
	TotalLockedBlockStakes     *BigInt `json:"TotalLockedBlockStakes"`
	EstimatedActiveBlockStakes *BigInt `json:"EstimatedActiveBlockStakes"`
}

type BlockFacts struct {
	Difficulty    *BigInt                  `json:"Difficulty"`
	Target        *crypto.Hash             `json:"Target"`
	ChainSnapshot *BlockChainSnapshotFacts `json:"ChainSnapshot"`
}

type BlockHeader struct {
	ID          crypto.Hash        `json:"ID"`
	ParentID    *crypto.Hash       `json:"ParentID"`
	Parent      *Block             `json:"Parent"`
	Child       *Block             `json:"Child"`
	BlockTime   *types.Timestamp   `json:"BlockTime"`
	BlockHeight *types.BlockHeight `json:"BlockHeight"`
	Payouts     []*BlockPayout     `json:"Payouts"`
}

type ChainAggregatedData struct {
	TotalCoins                 *BigInt `json:"TotalCoins"`
	TotalLockedCoins           *BigInt `json:"TotalLockedCoins"`
	TotalBlockStakes           *BigInt `json:"TotalBlockStakes"`
	TotalLockedBlockStakes     *BigInt `json:"TotalLockedBlockStakes"`
	EstimatedActiveBlockStakes *BigInt `json:"EstimatedActiveBlockStakes"`
}

type ChainConstants struct {
	Name                              string            `json:"Name"`
	NetworkName                       string            `json:"NetworkName"`
	CoinUnit                          string            `json:"CoinUnit"`
	CoinPecision                      int               `json:"CoinPecision"`
	ChainVersion                      string            `json:"ChainVersion"`
	DefaultTransactionVersion         ByteVersion       `json:"DefaultTransactionVersion"`
	GatewayProtocolVersion            string            `json:"GatewayProtocolVersion"`
	ConsensusPlugins                  []string          `json:"ConsensusPlugins"`
	GenesisTimestamp                  types.Timestamp   `json:"GenesisTimestamp"`
	BlockSizeLimitInBytes             int               `json:"BlockSizeLimitInBytes"`
	AverageBlockCreationTimeInSeconds int               `json:"AverageBlockCreationTimeInSeconds"`
	GenesisTotalBlockStakes           BigInt            `json:"GenesisTotalBlockStakes"`
	BlockStakeAging                   int               `json:"BlockStakeAging"`
	BlockCreatorFee                   *BigInt           `json:"BlockCreatorFee"`
	MinimumTransactionFee             *BigInt           `json:"MinimumTransactionFee"`
	TransactionFeeBeneficiary         UnlockCondition   `json:"TransactionFeeBeneficiary"`
	PayoutMaturityDelay               types.BlockHeight `json:"PayoutMaturityDelay"`
}

type ChainFacts struct {
	Constants  *ChainConstants      `json:"Constants"`
	LastBlock  *Block               `json:"LastBlock"`
	Aggregated *ChainAggregatedData `json:"Aggregated"`
}

type LockTimeCondition struct {
	Version    ByteVersion       `json:"Version"`
	UnlockHash *types.UnlockHash `json:"UnlockHash"`
	LockValue  LockTime          `json:"LockValue"`
	LockType   LockType          `json:"LockType"`
	Condition  UnlockCondition   `json:"Condition"`
}

func (LockTimeCondition) IsUnlockCondition() {}

type MultiSignatureCondition struct {
	Version                ByteVersion                `json:"Version"`
	UnlockHash             types.UnlockHash           `json:"UnlockHash"`
	Owners                 []*UnlockHashPublicKeyPair `json:"Owners"`
	RequiredSignatureCount int                        `json:"RequiredSignatureCount"`
}

func (MultiSignatureCondition) IsUnlockCondition() {}

type MultiSignatureFulfillment struct {
	Version         ByteVersion               `json:"Version"`
	ParentCondition UnlockCondition           `json:"ParentCondition"`
	Pairs           []*PublicKeySignaturePair `json:"Pairs"`
}

func (MultiSignatureFulfillment) IsUnlockFulfillment() {}

type NilCondition struct {
	Version    ByteVersion      `json:"Version"`
	UnlockHash types.UnlockHash `json:"UnlockHash"`
}

func (NilCondition) IsUnlockCondition() {}

type PublicKeySignaturePair struct {
	PublicKey types.PublicKey `json:"PublicKey"`
	Signature Signature       `json:"Signature"`
}

type SingleSignatureFulfillment struct {
	Version         ByteVersion     `json:"Version"`
	ParentCondition UnlockCondition `json:"ParentCondition"`
	PublicKey       types.PublicKey `json:"PublicKey"`
	Signature       Signature       `json:"Signature"`
}

func (SingleSignatureFulfillment) IsUnlockFulfillment() {}

type TransactionFeePayout struct {
	BlockPayout *BlockPayout `json:"BlockPayout"`
	Value       BigInt       `json:"Value"`
}

type UnlockHashCondition struct {
	Version    ByteVersion      `json:"Version"`
	UnlockHash types.UnlockHash `json:"UnlockHash"`
	PublicKey  *types.PublicKey `json:"PublicKey"`
}

func (UnlockHashCondition) IsUnlockCondition() {}

type UnlockHashPublicKeyPair struct {
	UnlockHash types.UnlockHash `json:"UnlockHash"`
	PublicKey  *types.PublicKey `json:"PublicKey"`
}

type BlockPayoutType string

const (
	BlockPayoutTypeBlockReward    BlockPayoutType = "BLOCK_REWARD"
	BlockPayoutTypeTransactionFee BlockPayoutType = "TRANSACTION_FEE"
)

var AllBlockPayoutType = []BlockPayoutType{
	BlockPayoutTypeBlockReward,
	BlockPayoutTypeTransactionFee,
}

func (e BlockPayoutType) IsValid() bool {
	switch e {
	case BlockPayoutTypeBlockReward, BlockPayoutTypeTransactionFee:
		return true
	}
	return false
}

func (e BlockPayoutType) String() string {
	return string(e)
}

func (e *BlockPayoutType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = BlockPayoutType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid BlockPayoutType", str)
	}
	return nil
}

func (e BlockPayoutType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type LockType string

const (
	LockTypeBlockHeight LockType = "BLOCK_HEIGHT"
	LockTypeTimestamp   LockType = "TIMESTAMP"
)

var AllLockType = []LockType{
	LockTypeBlockHeight,
	LockTypeTimestamp,
}

func (e LockType) IsValid() bool {
	switch e {
	case LockTypeBlockHeight, LockTypeTimestamp:
		return true
	}
	return false
}

func (e LockType) String() string {
	return string(e)
}

func (e *LockType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = LockType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid LockType", str)
	}
	return nil
}

func (e LockType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type OutputType string

const (
	OutputTypeCoin                OutputType = "COIN"
	OutputTypeBlockStake          OutputType = "BLOCK_STAKE"
	OutputTypeBlockCreationReward OutputType = "BLOCK_CREATION_REWARD"
	OutputTypeTransactionFee      OutputType = "TRANSACTION_FEE"
)

var AllOutputType = []OutputType{
	OutputTypeCoin,
	OutputTypeBlockStake,
	OutputTypeBlockCreationReward,
	OutputTypeTransactionFee,
}

func (e OutputType) IsValid() bool {
	switch e {
	case OutputTypeCoin, OutputTypeBlockStake, OutputTypeBlockCreationReward, OutputTypeTransactionFee:
		return true
	}
	return false
}

func (e OutputType) String() string {
	return string(e)
}

func (e *OutputType) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = OutputType(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid OutputType", str)
	}
	return nil
}

func (e OutputType) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}