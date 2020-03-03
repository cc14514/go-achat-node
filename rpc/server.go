package rpc

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/cc14514/go-alibp2p-chat"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/net/websocket"
)

var (
	chatservice *chat.ChatService
	pwd         string
	rpcport     int
	tokenmap    = make(map[string]int64)
	servicemap  = make(map[string]Service)
	serviceReg  = func(s Service) {
		servicemap[s.APIs().Namespace] = s
	}
	getRpcFn = func(method string) RpcFn {
		args := strings.Split(method, "_")
		if len(args) != 2 {
			return nil
		}
		ns, fn := args[0], args[1]
		if s, ok := servicemap[ns]; ok {
			return s.APIs().Api[fn]
		}
		return nil
	}
	fnReg = map[string]RpcFn{
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
			c, err := X2Str(req.Params[1:])
			if err != nil {
				return NewRsp(req.Id, nil, &RspError{Code: "1003", Message: err.Error()})
			}
			content := strings.Join(c, " ")
			msg := chat.NewNormalMessage(chatservice.GetMyid(), chat.JID(to.(string)), content)
			if err := chatservice.SendMsg(msg); err != nil {
				return NewRsp(req.Id, nil, &RspError{Code: "1003", Message: err.Error()})
			} else {
				return NewRsp(req.Id, "success", nil)
			}
		},

		"myid": func(req *Req) *Rsp { return NewRsp(req.Id, chatservice.GetMyid(), nil) },

		"conns": func(req *Req) *Rsp {
			p2pservice := chatservice.GetLibp2pService()
			s := time.Now()
			direct, relay, total := p2pservice.Peers()
			dpis := make([]Peerinfo, 0)
			for _, id := range direct {
				if addrs, err := p2pservice.Findpeer(id); err == nil {
					dpis = append(dpis, Peerinfo{id, addrs})
				}
			}
			rpis := make(map[string][]Peerinfo)
			for rid, ids := range relay {
				pis := make([]Peerinfo, 0)
				for _, id := range ids {
					if addrs, err := p2pservice.Findpeer(id); err == nil {
						pis = append(pis, Peerinfo{id, addrs})
					}
				}
				rpis[rid] = pis
			}
			entity := struct {
				TimeUsed string
				Total    int
				Direct   []Peerinfo
				Relay    map[string][]Peerinfo
			}{time.Since(s).String(), total, dpis, rpis}
			return NewRsp(req.Id, entity, nil)
		},
	}

	X2Str = func(il []interface{}) ([]string, error) {
		r := make([]string, 0)
		for _, s := range il {
			i, ok := s.(string)
			if !ok {
				return nil, errors.New("item not str")
			}
			r = append(r, i)
		}
		return r, nil
	}
	Str2X = func(sl []string) []interface{} {
		r := make([]interface{}, 0)
		for _, s := range sl {
			r = append(r, s)
		}
		return r
	}
)

func startService() {
	serviceReg(NewUserService(chatservice))
	serviceReg(NewGroupService(chatservice))

}

func StartRPC(_pwd string, _rpcport int, _chatservice *chat.ChatService) {
	chatservice, pwd, rpcport = _chatservice, _pwd, _rpcport
	http.HandleFunc("/rpc", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*") //允许访问所有域
		w.Header().Set("content-type", "application/json") //返回数据格式是json

		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.Println("ws_error", "err", err)
			return
		}
		req := new(Req).FromBytes(data)
		if req.Method != "auth" && tokenmap[req.Token] <= 0 {
			NewRsp(req.Id, nil, &RspError{
				Code:    "1001",
				Message: "error token , please relogin .",
			}).WriteTo(w)
		} else if fn, ok := fnReg[req.Method]; ok {
			fn(req).WriteTo(w)
		} else if _fn := getRpcFn(req.Method); _fn != nil {
			_fn(req).WriteTo(w)
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
			ws.Write(chat.NewSysMessage("",
				chat.Attr{Key: "method", Val: req.Method},
				chat.Attr{Key: "error", Val: "error_token"}).Json())
			return
		}
		ws.Write(chat.NewSysMessage("",
			chat.Attr{Key: "method", Val: req.Method},
			chat.Attr{Key: "result", Val: "success"}).Json())
		if ml, err := chatservice.QueryMsg(); err == nil {
			ids := make([]string, 0)
			for _, m := range ml.Messages {
				ids = append(ids, m.Envelope.Id)
				ws.Write(m.Json())
			}
			chatservice.CleanMsg(ids)
		}
		fid := chatservice.AppendHandleMsg(func(service *chat.ChatService, msg *chat.Message) {
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
	startService()
	if err := http.ListenAndServe(fmt.Sprintf("127.0.0.1:%d", rpcport), nil); err != nil {
		log.Println("listen_error", "err", err)
		os.Exit(1)
	}
}
