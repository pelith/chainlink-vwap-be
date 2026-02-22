package orderbook

import (
	"encoding/binary"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// EIP-712 domain and order type hashes matching contract VWAPRFQSpot.
const (
	eip712DomainType = "EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)"
	orderType        = "Order(address maker,bool makerIsSellETH,uint256 amountIn,uint256 minAmountOut,int32 deltaBps,uint256 salt,uint256 deadline)"
)

var (
	orderTypeHash    = crypto.Keccak256Hash([]byte(orderType))
	eip712DomainHash = crypto.Keccak256Hash([]byte(eip712DomainType))
)

// Verifier verifies EIP-712 order signatures using the same domain as the contract.
type Verifier struct {
	domainSeparator [32]byte
}

// NewVerifier builds the domain separator from chainId and verifying contract address.
func NewVerifier(chainID *big.Int, verifyingContract common.Address) *Verifier {
	nameHash := crypto.Keccak256Hash([]byte("VWAP-RFQ-Spot"))
	versionHash := crypto.Keccak256Hash([]byte("1"))
	domainSeparator := crypto.Keccak256Hash(
		packBytes32(eip712DomainHash[:]),
		packBytes32(nameHash[:]),
		packBytes32(versionHash[:]),
		packUint256(chainID),
		packAddress(verifyingContract),
	)
	return &Verifier{domainSeparator: domainSeparator}
}

// RecoverOrderSigner returns the address that signed the order, or error if invalid.
// Order hash (digest) must match the contract's _hashOrder output.
func (v *Verifier) RecoverOrderSigner(order *Order, signature []byte) (common.Address, error) {
	digest := v.OrderDigest(order)
	if len(signature) != 65 {
		return common.Address{}, ErrInvalidSignature
	}
	// signature is [R || S || V] with V being 0/1; ethereum uses V+27
	if signature[64] >= 27 {
		signature = append(signature[:64], signature[64]-27)
	}
	pubKey, err := crypto.SigToPub(digest.Bytes(), signature)
	if err != nil {
		return common.Address{}, ErrInvalidSignature
	}
	return crypto.PubkeyToAddress(*pubKey), nil
}

// OrderDigest returns the EIP-712 digest that was signed (keccak256("\x19\x01" || domainSeparator || structHash)).
func (v *Verifier) OrderDigest(order *Order) common.Hash {
	structHash := v.orderStructHash(order)
	return crypto.Keccak256Hash(
		[]byte("\x19\x01"),
		v.domainSeparator[:],
		structHash[:],
	)
}

func (v *Verifier) orderStructHash(order *Order) common.Hash {
	amountIn := new(big.Int)
	amountIn.SetString(order.AmountIn, 10)
	minAmountOut := new(big.Int)
	minAmountOut.SetString(order.MinAmountOut, 10)
	salt := new(big.Int)
	salt.SetString(order.Salt, 10)
	return crypto.Keccak256Hash(
		packBytes32(orderTypeHash[:]),
		packAddress(common.HexToAddress(order.Maker)),
		packBool(order.MakerIsSellETH),
		packUint256(amountIn),
		packUint256(minAmountOut),
		packInt32(order.DeltaBps),
		packUint256(salt),
		packUint64(uint64(order.Deadline)),
	)
}

func packBytes32(b []byte) []byte {
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func packAddress(a common.Address) []byte {
	out := make([]byte, 32)
	copy(out[12:], a.Bytes())
	return out
}

func packBool(b bool) []byte {
	out := make([]byte, 32)
	if b {
		out[31] = 1
	}
	return out
}

func packUint256(z *big.Int) []byte {
	b := z.Bytes()
	if len(b) > 32 {
		b = b[len(b)-32:]
	}
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

func packInt32(i int32) []byte {
	out := make([]byte, 32)
	if i < 0 {
		for j := 0; j < 28; j++ {
			out[j] = 0xff
		}
	}
	binary.BigEndian.PutUint32(out[28:], uint32(i))
	return out
}

func packUint64(u uint64) []byte {
	out := make([]byte, 32)
	binary.BigEndian.PutUint64(out[24:], u)
	return out
}
