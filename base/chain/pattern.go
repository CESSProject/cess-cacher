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
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

const (
	ERR_Failed  = "failed"
	ERR_Timeout = "timeout"
	ERR_Empty   = "empty"
)

// error type
var (
	ERR_RPC_CONNECTION  = errors.New("rpc connection failed")
	ERR_RPC_IP_FORMAT   = errors.New("unsupported ip format")
	ERR_RPC_TIMEOUT     = errors.New("timeout")
	ERR_RPC_EMPTY_VALUE = errors.New("empty")
	ERR_TX_FAILED       = errors.New("tx failed")
)

type FileHash [64]types.U8
type SliceId [68]types.U8
type Random [20]types.U8
type Signature [65]types.U8
type Filter [256]types.U64
type Public [33]types.U8

// file meta info
type FileMetaInfo struct {
	Size       types.U64
	Index      types.U32
	State      types.Bytes
	UserBriefs []UserBrief
	Backups    []Backup
}

type UserBrief struct {
	User        types.AccountID
	File_name   types.Bytes
	Bucket_name types.Bytes
}

// bill info
type Bill struct {
	BID       string
	FileHash  string
	SliceHash string
	Account   string
	Expires   time.Time
	Amount    types.U128
}

type BillOnChain struct {
	Id              [16]byte
	To              types.AccountID
	Amount          types.U128
	File_hash       types.Hash
	Slice_hash      types.Hash
	Expiration_time uint64
}

// Backups
type Backup struct {
	Backup_index types.U8
	State        types.Bool
	Slice_info   []SliceInfo
}

// SliceInfo
type SliceInfo struct {
	Shard_id   SliceId
	Slice_hash FileHash
	Shard_size types.U64
	Miner_ip   Ipv4Type
	Miner_acc  types.AccountID
}

type Ipv4Type_Query struct {
	Placeholder types.U8 //
	Index       types.U8
	Value       [4]types.U8
	Port        types.U16
}

type IpAddress struct {
	IPv4 Ipv4Type
	IPv6 Ipv6Type
}
type Ipv4Type struct {
	Index types.U8
	Value [4]types.U8
	Port  types.U16
}
type Ipv6Type struct {
	Index types.U8
	Value [8]types.U16
	Port  types.U16
}

type BucketInfo struct {
	Total_capacity     types.U32
	Available_capacity types.U32
	Objects_num        types.U32
	Objects_list       []FileHash
	Authority          []types.AccountID
}

// cacher info
type CacherInfo struct {
	Ip         IpAddress
	Byte_price types.U128
}
