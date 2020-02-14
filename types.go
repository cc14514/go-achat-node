package chat

import (
	"encoding/json"
	"fmt"
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

	MsgHandle func(service *ChatService, msg *Message)

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
	MessageBag struct {
		Messages MessageList
	}
	MessageList []*Message

	CleanMsg struct {
		Jid JID
		Ids []string
	}
)

func (m MessageList) Len() int {
	return len(m)
}

func (m MessageList) Less(i, j int) bool {
	return m[i].Envelope.Ct < m[j].Envelope.Ct
}

func (m MessageList) Swap(i, j int) {
	m[i], m[j] = m[j], m[i]
}

func (c *MessageBag) FromBytes(data []byte) (Msg, error) {
	return c, amino.UnmarshalBinaryLengthPrefixed(data, c)
}

func (c *MessageBag) FromReader(r io.Reader) (Msg, error) {
	_, err := amino.UnmarshalBinaryLengthPrefixedReader(r, c, MAX_PKG)
	return c, err
}

func (c *MessageBag) Bytes() []byte {
	data, _ := amino.MarshalBinaryLengthPrefixed(c)
	return data
}
func (c *MessageBag) FromJson(data []byte) (Msg, error) {
	err := amino.UnmarshalJSON(data, c)
	return c, err
}

func (c *MessageBag) Json() []byte {
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
	return newMessage(from, to, "", content, NormalMsg, attr...)
}

func NewSysMessage(id string, attr ...Attr) *Message {
	return newMessage("", "", id, "", SysMsg, attr...)
}

func newMessage(from, to JID, id, content string, mt MsgType, attr ...Attr) *Message {
	envelope := Envelope{
		Id:   uuid.New().String(),
		Type: mt,
		Ack:  NoACK,
		Ct:   time.Now().Unix(),
	}
	if id != "" {
		envelope.Id = id
	}
	if from != "" {
		envelope.From = from
	}
	if to != "" {
		envelope.To = to
	}
	m := &Message{
		Envelope: envelope,
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

// ====================>
// RPC
// ====================>

type (
	Req struct {
		Id     string   `json:"id,omitempty"`
		Token  string   `json:"token"`
		Method string   `json:"method"`
		Params []string `json:"params,omitempty"`
	}
	/*
		{
			"result": "Hello JSON-RPC",
		    "error": null,
			"id": 1
		}
	*/
	RspError struct {
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}
	Rsp struct {
		Result interface{} `json:"result,omitempty"`
		Error  *RspError   `json:"error,omitempty"`
		Id     string
	}

	RspMessage struct {
		Result *Message  `json:"result,omitempty"`
		Error  *RspError `json:"error,omitempty"`
		Id     string
	}
)

func NewRsp(id string, result interface{}, err *RspError) *Rsp {
	rsp := &Rsp{Id: id}
	if err != nil {
		rsp.Error = err
	} else {
		rsp.Result = result
	}
	return rsp
}

func (r *Rsp) WriteTo(rw io.Writer) error {
	_, err := rw.Write(r.Bytes())
	return err
}

func (r *Req) WriteTo(rw io.Writer) error {
	fmt.Println("request =>", string(r.Bytes()))
	_, err := rw.Write(r.Bytes())
	return err
}

func (r *Rsp) String() string {
	return string(r.Bytes())
}

func (r *RspMessage) FromBytes(data []byte) (*RspMessage, error) {
	err := json.Unmarshal(data, r)
	return r, err
}

func (r *Rsp) FromBytes(data []byte) (*Rsp, error) {
	err := json.Unmarshal(data, r)
	return r, err
}

func (r *Rsp) Bytes() []byte {
	buf, _ := json.Marshal(r)
	return buf
}

func NewReq(token, m string, p []string) *Req {
	return &Req{uuid.New().String(), token, m, p}
}

func (r *Req) String() string {
	return string(r.Bytes())
}

func (r *Req) FromBytes(data []byte) *Req {
	json.Unmarshal(data, r)
	return r
}

func (r *Req) Bytes() []byte {
	buf, _ := json.Marshal(r)
	return buf
}

// <====================
// RPC
// <====================
