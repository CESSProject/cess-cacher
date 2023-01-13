package trans

import (
	"cess-cacher/base/chain"
	"cess-cacher/config"
	"cess-cacher/logger"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// Download files from cess storage service
func DownloadFile(fid, dir string) error {
	// file meta info
	fmeta, err := chain.Cli.GetFileMetaInfo(fid)
	if err != nil {
		logger.Uld.Sugar().Errorf("get file %s metadata error:%v\n", fid, err)
		if err.Error() == chain.ERR_Empty {
			logger.Uld.Sugar().Errorf("file %s not found on chain.\n", fid)
		}
		return errors.Wrap(err, "download file from miner error")
	}

	if string(fmeta.State) != chain.FILE_STATE_ACTIVE {
		return errors.Wrap(errors.New("file is not ready"), "download file from miner error")
	}
	var fsize int64 = SIZE_SLICE
	for j := 0; j < len(fmeta.Backups[0].Slice_info); j++ {
		for i := 0; i < len(fmeta.Backups); i++ {
			// Download the file from the scheduler service
			fname := filepath.Join(dir, string(fmeta.Backups[i].Slice_info[j].Slice_hash[:]))
			mip := fmt.Sprintf("%d.%d.%d.%d:%d",
				fmeta.Backups[i].Slice_info[j].Miner_ip.Value[0],
				fmeta.Backups[i].Slice_info[j].Miner_ip.Value[1],
				fmeta.Backups[i].Slice_info[j].Miner_ip.Value[2],
				fmeta.Backups[i].Slice_info[j].Miner_ip.Value[3],
				fmeta.Backups[i].Slice_info[j].Miner_ip.Port,
			)
			if (j + 1) == len(fmeta.Backups[i].Slice_info) {
				fsize = int64(fmeta.Size % SIZE_SLICE)
			}
			if err = DownloadFromStorage(fname, fsize, mip); err != nil {
				logger.Uld.Sugar().Errorf("downloading the %dth shard of the file %s error: %v", i, fid, err)
				if (i + 1) == len(fmeta.Backups) {
					return errors.Wrap(err, "download file from miner error")
				}
				continue
			}
		}
	}
	return nil
}

func DownloadFromStorage(fpath string, fsize int64, mip string) error {
	fsta, err := os.Stat(fpath)
	if err == nil {
		if fsta.Size() == fsize {
			return nil
		} else if fsta.Size() > fsize {
			os.Remove(fpath)
		}
	}

	conTcp, err := dialTcpServer(mip)
	if err != nil {
		logger.Uld.Sugar().Errorf("dial %v error: %v", mip, err)
		return err
	}

	token, err := AuthReq(conTcp, config.GetConfig().AccountSeed)
	if err != nil {
		logger.Uld.Sugar().Errorf("get request token error: %v", err)
		return err
	}

	return DownReq(conTcp, token, fpath, fsize)
}

func dialTcpServer(address string) (*net.TCPConn, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return nil, err
	}
	dialer := net.Dialer{Timeout: Tcp_Dial_Timeout}
	netCon, err := dialer.Dial("tcp", tcpAddr.String())
	if err != nil {
		return nil, err
	}
	conTcp, ok := netCon.(*net.TCPConn)
	if !ok {
		conTcp.Close()
		return nil, errors.New("network conversion failed")
	}
	return conTcp, nil
}

// func copyFile(src, dst string, length int64) error {
// 	srcfile, err := os.OpenFile(src, os.O_RDONLY, os.ModePerm)
// 	if err != nil {
// 		return err
// 	}
// 	defer srcfile.Close()
// 	dstfile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
// 	if err != nil {
// 		return err
// 	}
// 	defer dstfile.Close()

// 	var buf = make([]byte, 64*1024)
// 	var count int64
// 	for {
// 		n, err := srcfile.Read(buf)
// 		if err != nil && err != io.EOF {
// 			return err
// 		}
// 		if n == 0 {
// 			break
// 		}
// 		count += int64(n)
// 		if count < length {
// 			dstfile.Write(buf[:n])
// 		} else {
// 			tail := count - length
// 			if n >= int(tail) {
// 				dstfile.Write(buf[:(n - int(tail))])
// 			}
// 		}
// 	}

// 	return nil
// }
