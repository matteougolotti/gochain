package gochain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/websocket"
)

// Block is the basic block of the chain
type Block struct {
	Index        int
	PreviousHash string
	Timestamp    string // 2017-06-03T16:34:32
	Data         string
	Hash         string
}

// NewBlock builds a new block struct
func NewBlock(index int, previousHash *string, timestamp *time.Time, data *string) *Block {
	block := new(Block)
	block.Index = index
	block.PreviousHash = *previousHash
	block.Timestamp = FormatFromTime(timestamp)
	block.Data = *data
	block.Hash = Hash(block)
	return block
}

// BlockChain contains a list of blocks
type BlockChain struct {
	Blockchain []Block
}

// Types of messages
const (
	QueryBlockchain    = 0
	ResponseBlockchain = 1
	QueryAddBlock      = 2
)

// WSMessage represents a WebSocket query
type WSMessage struct {
	Type int
}

// NewWSMessage parses a JSON string into a WSMessage
func NewWSMessage(msg *[]byte) *WSMessage {
	wsmsg := new(WSMessage)
	json.Unmarshal(*msg, wsmsg)
	return wsmsg
}

// WSResponseBlockchain represents a response to synchronize the blockchain
type WSResponseBlockchain struct {
	Type       int
	Blockchain BlockChain
}

// NewWSResponseBlockchain deserializes a JSON string into a blockchain response
func NewWSResponseBlockchain(msg *[]byte) *WSResponseBlockchain {
	wsmsg := new(WSResponseBlockchain)
	json.Unmarshal(*msg, wsmsg)
	return wsmsg
}

// WSQueryAddBlock is a query to add a newly added block to other nodes
type WSQueryAddBlock struct {
	Type  int
	Block Block
}

// NewWSQueryAddBlock deserialized a JSON string into a WSQueryAddBlock
func NewWSQueryAddBlock(msg *[]byte) *WSQueryAddBlock {
	wsmsg := new(WSQueryAddBlock)
	json.Unmarshal(*msg, wsmsg)
	return wsmsg
}

// Hash computes the SHA256 hash of the given block
func Hash(block *Block) string {
	data := strconv.Itoa(block.Index) + block.PreviousHash + block.Timestamp + block.Data
	h := sha256.New()
	io.WriteString(h, data)
	return hex.EncodeToString(h.Sum(nil))
}

// FormatFromTime serializes a time to a string
func FormatFromTime(t *time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

// FormatToTime deserialized a string to a time
func FormatToTime(s *string) (time.Time, error) {
	t, err := time.Parse("2006-01-02 15:04:05", *s)
	return t, err
}

// NextIndex returns the next available index in the blockchain
func NextIndex(bc *BlockChain) int {
	return len(bc.Blockchain) + 1
}

// LastBlock returns the last valid block in the blockchain
func LastBlock(b *BlockChain) *Block {
	return &(b.Blockchain[len(b.Blockchain)-1])
}

// AddBlock adds the block to the blockchain
func AddBlock(b *Block, bc *BlockChain) {
	bc.Blockchain = append(bc.Blockchain, *b)
}

// NextBlock generates a new block for the chain with the given data
func NextBlock(data string, bc *BlockChain) *Block {
	index := NextIndex(bc)
	prevHash := Hash(LastBlock(bc))
	timestamp := time.Now()
	return NewBlock(index, &prevHash, &timestamp, &data)
}

// GetGenesisBlock initializes a genesis block
func GetGenesisBlock() *Block {
	b := new(Block)
	b.Index = 0
	b.Data = "The genesis block"
	b.PreviousHash = ""
	t := time.Now()
	b.Timestamp = FormatFromTime(&t)
	b.Hash = Hash(b)
	return b
}

// IsBlockValid returns true if the new block is valid compared to the previous one, false otherwise
func IsBlockValid(block *Block, prevBlock *Block) bool {
	if (block.Index == prevBlock.Index+1) &&
		(block.PreviousHash == prevBlock.Hash) &&
		(Hash(block) == block.Hash) {
		return true
	}

	return false
}

// IsChainValid checks the validity of all the blocks in the chain
func IsChainValid(blockchain *BlockChain) bool {
	prevBlock := &(blockchain.Blockchain[0])
	for i := 1; i < len(blockchain.Blockchain); i++ {
		curBlock := &(blockchain.Blockchain[i])
		if !IsBlockValid(curBlock, prevBlock) {
			return false
		}

		prevBlock = curBlock
	}

	return true
}

// ReplaceChain returns the longest valid chain between the two
func ReplaceChain(newBlocks *BlockChain, blockchain *BlockChain) *BlockChain {
	if IsChainValid(newBlocks) &&
		(len(newBlocks.Blockchain) > len(blockchain.Blockchain)) {
		return newBlocks
	}

	return blockchain
}

// HTTP services

// InitHTTPServer sets up the REST services to control the node
func InitHTTPServer() {
	http.HandleFunc("/blocks", GetBlocks)
	http.HandleFunc("/addBlock", PostBlock)
	http.HandleFunc("/peers", GetPeers)
	http.HandleFunc("/addPeer", AddPeer)

	http.ListenAndServe(":8080", nil)
}

// GetBlocks returns a full copy of the current blockchain
func GetBlocks(w http.ResponseWriter, r *http.Request) {
	resJSON, ok := json.Marshal(blockchain.Blockchain)
	if ok == nil {
		n := len(resJSON)
		res := string(resJSON[:n])
		io.WriteString(w, res)
	}
}

// PostBlock handles HTTP POST requests to add new data to the blockchain.
// All the data in the request's body is added to a new block in the chain.
func PostBlock(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	data, err := ioutil.ReadAll(r.Body)
	if err == nil {
		n := len(data)
		s := string(data[:n])
		nb := NextBlock(s, blockchain)
		prevBlock := LastBlock(blockchain)
		if IsBlockValid(nb, prevBlock) {
			AddBlock(nb, blockchain)
			wsq := new(WSQueryAddBlock)
			wsq.Type = QueryAddBlock
			wsq.Block = *nb
			Broadcast(wsq)
			io.WriteString(w, "New block added")
			return
		}
	}

	io.WriteString(w, "Error adding block")
	return
}

// GetPeers returns the registered peers
func GetPeers(w http.ResponseWriter, r *http.Request) {
	resJSON, err := json.Marshal(peers)
	if err == nil {
		n := len(resJSON)
		res := string(resJSON[:n])
		io.WriteString(w, res)
	}
}

// AddPeer adds a new peer to the list of known peers
// { "peer": "URI"}
func AddPeer(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var peer struct {
		Peer string `json:"peer"`
	}

	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Fatal("addPeer", err)
	}

	err = json.Unmarshal(data, &peer)
	if err != nil {
		log.Fatal("addPeer", err)
	}

	peers = append(peers, peer.Peer)
	log.Println("Added new peer ", peer.Peer)
}

// P2P WebSocket services

// InitP2PServer sets up the WebSocket services for P2P communication
func InitP2PServer() {
	http.HandleFunc("/ws", P2PHandler)

	http.ListenAndServe(":8090", nil)
}

// P2PHandler handles P2P communications over WebSocket
func P2PHandler(w http.ResponseWriter, r *http.Request) {
	upgrader := new(websocket.Upgrader)
	upgrader.ReadBufferSize = 1024
	upgrader.WriteBufferSize = 1024

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	_, msg, e := conn.ReadMessage()
	if e != nil {
		log.Fatal(e)
	}

	wsmsg := NewWSMessage(&msg)
	if wsmsg.Type == QueryAddBlock {
		// Add the received block
		wsmsgab := NewWSQueryAddBlock(&msg)
		AddBlock(&wsmsgab.Block, blockchain)
	} else if wsmsg.Type == QueryBlockchain {
		// Send back a copy of the local blockchain
		conn.WriteJSON(*blockchain)
	}
}

// Broadcast sends a message to all known peers
func Broadcast(msg interface{}) {
	for _, peer := range peers {
		//c, _, err := websocket.DefaultDialer.Dial("http://localhost/", nil)
		c, _, err := websocket.DefaultDialer.Dial(peer, nil)
		defer c.Close()
		if err != nil {
			log.Fatal(err)
		}

		err = c.WriteJSON(msg)
		if err != nil {
			log.Print(err)
		}
	}
}

var sockets []*websocket.Conn
var blockchain *BlockChain
var peers []string

func main() {
	blockchain = new(BlockChain)
	go InitHTTPServer()
	go InitP2PServer()

	// TODO
}
