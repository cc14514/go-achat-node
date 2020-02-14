package chat

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

var (
	chatservice *ChatService
	pwd         string
	rpcport     int
	tokenmap    = make(map[string]int64)
	mfn         = map[string]func(io.Writer, *Req) error{
		"auth": func(ws io.Writer, req *Req) error {
			var err error
			if pwd == "" || (len(req.Params) == 1 && req.Params[0] == pwd) {
				s := sha1.New()
				now := time.Now()
				s.Write([]byte(fmt.Sprintf("%s%d", pwd, now.UnixNano())))
				h := s.Sum(nil)
				token := hex.EncodeToString(h)
				tokenmap[token] = now.Unix()
				NewRsp(req.Id, token, nil).WriteTo(ws)
			} else {
				err = errors.New("auth_fail")
				NewRsp(req.Id, nil, &RspError{
					Code:    "1002",
					Message: err.Error(),
				}).WriteTo(ws)
			}
			return err
		},

		"sendmsg": func(ws io.Writer, req *Req) error {
			to := req.Params[0]
			content := strings.Join(req.Params[1:], " ")
			msg := NewNormalMessage(chatservice.GetMyid(), JID(to), content)
			if err := chatservice.SendMsg(msg); err != nil {
				NewRsp(req.Id, nil, &RspError{
					Code:    "1003",
					Message: err.Error(),
				}).WriteTo(ws)
				return err
			} else {
				NewRsp(req.Id, "success", nil).WriteTo(ws)
			}
			return nil
		},
	}
)

func StartRPC(_pwd string, _rpcport int, _chatservice *ChatService) {
	chatservice, pwd, rpcport = _chatservice, _pwd, _rpcport
	http.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("ws_error", "err", err)
			return
		}
		req := new(Req).FromBytes(data)
		if fn, ok := mfn[req.Method]; ok {
			if req.Method != "auth" && tokenmap[req.Token] <= 0 {
				NewRsp(req.Id, nil, &RspError{
					Code:    "1001",
					Message: "error token , please relogin .",
				}).WriteTo(w)
			}
			fn(w, req)
		} else {
			NewRsp(req.Id, nil, &RspError{
				Code:    "1003",
				Message: "method_not_support",
			}).WriteTo(w)
		}
	})

	http.Handle("/chat", websocket.Handler(func(ws *websocket.Conn) {
		defer ws.Close()
		var err error
		var in string
		if err = websocket.Message.Receive(ws, &in); err != nil {
			log.Println("ws_error", "err", err)
			return
		}
		req := new(Req).FromBytes([]byte(in))
		_, ok := tokenmap[req.Token]
		if !ok {
			NewRsp(req.Id, nil, &RspError{Code: "1003", Message: "error_token"}).WriteTo(ws)
			return
		}
		ws.Write(NewSysMessage("", Attr{
			Key: "method",
			Val: req.Method,
		}, Attr{
			Key: "result",
			Val: "success",
		}).Json())
		if ml, err := chatservice.QueryMsg(); err == nil {
			ids := make([]string, 0)
			for _, m := range ml.Messages {
				ids = append(ids, m.Envelope.Id)
				ws.Write(m.Json())
			}
			chatservice.CleanMsg(ids)
		}
		fid := chatservice.AppendHandleMsg(func(service *ChatService, msg *Message) {
			ws.Write(msg.Json())
		})
		defer chatservice.DropHandleMsg(fid)
		for {
			if err = websocket.Message.Receive(ws, &in); err != nil {
				return
			}
			log.Println("==ws==>", in)
		}
	}))
	if err := http.ListenAndServe(fmt.Sprintf(":%d", rpcport), nil); err != nil {
		log.Println("listen_error", "err", err)
		os.Exit(1)
	}
}
