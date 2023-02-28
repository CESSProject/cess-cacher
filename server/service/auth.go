package service

import (
	"bytes"
	"cess-cacher/base/cache"
	resp "cess-cacher/server/response"
	"cess-cacher/utils"
	"crypto/aes"
	"crypto/cipher"

	"github.com/btcsuite/btcutil/base58"

	"github.com/pkg/errors"
)

var aesHandle AES

type AuthReq struct {
	Hash string `json:"hash"`
	BID  string `json:"bid"`
	Sign []byte `json:"sign"`
}

func GenerateToken(hash, bid string, sign []byte) (string, resp.Error) {
	var token string
	t, err := PraseTicketByBID(hash, bid)
	if err != nil {
		return token, resp.NewError(400, errors.Wrap(err, "generate token error"))
	}
	if !utils.VerifySign(t.Account, []byte(hash+bid), sign) {
		return token, resp.NewError(400, errors.Wrap(err, "generate token error"))
	}
	if ticketBeUsed(bid, t.Expires) {
		err := errors.New("invalid bill")
		return token, resp.NewError(400, errors.Wrap(err, "generate token error"))
	}
	if aesHandle.Enc == nil {
		aesHandle.Enc = []byte(utils.GetRandomcode(32))
	}
	hash58, err := utils.HexStringToBase58(hash)
	if err != nil {
		return token, resp.NewError(400, errors.Wrap(err, "generate token error"))
	}
	bid58, err := utils.HexStringToBase58(bid)
	if err != nil {
		return token, resp.NewError(400, errors.Wrap(err, "generate token error"))
	}
	cipText, err := aesHandle.SymmetricEncrypt([]byte(hash58 + "-" + bid58))
	if err != nil {
		return token, resp.NewError(400, errors.Wrap(err, "generate token error"))
	}
	token = base58.Encode(cipText)
	//data preheating: prepare the files not downloaded
	cache.GetCacheHandle().HitOrLoad(t.FileHash + "-" + t.SliceHash)
	deleteTicket(bid)
	return token, nil
}

type AES struct {
	Enc []byte
}

type AESHandle interface {
	SymmetricEncrypt(origData []byte) ([]byte, error)
	SymmetricDecrypt(ciphertext []byte) ([]byte, error)
}

func GetAESHandle() AESHandle {
	if aesHandle.Enc == nil {
		aesHandle.Enc = []byte(utils.GetRandomcode(32))
	}
	return aesHandle
}

func (method AES) SymmetricEncrypt(origData []byte) ([]byte, error) {
	block, err := aes.NewCipher(method.Enc)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	origData = PKCS5Padding(origData, blockSize)
	blockMode := cipher.NewCBCEncrypter(block, method.Enc[:blockSize])
	crypted := make([]byte, len(origData))
	blockMode.CryptBlocks(crypted, origData)
	return crypted, nil
}

func PKCS5Padding(ciphertext []byte, blockSize int) []byte {
	padding := blockSize - len(ciphertext)%blockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(ciphertext, padtext...)
}

func (method AES) SymmetricDecrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(method.Enc)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, method.Enc[:blockSize])
	origData := make([]byte, len(ciphertext))
	blockMode.CryptBlocks(origData, ciphertext)
	origData = PKCS5UnPadding(origData)
	return origData, nil
}

func PKCS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	if (length - unpadding) < 0 {
		return nil
	}
	return origData[:(length - unpadding)]
}
