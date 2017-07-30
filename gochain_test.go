package gochain

import (
	"encoding/json"
	"log"
	"os"
	"testing"
)

func TestBlockMarshal(t *testing.T) {
	b := new(Block)
	b.Data = "Some data"
	b.Index = 1
	b.Timestamp = "2017-07-30 12:45:20"
	b.PreviousHash = "0"
	b.Hash = Hash(b)

	data, err := json.Marshal(*b)
	if err != nil {
		log.Fatal(err)
		t.Error()
	}

	expected := `{"Index":1,"PreviousHash":"0","Timestamp":"2017-07-30 12:45:20","Data":"Some data","Hash":"aa6d05a9ddd3cc4811fce8ebfc2a4a25f2fbeb093f2778b55584f807e0063f08"}`
	actual := string(data[:])

	os.Stdout.WriteString(actual)
	os.Stdout.WriteString(expected)

	if actual != expected {
		t.Error()
	}
}

func TestAddBlock(t *testing.T) {
	blockchain := new(BlockChain)

	b := new(Block)
	b.Data = "Genesis block"
	b.Index = 1
	b.Timestamp = "2017-07-30 12:45:20"
	b.PreviousHash = "0"
	b.Hash = Hash(b)
	AddBlock(b, blockchain)

	if len(blockchain.Blockchain) != 1 {
		t.Error()
	}
}

func TestBlockchainMarshal(t *testing.T) {
	blockchain := new(BlockChain)

	b1 := new(Block)
	b1.Data = "Genesis block"
	b1.Index = 1
	b1.Timestamp = "2017-07-30 16:08:01"
	b1.PreviousHash = "0"
	b1.Hash = Hash(b1)
	AddBlock(b1, blockchain)

	b2 := new(Block)
	b2.Data = "First block"
	b2.Index = 2
	b2.Timestamp = "2017-07-30 16:08:01"
	b2.PreviousHash = Hash(b1)
	b2.Hash = Hash(b2)
	AddBlock(b2, blockchain)

	data, err := json.Marshal(*blockchain)
	if err != nil {
		log.Fatal(err)
		t.Error()
	}

	expected := `{"Blockchain":[{"Index":1,"PreviousHash":"0","Timestamp":"2017-07-30 16:08:01","Data":"Genesis block","Hash":"cf56461fb9b0bf610292f777dab0088af3bcc8417b406bb5d5956a067eb6bdef"},{"Index":2,"PreviousHash":"cf56461fb9b0bf610292f777dab0088af3bcc8417b406bb5d5956a067eb6bdef","Timestamp":"2017-07-30 16:08:01","Data":"First block","Hash":"8721c937c765344fef804b667c2ae77dfa400bc427c4ef272b73b62e226914c9"}]}`
	actual := string(data[:])

	if actual != expected {
		t.Error()
	}

	//os.Stdout.Write(data)
}

func test(i interface{}) []byte {
	b, _ := json.Marshal(i)
	return b
}
