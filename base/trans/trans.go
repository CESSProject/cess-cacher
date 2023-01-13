package trans

import "time"

const (
	Msg_Ping     = 100
	Msg_Auth     = 101
	Msg_File     = 102
	Msg_Down     = 103
	Msg_Confirm  = 104
	Msg_Progress = 105
)

const (
	Msg_OK        = 200
	Msg_OK_FILE   = 201
	Msg_ClientErr = 400
	Msg_Forbidden = 403
	Msg_ServerErr = 500
)

const (
	SIZE_1KiB  = 1024
	SIZE_1MiB  = 1024 * SIZE_1KiB
	SIZE_1GiB  = 1024 * SIZE_1MiB
	SIZE_SLICE = 512 * SIZE_1MiB
)

const (
	// Tcp message interval
	TCP_Message_Interval = time.Duration(time.Millisecond * 10)

	Tcp_Dial_Timeout = time.Duration(time.Second * 5)

	TCP_MaxPacketSize = SIZE_1MiB * 2
)

// Message
type Message struct {
	DataLen uint32
	ID      uint32
	Data    []byte
}

type IMessage interface {
	GetDataLen() uint32
	GetMsgID() uint32
	GetData() []byte

	SetMsgID(uint32)
	SetData([]byte)
	SetDataLen(uint32)
}

// NewMsgPackage
func NewMsgPackage(ID uint32, data []byte) *Message {
	return &Message{
		DataLen: uint32(len(data)),
		ID:      ID,
		Data:    data,
	}
}

func (msg *Message) Init(ID uint32, data []byte) {
	msg.ID = ID
	msg.Data = data
	msg.DataLen = uint32(len(data))
}

// GetDataLen
func (msg *Message) GetDataLen() uint32 {
	return msg.DataLen
}

// GetMsgID
func (msg *Message) GetMsgID() uint32 {
	return msg.ID
}

// GetData
func (msg *Message) GetData() []byte {
	return msg.Data
}

// SetDataLen
func (msg *Message) SetDataLen(len uint32) {
	msg.DataLen = len
}

// SetMsgID
func (msg *Message) SetMsgID(msgID uint32) {
	msg.ID = msgID
}

// SetData
func (msg *Message) SetData(data []byte) {
	msg.Data = data
}
