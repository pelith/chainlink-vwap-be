package indexer

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// Event topic0 hashes (keccak256 of event signature) matching VWAPRFQSpot.sol.
var (
	// Filled(address indexed maker, address indexed taker, bytes32 indexed orderHash, ...).
	TopicFilled = crypto.Keccak256Hash([]byte("Filled(address,address,bytes32,uint64,uint64,uint256,uint256,bool,int32)"))
	// Cancelled(address indexed maker, bytes32 indexed orderHash).
	TopicCancelled = crypto.Keccak256Hash([]byte("Cancelled(address,bytes32)"))
	// Settled(bytes32 indexed tradeId, uint256 usdcPerEth, uint256 adjustedPrice, ...).
	TopicSettled = crypto.Keccak256Hash([]byte("Settled(bytes32,uint256,uint256,uint256,uint256,uint256,uint256)"))
	// Refunded(bytes32 indexed tradeId, uint256 makerRefund, uint256 takerRefund).
	TopicRefunded = crypto.Keccak256Hash([]byte("Refunded(bytes32,uint256,uint256)"))
)

// AllVWAPRFQTopics returns topic0 hashes for FilterQuery (any of these events).
func AllVWAPRFQTopics() []common.Hash {
	return []common.Hash{TopicFilled, TopicCancelled, TopicSettled, TopicRefunded}
}
