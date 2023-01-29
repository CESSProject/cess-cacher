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
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

func (c *chainClient) SubmitExtrinsic(method string,
	callback func(events CacheEventRecords) bool, args ...any) (string, error) {

	defer func() { recover() }()
	var (
		txhash string
		ext    types.Extrinsic
	)

	c.lock.Lock()
	defer c.lock.Unlock()

	if !c.IsChainClientOk() {
		c.SetChainState(false)
		return txhash, errors.Wrap(ERR_RPC_CONNECTION, "submit extrinsic error")
	}
	c.SetChainState(true)
	if call, err := types.NewCall(c.metadata, method, args...); err != nil {
		return txhash, errors.Wrap(err, "submit extrinsic error")
	} else {
		ext = types.NewExtrinsic(call)
	}
	accInfo, err := c.GetAccountInfo()
	if err != nil {
		return txhash, errors.Wrap(err, "submit extrinsic error")
	}
	o := types.SignatureOptions{
		BlockHash:          c.genesisHash,
		Era:                types.ExtrinsicEra{IsMortalEra: false},
		GenesisHash:        c.genesisHash,
		Nonce:              types.NewUCompactFromUInt(uint64(accInfo.Nonce)),
		SpecVersion:        c.runtimeVersion.SpecVersion,
		Tip:                types.NewUCompactFromUInt(0),
		TransactionVersion: c.runtimeVersion.TransactionVersion,
	}
	// Sign the transaction
	if err = ext.Sign(c.keyring, o); err != nil {
		return txhash, errors.Wrap(err, "submit extrinsic error")
	}
	// Do the transfer and track the actual status
	sub, err := c.api.RPC.Author.SubmitAndWatchExtrinsic(ext)
	if err != nil {
		if !strings.Contains(err.Error(), "Priority is too low") {
			return txhash, errors.Wrap(err, "submit extrinsic error")
		}
		for i := 0; i < 20; i++ {
			o.Nonce = types.NewUCompactFromUInt(uint64(accInfo.Nonce + types.NewU32(1)))
			// Sign the transaction
			if err = ext.Sign(c.keyring, o); err != nil {
				return txhash, errors.Wrap(err, "submit extrinsic error")
			}
			sub, err = c.api.RPC.Author.SubmitAndWatchExtrinsic(ext)
			if err == nil {
				break
			}
		}
		if err != nil {
			return txhash, errors.Wrap(err, "submit extrinsic error")
		}
	}
	defer sub.Unsubscribe()
	timeout := time.After(c.timeForBlockOut)
	for {
		select {
		case status := <-sub.Chan():
			if status.IsInBlock {
				events := CacheEventRecords{}
				txhash, _ = types.EncodeToHex(status.AsInBlock)
				h, err := c.api.RPC.State.GetStorageRaw(c.keyEvents, status.AsInBlock)
				if err != nil {
					return txhash, errors.Wrap(err, "submit extrinsic error")
				}
				types.EventRecordsRaw(*h).DecodeEventRecords(c.metadata, &events)

				if callback(events) {
					return txhash, nil
				}
				return txhash, errors.Wrap(ERR_TX_FAILED, "submit extrinsic error")
			}
		case err = <-sub.Err():
			return txhash, errors.Wrap(err, "submit extrinsic error")
		case <-timeout:
			return txhash, errors.Wrap(ERR_RPC_TIMEOUT, "submit extrinsic error")
		}
	}
}

func (c *chainClient) Register(ip, port string, price int64) (string, error) {
	info, err := NewCacherInfo(ip, port, price)
	if err != nil {
		return "", errors.Wrap(err, "register cacher error")
	}
	txhash, err := c.SubmitExtrinsic(
		CACHER_REGISTER,
		func(events CacheEventRecords) bool {
			return len(events.Cacher_Register) > 0
		},
		info,
	)
	return txhash, errors.Wrap(err, "register cacher error")
}

func (c *chainClient) Update(ip, port string, price int64) (string, error) {
	info, err := NewCacherInfo(ip, port, price)
	if err != nil {
		return "", errors.Wrap(err, "update cacher info error")
	}
	if _, err = c.GetMinerInfo(); err != nil {
		return "", errors.Wrap(err, "update cacher info error")
	}
	txhash, err := c.SubmitExtrinsic(
		CACHER_UPDATE,
		func(events CacheEventRecords) bool {
			return len(events.Cacher_Update) > 0
		},
		info,
	)
	return txhash, errors.Wrap(err, "update cacher info error")
}

func (c *chainClient) Logout() (string, error) {
	txhash, err := c.SubmitExtrinsic(
		CACHER_LOGOUT,
		func(events CacheEventRecords) bool {
			return len(events.Cacher_Logout) > 0
		},
	)
	return txhash, errors.Wrap(err, "logout cacher error")
}

func NewCacherInfo(ip, port string, price int64) (CacherInfo, error) {
	var info CacherInfo
	if !utils.IsIPv4(ip) {
		return info, ERR_RPC_IP_FORMAT
	}
	info.Ip.IPv4.Index = 0
	ips := strings.Split(ip, ".")
	for i := 0; i < len(info.Ip.IPv4.Value); i++ {
		tmp, err := strconv.Atoi(ips[i])
		if err != nil {
			return info, err
		}
		info.Ip.IPv4.Value[i] = types.U8(tmp)
	}
	tmp, err := strconv.Atoi(port)
	if err != nil {
		return info, err
	}
	info.Ip.IPv4.Port = types.U16(tmp)
	info.Byte_price = types.NewU128(*big.NewInt(price))
	return info, nil
}
