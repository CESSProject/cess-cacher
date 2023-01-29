package service

import (
	"cess-cacher/base/cache"
	"cess-cacher/base/chain"
	resp "cess-cacher/server/response"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

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
	if ok, err := cache.GetCacheHandle().HitOrLoad(t.FileHash); !ok {
		if err != nil {
			tickets.Delete(t.BID)
			return slicePath, resp.NewError(500, errors.Wrap(err, "download service error"))
		}
		duration := t.Size / uint64(cache.GetNetInfo().Upload)
		slicePath = fmt.Sprintf("file %s is being cached,about %d s", t.FileHash, duration)
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

func PraseTicketByBID(bid string) (Ticket, error) {
	var ticket Ticket
	bill, err := chain.GetChainCli().GetBill(bid)
	if err != nil {
		return ticket, errors.Wrap(err, "prase ticket error")
	}
	fmeta, err := chain.GetChainCli().GetFileMetaInfo(bill.FileHash)
	if err != nil {
		return ticket, errors.Wrap(err, "prase ticket error")
	}
	ticket.BID = bid
	ticket.Account = bill.Account
	ticket.FileHash = bill.FileHash
	ticket.SliceHash = bill.SliceHash
	ticket.Expires = bill.Expires
	ticket.Size = uint64(fmeta.Size)
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
