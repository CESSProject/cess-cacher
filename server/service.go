package server

import (
	"cess-cacher/base/cache"
	"cess-cacher/base/chain"
	"cess-cacher/utils"
	"fmt"
	"log"
	"path"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type Order struct {
	Account string
	Size    uint64
	Silces  map[string]struct{}
	Used    map[string]struct{}
}

const TIMEOUT = 3 * time.Second

var orders *sync.Map

func DownloadService(filehash, index string, sign []byte) (string, Error) {
	var (
		slicePath string
		order     Order
		err       error
		orderChan chan Order
	)
	if ch, ok := orders.Load(filehash); ok {
		orderChan = ch.(chan Order)
		if order, ok = getOrder(orderChan); !ok {
			err = errors.Wrap(errors.New("busy business"), "download service error")
			return slicePath, NewError(400, err)
		}
	} else {
		order, err = getOrderFromChain(filehash)
		if err != nil {
			return slicePath, NewError(400, errors.Wrap(err, "download service error"))
		}
		orderChan = make(chan Order, 1)
	}
	defer func() {
		if len(order.Used) == len(order.Silces) {
			orders.Delete(filehash)
		} else {
			orderChan <- order
			orders.Store(filehash, orderChan)
		}
	}()
	if _, ok := order.Silces[index]; !ok {
		err = errors.Wrap(errors.New("bad index"), "download service error")
		return slicePath, NewError(400, err)
	}
	if !utils.VerifySign(order.Account, []byte(filehash), sign) {
		err = errors.Wrap(errors.New("bad sign"), "download service error")
		return slicePath, NewError(400, err)
	}
	if ok, err := cache.GetCacheHandle().HitOrLoad(filehash); !ok {
		if err != nil {
			return slicePath, NewError(500, errors.Wrap(err, "download service error"))
		}
		duration := order.Size / uint64(cache.GetNetInfo().Upload)
		slicePath = fmt.Sprintf("The file %s is being cached. Please wait about %d seconds", filehash, duration)
		return slicePath, NewError(0, nil)
	}
	if _, ok := order.Used[index]; ok {
		err = errors.Wrap(errors.New("slice already been used"), "download service error")
		return slicePath, NewError(400, err)
	}
	order.Used[index] = struct{}{}
	slicePath = path.Join(cache.FilesDir, filehash, index)
	return slicePath, nil
}

func getOrder(ch <-chan Order) (Order, bool) {
	for {
		select {
		case order := <-ch:
			return order, true
		case <-time.After(TIMEOUT):
			return Order{}, false
		}
	}
}

func getOrderFromChain(filehash string) (Order, error) {
	var order Order
	fmeta, err := chain.Cli.GetFileMetaInfo(filehash)
	if err != nil {
		return order, err
	}
	order.Size = uint64(fmeta.Size)
	for _, v := range fmeta.Backups[0].Slice_info {
		//v.Slice_hash
		log.Println(v)
	}
	return order, nil
}
