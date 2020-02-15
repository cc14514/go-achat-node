package chat

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
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
	mfn         = map[string]func(*Req) *Rsp{
		"auth": func(req *Req) *Rsp {
			if pwd == "" || (len(req.Params) == 1 && req.Params[0] == pwd) {
				s := sha1.New()
				now := time.Now()
				s.Write([]byte(fmt.Sprintf("%s%d", pwd, now.UnixNano())))
				h := s.Sum(nil)
				token := hex.EncodeToString(h)
				tokenmap[token] = now.Unix()
				return NewRsp(req.Id, token, nil)
			} else {
				return NewRsp(req.Id, nil, &RspError{
					Code:    "1002",
					Message: "auth_fail",
				})
			}
		},

		"sendmsg": func(req *Req) *Rsp {
			to := req.Params[0]
			content := strings.Join(req.Params[1:], " ")
			msg := NewNormalMessage(chatservice.GetMyid(), JID(to), content)
			if err := chatservice.SendMsg(msg); err != nil {
				return NewRsp(req.Id, nil, &RspError{
					Code:    "1003",
					Message: err.Error(),
				})
			} else {
				return NewRsp(req.Id, "success", nil)
			}
		},

		"myid": func(req *Req) *Rsp { return NewRsp(req.Id, chatservice.myid, nil) },

		"conns": func(req *Req) *Rsp {
			p2pservice := chatservice.p2pservice
			s := time.Now()
			direct, relay, total := p2pservice.Peers()
			dpis := make([]peerinfo, 0)
			for _, id := range direct {
				if addrs, err := p2pservice.Findpeer(id); err == nil {
					dpis = append(dpis, peerinfo{id, addrs})
				}
			}
			rpis := make(map[string][]peerinfo)
			for rid, ids := range relay {
				pis := make([]peerinfo, 0)
				for _, id := range ids {
					if addrs, err := p2pservice.Findpeer(id); err == nil {
						pis = append(pis, peerinfo{id, addrs})
					}
				}
				rpis[rid] = pis
			}
			entity := struct {
				TimeUsed string
				Total    int
				Direct   []peerinfo
				Relay    map[string][]peerinfo
			}{time.Since(s).String(), total, dpis, rpis}
			return NewRsp(req.Id, entity, nil)
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
			fn(req).WriteTo(w)
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
			ws.Write(NewSysMessage("",
				Attr{Key: "method", Val: req.Method},
				Attr{Key: "error", Val: "error_token"}).Json())
			return
		}
		ws.Write(NewSysMessage("",
			Attr{Key: "method", Val: req.Method},
			Attr{Key: "result", Val: "success"}).Json())
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
