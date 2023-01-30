/*
   Copyright 2022 CESS (Cumulus Encrypted Storage System) authors

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

        http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package chain

import (
	"cess-cacher/utils"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

// GetPublicKey returns your own public key
func (c *chainClient) GetPublicKey() []byte {
	return c.keyring.PublicKey
}

// GetPublicKey returns your own public key
func (c *chainClient) GetIncomeAccount() string {
	return c.IncomeAcc
}

func (c *chainClient) GetCessAccount() (string, error) {
	return utils.EncodePublicKeyAsCessAccount(c.keyring.PublicKey)
}

func (c *chainClient) GetSyncStatus() (bool, error) {
	if !c.IsChainClientOk() {
		return false, ERR_RPC_CONNECTION
	}
	h, err := c.api.RPC.System.Health()
	if err != nil {
		return false, err
	}
	return h.IsSyncing, nil
}

func (c *chainClient) GetStorageFromChain(target any, prefix, method string, args ...[]byte) error {
	defer func() {
		recover()
	}()
	if !c.IsChainClientOk() {
		c.SetChainState(false)
		return ERR_RPC_CONNECTION
	}
	c.SetChainState(true)

	key, err := types.CreateStorageKey(c.metadata, prefix, method, args...)
	if err != nil {
		return errors.Wrap(err, "get storage from chain error")
	}
	ok, err := c.api.RPC.State.GetStorageLatest(key, target)
	if err != nil {
		return errors.Wrap(err, "get storage from chain error")
	} else if !ok {
		return errors.Wrap(errors.New(ERR_Empty), "get storage from chain error")
	}
	return nil
}

// Query file meta info
func (c *chainClient) GetFileMetaInfo(fid string) (FileMetaInfo, error) {
	var (
		data FileMetaInfo
		hash FileHash
	)
	if len(fid) != len(hash) {
		return data, errors.New(ERR_Failed)
	}
	for i := 0; i < len(hash); i++ {
		hash[i] = types.U8(fid[i])
	}

	b, err := types.Encode(hash)
	if err != nil {
		return data, errors.Wrap(err, "get file metadata error")
	}
	err = c.GetStorageFromChain(&data, _FILEBANK, _FILEMAP_FILEMETA, b)
	if err != nil {
		return data, errors.Wrap(err, "get file metadata error")
	}
	return data, nil
}

func (c *chainClient) GetAccountInfo() (types.AccountInfo, error) {
	var info types.AccountInfo
	err := c.GetStorageFromChain(
		&info,
		_SYSTEM,
		_SYSTEM_ACCOUNT,
		c.keyring.PublicKey,
	)
	if err != nil {
		return info, errors.Wrap(err, "get account info error")
	}
	return info, nil
}

func (c *chainClient) GetMinerInfo() (CacherInfo, error) {
	var info CacherInfo
	err := c.GetStorageFromChain(
		&info,
		_CACHER,
		_CACHER_CACHER,
		[]byte(c.GetIncomeAccount()),
	)
	if err != nil {
		return info, errors.Wrap(err, "get cacher info error")
	}
	return info, nil
}

func (c *chainClient) GetBill(hash types.Hash, bid string) (Bill, error) {
	var bill Bill
	block, err := c.api.RPC.Chain.GetBlock(hash)
	if err != nil {
		return bill, errors.Wrap(err, "get bill error")
	}
	for _, ext := range block.Block.Extrinsics {
		args := parseArgs(ext.Method.Args, []int{128, 32, 32, 13, 128})
		if types.Decode(args[0], &bill.BID) != nil || bill.BID != bid {
			continue
		}
		if types.Decode(args[1], &bill.FileHash) != nil ||
			types.Decode(args[2], &bill.SliceHash) != nil ||
			types.Decode(args[3], &bill.Expires) != nil ||
			types.Decode(args[4], &bill.Amount) != nil {
			continue
		}
		acc, err := utils.EncodePublicKeyAsSubstrateAccount(ext.Signature.Signer.AsID[:])
		if err != nil {
			return bill, errors.Wrap(err, "get bill error")
		}
		bill.Account = acc
		return bill, nil
	}
	return bill, nil
}

func parseArgs(args []byte, lens []int) [][]byte {
	res := make([][]byte, len(lens))
	for i, l := range lens {
		res[i] = args[:l]
		args = args[l:]
	}
	return res
}
