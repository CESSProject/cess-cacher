package service

import (
	"cess-cacher/base/cache"
	"cess-cacher/base/chain"
	resp "cess-cacher/server/response"
	"cess-cacher/utils"
	"fmt"
	"path"
	"sync"
	"time"

	"github.com/pkg/errors"
)

type Order struct {
	Account string
	Size    uint64
	Silces  map[int]string
	Used    map[int]struct{}
}

const TIMEOUT = 3 * time.Second

var orders *sync.Map
var olk sync.Mutex

func InitOrders() {
	if orders == nil {
		orders = new(sync.Map)
	}
}

func DownloadService(fhash string, index int) (string, resp.Error) {
	var slicePath string
	orderChan, err := getOrderChan(fhash)
	if err != nil {
		return slicePath, resp.NewError(500, errors.Wrap(err, "download service error"))
	}
	order, ok := getOrder(orderChan)
	if !ok {
		err = errors.Wrap(errors.New("busy business"), "download service error")
		return slicePath, resp.NewError(500, err)
	}
	defer func() {
		if len(order.Used) == len(order.Silces) {
			orders.Delete(fhash)
		} else {
			orderChan <- order
		}
	}()
	if _, ok := order.Silces[index]; !ok {
		err = errors.Wrap(errors.New("bad index"), "download service error")
		return slicePath, resp.NewError(400, err)
	}
	if ok, err := cache.GetCacheHandle().HitOrLoad(fhash); !ok {
		if err != nil {
			return slicePath, resp.NewError(500, errors.Wrap(err, "download service error"))
		}
		duration := order.Size / uint64(cache.GetNetInfo().Upload)
		slicePath = fmt.Sprintf("The file %s is being cached. Please wait about %d seconds", fhash, duration)
		return slicePath, resp.NewError(0, nil)
	}
	if _, ok := order.Used[index]; ok {
		err = errors.Wrap(errors.New("slice already been used"), "download service error")
		return slicePath, resp.NewError(400, err)
	}
	order.Used[index] = struct{}{}
	slicePath = path.Join(cache.FilesDir, fhash, order.Silces[index])
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

func getOrderChan(hash string) (chan Order, error) {
	var ch chan Order
	if v, ok := orders.Load(hash); ok {
		ch = v.(chan Order)
		return ch, nil
	}
	order, err := getOrderFromChain(hash)
	if err != nil {
		return nil, err
	}
	olk.Lock()
	if _, ok := orders.Load(hash); !ok {
		ch = make(chan Order, 1)
		ch <- order
		orders.Store(hash, ch)
	}
	olk.Unlock()
	return ch, nil
}

func getOrderFromChain(filehash string) (Order, error) {
	var order Order
	fmeta, err := chain.Cli.GetFileMetaInfo(filehash)
	if err != nil {
		return order, err
	}
	order.Account, err = utils.EncodePublicKeyAsCessAccount(fmeta.UserBriefs[0].User[:])
	if err != nil {
		return order, err
	}
	order.Silces = make(map[int]string)
	order.Used = make(map[int]struct{})
	order.Size = uint64(fmeta.Size)
	for i, v := range fmeta.Backups[0].Slice_info {
		order.Silces[i] = string(v.Slice_hash[:])
	}
	return order, nil
}
