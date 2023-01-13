package trans

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
)

var sendFileBufPool = &sync.Pool{
	New: func() interface{} {
		return make([]byte, SIZE_1MiB)
	},
}

type MsgFile struct {
	Token    string `json:"token"`
	RootHash string `json:"roothash"`
	FileHash string `json:"filehash"`
	FileSize int64  `json:"filesize"`
	Lastfile bool   `json:"lastfile"`
	Data     []byte `json:"data"`
}

func FileReq(conn net.Conn, token, fid string, fpath string, fsize int64, lastfile bool) error {
	var (
		err     error
		num     int
		total   int64
		tempBuf []byte
		msgHead IMessage
		fs      *os.File
		message = MsgFile{
			Token:    token,
			RootHash: fid,
			FileHash: "",
			FileSize: fsize,
			Lastfile: lastfile,
			Data:     nil,
		}
		dp       = NewDataPack()
		headData = make([]byte, dp.GetHeadLen())
	)

	readBuf := sendFileBufPool.Get().([]byte)
	defer func() {
		sendFileBufPool.Put(readBuf)
		if fs != nil {
			fs.Close()
		}
	}()

	fs, err = os.Open(fpath)
	if err != nil {
		return err
	}

	message.FileHash = filepath.Base(fpath)

	for {
		num, err = fs.Read(readBuf)
		if err != nil && err != io.EOF {
			return err
		}
		if num == 0 {
			break
		}
		total += int64(num)
		if total >= fsize {
			message.Data = readBuf[:(SIZE_1MiB + fsize - total)]
		} else {
			message.Data = readBuf[:num]
		}
		tempBuf, err = json.Marshal(&message)
		if err != nil {
			return err
		}

		//send auth message
		tempBuf, _ = dp.Pack(NewMsgPackage(Msg_File, tempBuf))
		_, err = conn.Write(tempBuf)
		if err != nil {
			return err
		}

		//read head
		_, err = io.ReadFull(conn, headData)
		if err != nil {
			return err
		}

		msgHead, err = dp.Unpack(headData)
		if err != nil {
			return err
		}

		if msgHead.GetMsgID() == Msg_OK_FILE {
			return nil
		}

		if msgHead.GetMsgID() != Msg_OK {
			return fmt.Errorf("send file error")
		}
		if total >= fsize {
			return nil
		}
	}

	return err
}

type MsgDown struct {
	Token     string `json:"token"`
	SliceHash string `json:"slicehash"`
	FileSize  int64  `json:"filesize"`
	Index     uint32 `json:"index"`
}

func DownReq(conn net.Conn, token, fpath string, fsize int64) error {
	var (
		err     error
		tempBuf []byte
		num     int
		msgHead IMessage
		fs      *os.File
		message = MsgDown{
			Token:     token,
			SliceHash: "",
			FileSize:  fsize,
			Index:     0,
		}
		dp       = NewDataPack()
		headData = make([]byte, dp.GetHeadLen())
	)

	readBuf := sendFileBufPool.Get().([]byte)
	defer func() {
		sendFileBufPool.Put(readBuf)
		if fs != nil {
			fs.Close()
		}
	}()

	fs, err = os.OpenFile(fpath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}
	fstat, _ := fs.Stat()
	message.Index = uint32(fstat.Size())

	message.SliceHash = filepath.Base(fpath)

	for {
		tempBuf, err = json.Marshal(&message)
		if err != nil {
			return err
		}

		//send message
		tempBuf, _ = dp.Pack(NewMsgPackage(Msg_Down, tempBuf))
		_, err = conn.Write(tempBuf)
		if err != nil {
			return err
		}

		//read head
		_, err = io.ReadFull(conn, headData)
		if err != nil {
			return err
		}

		msgHead, err = dp.Unpack(headData)
		if err != nil {
			return err
		}

		if msgHead.GetMsgID() == Msg_OK {
			if msgHead.GetDataLen() > 0 {
				num, err = io.ReadAtLeast(conn, readBuf, int(msgHead.GetDataLen()))
				if err != nil {
					return err
				}
				fs.Write(readBuf[:num])
				fs.Sync()
			}
		} else {
			return fmt.Errorf("read file error")
		}

		fstat, _ = fs.Stat()
		if fstat.Size() >= fsize {
			return nil
		}
		message.Index = uint32(fstat.Size())
	}
}
