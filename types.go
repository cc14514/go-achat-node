package chat

import (
	"github.com/google/uuid"
	"github.com/tendermint/go-amino"
	"io"
	"time"
)

/*
值	含义	描述
0	打开会话	成功创建连接后，5秒内必须要发送此类型请求打开会话，否则将被断开
1	普通聊天	此类型消息用于1对1聊天场景
2	群聊	此类型消息用于1对多聊天场景
3	状态同步	可以实现消息的事件同步，例如：已送达、已读 等
4	系统消息	服务端返回给客户端的通知
5	RTP拨号	P2P服务拨号
6	暂无	预留
7	订阅	发布/订阅
*/

const (
	NormalMsg MsgType = iota + 1
	GroupMsg
	SyncMsg
	SysMsg
	RtpMsg
	BakMsg
	PubsubMsg
)

const (
	NoACK byte = iota
	ACK
)

const MAX_PKG = 2048

/*
  {
    "envelope":{
      "id":"UUID，要求必须唯一"
      "from":"发送人JID",
      "to":"接收人JID",
      "type":"int型，含义是 消息类型",
      "ack":"int型，0 或空是不必响应，1 必须响应",
      "ct":"13位时间戳，由服务器来补充此值",
      "pwd":"只有当 type=0 时，即 opensession 时，才会使用此属性",
      "gid":"群ID，只在 type=2 时会用到此属性"
    },
    "vsn":"消息版本(预留属性)",
    "payload":{
        "attrs":{"k":"v",...},
        "content":"..."
    }
  }
*/

//omitempty
type (
	Msg interface {
		FromBytes(data []byte) (Msg, error)
		FromReader(r io.Reader) (Msg, error)
		Bytes() []byte
		FromJson(data []byte) (Msg, error)
		Json() []byte
	}

	MsgType  int
	JID      string
	Envelope struct {
		Id   string  `json:"id,omitempty"`
		From JID     `json:"from,omitempty"`
		To   JID     `json:"to,omitempty"`
		Type MsgType `json:"type"`
		Ack  byte    `json:"ack,omitempty"`
		Ct   int64   `json:"ct,omitempty"`
		Gid  JID     `json:"gid,omitempty"`
	}
	Attr struct {
		Key string `json:"key"`
		Val string `json:"val"`
	}
	Payload struct {
		Attrs   []Attr `json:"attrs,omitempty"`
		Content string `json:"content,omitempty"`
	}
	Message struct {
		Envelope Envelope `json:"envelope"`
		Payload  Payload  `json:"payload,omitempty"`
		Vsn      string   `json:"vsn,omitempty"`
	}
	Messages []*Message
)

func (c *Messages) FromBytes(data []byte) (Msg, error) {
	return c, amino.UnmarshalBinaryLengthPrefixed(data, c)
}

func (c *Messages) FromReader(r io.Reader) (Msg, error) {
	_, err := amino.UnmarshalBinaryLengthPrefixedReader(r, c, MAX_PKG)
	return c, err
}

func (c *Messages) Bytes() []byte {
	data, _ := amino.MarshalBinaryLengthPrefixed(c)
	return data
}
func (c *Messages) FromJson(data []byte) (Msg, error) {
	err := amino.UnmarshalJSON(data, c)
	return c, err
}

func (c *Messages) Json() []byte {
	d, _ := amino.MarshalJSON(c)
	return d
}

func NewJID(id, mailbox string) JID {
	if mailbox != "" {
		return JID(id + mailbox)
	}
	return JID(id)
}

func NewNormalMessage(from, to JID, content string, attr ...Attr) *Message {
	m := &Message{
		Envelope: Envelope{
			Id:   uuid.New().String(),
			From: from,
			To:   to,
			Type: NormalMsg,
			Ack:  NoACK,
			Ct:   time.Now().Unix(),
		},
		Payload: Payload{
			Content: content,
			Attrs:   attr,
		},
		Vsn: "0.0.2",
	}
	return m
}

func (i JID) Peerid() string {
	if len(i) < 53 {
		return ""
	}
	return string(i)[:53]
}

func (i JID) Mailid() string {
	if len(i) < 53*2 {
		return ""
	}
	return string(i)[53 : 53*2]
}

func (c *Message) FromBytes(data []byte) (Msg, error) {
	return c, amino.UnmarshalBinaryLengthPrefixed(data, c)
}

func (c *Message) FromReader(r io.Reader) (Msg, error) {
	_, err := amino.UnmarshalBinaryLengthPrefixedReader(r, c, MAX_PKG)
	return c, err
}

func (c *Message) Bytes() []byte {
	data, _ := amino.MarshalBinaryLengthPrefixed(c)
	return data
}
func (c *Message) FromJson(data []byte) (Msg, error) {
	err := amino.UnmarshalJSON(data, c)
	return c, err
}

func (c *Message) Json() []byte {
	d, _ := amino.MarshalJSON(c)
	return d
}
