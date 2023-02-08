package test

import (
	"cess-cacher/server/service"
	"cess-cacher/utils"
	"log"
	"testing"

	"github.com/btcsuite/btcutil/base58"
)

func TestAes(t *testing.T) {
	aes := service.AES{[]byte(utils.GetRandomcode(32))}
	bytes, err := aes.SymmetricEncrypt([]byte("674f8c1146b547fb94906d085aea8294"))
	if err != nil {
		t.Fatal("encrypt error", err)
	}
	log.Println("enc text", base58.Encode(bytes), "len", len(base58.Encode(bytes)))
	pbs, err := aes.SymmetricDecrypt(bytes)
	if err != nil {
		t.Fatal("decrypt error", err)
	}
	log.Println("plan text", string(pbs), "len", len(pbs))
}
