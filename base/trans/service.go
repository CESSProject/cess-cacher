package trans

import (
	"cess-cacher/base/chain"
	"cess-cacher/base/trans/tcp"
	"cess-cacher/config"
	"cess-cacher/logger"
	"cess-cacher/utils"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/CESSProject/go-keyring"
	"github.com/pkg/errors"
)

func DownloadFile(fid, filesDir string) error {
	// file meta info
	fmeta, err := chain.GetChainCli().GetFileMetaInfo(fid)
	if err != nil {
		err = errors.Wrap(err, "get file meta info error")
		return errors.Wrap(err, "download file error")
	}

	if _, err := os.Stat(filesDir); err != nil {
		if err = os.MkdirAll(filesDir, 0755); err != nil {
			return errors.Wrap(err, "download file error")
		}
	}
	r := len(fmeta.BlockInfo) / 3
	d := len(fmeta.BlockInfo) - r
	down_count := 0
	for i := 0; i < len(fmeta.BlockInfo); i++ {
		fname := filepath.Join(filesDir, string(fmeta.BlockInfo[i].BlockId[:]))
		if len(fmeta.BlockInfo) == 1 {
			fname = fname[:(len(fname) - 4)]
		}
		mip := fmt.Sprintf("%d.%d.%d.%d:%d",
			fmeta.BlockInfo[i].MinerIp.Value[0],
			fmeta.BlockInfo[i].MinerIp.Value[1],
			fmeta.BlockInfo[i].MinerIp.Value[2],
			fmeta.BlockInfo[i].MinerIp.Value[3],
			fmeta.BlockInfo[i].MinerIp.Port,
		)
		err = downloadFromStorage(fname, int64(fmeta.BlockInfo[i].BlockSize), mip, filesDir)
		if err != nil {
			logger.Uld.Sugar().Error(errors.Wrap(err, "download file error"))
		} else {
			down_count++
		}
		if down_count >= d {
			break
		}
	}
	return nil
}

// Download files from cess storage service
func downloadFromStorage(fpath string, fsize int64, mip string, dir string) error {
	fsta, err := os.Stat(fpath)
	if err == nil {
		if fsta.Size() == fsize {
			return nil
		} else {
			os.Remove(fpath)
		}
	}

	msg := utils.GetRandomcode(16)

	kr, _ := keyring.FromURI(config.GetConfig().AccountSeed, keyring.NetSubstrate{})
	// sign message
	sign, err := kr.Sign(kr.SigningContext([]byte(msg)))
	if err != nil {
		return err
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", mip)
	if err != nil {
		return err
	}

	conTcp, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return err
	}
	srv := tcp.NewClient(tcp.NewTcp(conTcp), dir, nil)
	pubkey, err := utils.DecodePublicKeyOfCessAccount(config.GetConfig().AccountID)
	if err != nil {
		return err
	}
	return srv.RecvFile(filepath.Base(fpath), fsize, pubkey, []byte(msg), sign[:])
}
