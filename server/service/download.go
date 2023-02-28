package service

import (
	"cess-cacher/base/cache"
	"cess-cacher/base/chain"
	"cess-cacher/config"
	resp "cess-cacher/server/response"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/centrifuge/go-substrate-rpc-client/v4/types"
	"github.com/pkg/errors"
)

type Ticket struct {
	BID       string
	FileHash  string
	SliceHash string
	Account   string
	Size      uint64
	Expires   time.Time
}

const TAB_FLASH_TIME = 3 * time.Hour

var tickets *sync.Map

func InitTickets() {
	if tickets == nil {
		tickets = new(sync.Map)
	}
}

func deleteTicket(key string) {
	tickets.Delete(key)
}

func DownloadService(t Ticket) (string, resp.Error) {
	var slicePath string
	if time.Since(t.Expires) >= 0 {
		err := errors.New("The ticket has expired")
		return slicePath, resp.NewError(400, errors.Wrap(err, "download service error"))
	}
	if ticketBeUsed(t.BID, t.Expires) {
		err := errors.New("The ticket has been used")
		return slicePath, resp.NewError(400, errors.Wrap(err, "download service error"))
	}
	if ok, err := cache.GetCacheHandle().HitOrLoad(t.FileHash + "-" + t.SliceHash); !ok {
		if err != nil {
			tickets.Delete(t.BID)
			return slicePath, resp.NewError(500, errors.Wrap(err, "download service error"))
		}
		if count, ok := cache.GetCacheHandle().LoadFailedFile(t.SliceHash); ok && count >= 1 {
			err = errors.New("cache file failed,remote miner offline or refused")
			tickets.Delete(t.BID)
			return slicePath, resp.NewError(500, errors.Wrap(err, "download service error"))
		}
		progress, ect := cache.DownloadProgressBar(t.FileHash, t.SliceHash, t.Size)
		slicePath = fmt.Sprintf(
			"file %s is being cached %2.2f %%,it will take about %d s",
			t.SliceHash, progress*100, ect,
		)
		tickets.Delete(t.BID)
		return slicePath, resp.NewError(0, nil)
	}
	slicePath = path.Join(cache.FilesDir, t.FileHash, t.SliceHash)
	if _, err := os.Stat(slicePath); err != nil {
		tickets.Delete(t.BID)
		return slicePath, resp.NewError(500, errors.Wrap(err, "download service error"))
	}
	return slicePath, nil
}

func PraseTicketByBID(hash, bid string) (Ticket, error) {
	var ticket Ticket
	b, err := types.HexDecodeString(hash)
	if err != nil {
		return ticket, errors.Wrap(err, "prase ticket error")
	}
	//test chain
	bill, err := chain.GetTestChainCli().GetBill(types.NewHash(b), bid)
	if err != nil {
		return ticket, errors.Wrap(err, "prase ticket error")
	}
	fmeta, err := chain.GetChainCli().GetFileMetaInfo(bill.FileHash)
	if err != nil {
		return ticket, errors.Wrap(err, "prase ticket error")
	}
	var size uint64
	shash := ""
	for _, block := range fmeta.BlockInfo {
		if string(block.BlockId[:])[36:] == bill.SliceHash {
			size = uint64(block.BlockSize)
			shash = string(block.BlockId[:])
			break
		}
	}
	if bill.Amount.Int64() != int64(size*config.GetConfig().BytePrice) {
		return ticket, errors.Wrap(errors.New("bad amount"), "prase ticket error")
	}
	ticket.BID = bill.BID
	ticket.Account = bill.Account
	ticket.FileHash = bill.FileHash
	ticket.SliceHash = shash
	ticket.Expires = bill.Expires
	ticket.Size = size
	return ticket, nil
}

func ticketBeUsed(bid string, exp time.Time) bool {
	if t, ok := tickets.LoadOrStore(bid, exp); ok {
		if exp := t.(time.Time); time.Since(exp) >= 0 {
			tickets.Delete(bid)
		}
		return true
	}
	return false
}

func OrdersCleanServer() {
	for range time.NewTicker(TAB_FLASH_TIME).C {
		tickets.Range(func(key, value any) bool {
			exp := value.(time.Time)
			if time.Since(exp) >= 0 {
				tickets.Delete(key)
			}
			return true
		})
	}
}
