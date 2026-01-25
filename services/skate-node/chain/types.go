// Source [10] - The Production Types
package chain

import (
    "crypto/sha256"
    "encoding/json"
    "time"
)

type BlockHeader struct {
    Height     uint64   `json:"height"`
    PrevHash   [16]byte `json:"prev_hash"`
    MerkleRoot [16]byte `json:"merkle_root"`
    StateRoot  [16]byte `json:"state_root"`
    Timestamp  int64    `json:"timestamp"`
    Nonce      uint64   `json:"nonce"`
    Difficulty uint64   `json:"difficulty"`
    Miner      string   `json:"miner"`
}

type Block struct {
    Header       BlockHeader         `json:"header"`
    Transactions []Transaction       `json:"transactions"`
    Approvals    []ValidatorApproval `json:"approvals"`
}

type ValidatorApproval struct {
    Validator string `json:"validator"`
    Signature []byte `json:"signature"`
    Timestamp int64  `json:"timestamp"`
}

type Transaction struct {
    ID        [16]byte `json:"id"`
    Type      string   `json:"type"` // "transfer", "game_action", "stake"
    From      string   `json:"from"`
    To        string   `json:"to"`
    Amount    uint64   `json:"amount"`
    Nonce     uint64   `json:"nonce"`
    Payload   []byte   `json:"payload"`
    Signature []byte   `json:"signature"`
    Timestamp int64    `json:"timestamp"`
}
