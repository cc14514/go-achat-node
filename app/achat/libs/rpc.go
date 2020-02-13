package libs

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	chat "github.com/cc14514/go-alibp2p-chat"
	"github.com/google/uuid"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

var (
	pwd     string
	rpcport int
	//Endflag   = byte(1)
	Instructs = []string{
		"opensession",
		"close",
		"myid",
		"conns",
		"exit",
		"help",
	}
	help = func() string {
		return `
-------------------------------------------------------------------------
# 当前 shell 支持以下指令
-------------------------------------------------------------------------
opensession jid 打开一个会话，聊天
myid 获取当前节点信息
conns 获取当前网络连接信息
exit 退出 shell

`
	}
	tokenmap = make(map[string]int64)
)

type (
	peerinfo struct {
		ID    string
		Addrs []string
	}
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

func (r *Rsp) WriteTo(rw io.ReadWriter) error {
	_, err := rw.Write(r.Bytes())
	return err
}

func (r *Rsp) String() string {
	return string(r.Bytes())
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

func StartRPC(_pwd string, _rpcport int, chatservice *chat.ChatService) {
	pwd, rpcport = _pwd, _rpcport
	http.Handle("/chat", websocket.Handler(func(ws *websocket.Conn) {
		var err error
		for {
			var reply string
			if err = websocket.Message.Receive(ws, &reply); err != nil {
				break
			}
			req := new(Req).FromBytes([]byte(reply))
			if req.Method != "auth" && tokenmap[req.Token] <= 0 {
				NewRsp(req.Id, nil, &RspError{
					Code:    "1001",
					Message: "error token , please relogin .",
				}).WriteTo(ws)
			}
			switch req.Method {
			case "auth":
				if len(req.Params) == 1 && req.Params[0] == pwd {
					s := sha1.New()
					now := time.Now()
					s.Write([]byte(fmt.Sprintf("%s%d", pwd, now.UnixNano())))
					h := s.Sum(nil)
					token := hex.EncodeToString(h)
					tokenmap[token] = now.Unix()
					NewRsp(req.Id, "token "+token, nil).WriteTo(ws)
				} else {
					NewRsp(req.Id, nil, &RspError{
						Code:    "1002",
						Message: "auth_fail",
					}).WriteTo(ws)
				}
				ws.Close()
			case "sendmsg":
				to := req.Params[0]
				content := strings.Join(req.Params[1:], " ")
				msg := chat.NewNormalMessage(chatservice.GetMyid(), chat.JID(to), content)
				if err := chatservice.SendMsg(msg); err != nil {
					NewRsp(req.Id, nil, &RspError{
						Code:    "1003",
						Message: err.Error(),
					}).WriteTo(ws)
				} else {
					NewRsp(req.Id, "success", nil).WriteTo(ws)
				}
			default:
				NewRsp(req.Id, help(), nil).WriteTo(ws)

			}
		}
	}))
	if err := http.ListenAndServe(fmt.Sprintf(":%d", rpcport), nil); err != nil {
		fmt.Println("error", err)
		os.Exit(1)
	}
}
