// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 liangchuan

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/cc14514/go-achat-node"
	"github.com/cc14514/go-achat-node/rpc"
	"github.com/peterh/liner"
	"github.com/urfave/cli"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type urltype string

const (
	CHAT urltype = "chat"
	RPC  urltype = "rpc"
)

var (
	token  string
	rpcurl = func(p urltype) string {
		switch p {
		case CHAT:
			return fmt.Sprintf("ws://localhost:%d/%s", rpcport, p)
		case RPC:
			return fmt.Sprintf("http://localhost:%d/%s", rpcport, p)
		}
		return ""
	}
	ws        *websocket.Conn
	Stop      = make(chan struct{})
	reqCh     = make(chan *rpc.Req)
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
group_create 创建群
exit 退出 shell

`
	}
)

func rwLoop() {
	// read loop
	go func() {
		for {
			var j string
			err := websocket.Message.Receive(ws, &j)
			fmt.Println("RL -->", err, j)
			if err != nil {
				log.Println("readloop-error", err)
				return
			}
			msg, err := new(chat.Message).FromJson([]byte(j))
			fmt.Println("RL ==>", err, msg)

		}
	}()

	// write loop
	go func() {
		for {
			select {
			case <-Stop:
				return
			case req := <-reqCh:
				rsp, err := callrpc(req)
				fmt.Println("<--", err, rsp)
			}
		}
	}()
}

func callrpc(req *rpc.Req) (*rpc.Rsp, error) {
	buf := new(bytes.Buffer)
	req.WriteTo(buf)
	request, err := http.NewRequest("POST", rpcurl(RPC), buf)
	if err != nil {
		return nil, err
	}
	response, err := new(http.Client).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	rtn, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	rsp, err := new(rpc.Rsp).FromBytes(rtn)
	if err != nil {
		return nil, err
	}
	return rsp, nil
}

func auth() error {
	rsp, err := callrpc(rpc.NewReq(token, "auth", []interface{}{pwd}))
	if err != nil {
		return err
	}
	fmt.Println("---> rsp", rsp)
	if rsp.Error != nil {
		return errors.New(rsp.Error.Code + " : " + rsp.Error.Message)
	}
	if tkn := rsp.Result.(string); tkn != "" {
		token = tkn
		return nil
	}
	return errors.New("error pwd req / token resp")
}

func AttachCmd(_ *cli.Context) error {
	err := auth()
	if err != nil {
		log.Println("login fail", "err", err)
		//ws.Close()
		return err
	}
	ws, err = websocket.Dial(rpcurl(CHAT), "", "*")
	rwLoop()
	log.Println("login success", len(os.Args), os.Args, token)
	<-time.After(time.Second)
	rpc.NewReq(token, "open", nil).WriteTo(ws)
	func() {
		fmt.Println("----------------------------------")
		fmt.Println("hello chat example, rpcport", rpcport)
		fmt.Println("----------------------------------")
		var (
			targetId      = ""
			inCh          = make(chan string)
			history_fn    = filepath.Join(homedir, ".history")
			line          = liner.NewLiner()
			instructNames = Instructs
		)
		defer func() {
			close(Stop)
			if chatservice != nil {
				chatservice.Stop()
			}

			if f, err := os.Create(history_fn); err != nil {
				log.Print("Error writing history file: ", err)
			} else {
				line.WriteHistory(f)
				f.Close()
			}
			line.Close()
			close(inCh)
		}()

		line.SetCtrlCAborts(false)
		line.SetShouldRestart(func(err error) bool {
			return true
		})
		line.SetCompleter(func(line string) (c []string) {
			for _, n := range instructNames {
				if strings.HasPrefix(n, strings.ToLower(line)) {
					c = append(c, n)
				}
			}
			return
		})

		if f, err := os.Open(history_fn); err == nil {
			line.ReadHistory(f)
			f.Close()
		}

		for {
			label := "$> "
			if targetId != "" {
				label = fmt.Sprintf("@%s$> ", targetId[len(targetId)-6:])
			}
			if cmd, err := line.Prompt(label); err == nil {
				if cmd == "" {
					continue
				}
				line.AppendHistory(cmd)

				cmd = strings.Trim(cmd, " ")
				//app = app[:len([]byte(app))-1]
				// TODO 用正则表达式拆分指令和参数
				cmdArg := strings.Split(cmd, " ")
				//wsC.Write([]byte(cmdArg[0]))
				switch cmdArg[0] {
				case "opensession":
					//TODO verify id must be online
					targetId = cmdArg[1]
				case "help":
					fmt.Println(help())
				case "exit":
					if targetId != "" {
						targetId = ""
					} else {
						fmt.Println("bye bye ^_^ ")
						return
					}
				case "myid", "conns", "group_create", "user_query":
					var params []interface{}
					if len(cmdArg) > 1 {
						for _, p := range cmdArg[1:] {
							params = append(params, p)
						}
					}
					rsp, err := callrpc(rpc.NewReq(token, cmdArg[0], params))
					if err != nil {
						fmt.Println("error:", err)
					} else if rsp.Error != nil {
						fmt.Println("error:", rsp.Error.Code, rsp.Error.Message)
					} else {
						_, ok := rsp.Result.(string)
						if ok {
							fmt.Println(rsp.Result)
						} else {
							j, _ := json.Marshal(rsp.Result)
							d, _ := jshow(j)
							fmt.Println(string(d))
						}

					}
				default:
					down := make(chan struct{})
					go func() {
						defer func() {
							fmt.Println()
							close(down)
						}()
						req := rpc.NewReq(token, cmdArg[0], rpc.Str2X(cmdArg[1:]))
						if targetId != "" {
							req = rpc.NewReq(token, "sendmsg", append([]interface{}{targetId}, rpc.Str2X(cmdArg[:])...))
						}
						//fmt.Println(req)
						// TODO send req
						select {
						case <-Stop:
						case reqCh <- req:
						}
					}()
				}
			} else if err == liner.ErrPromptAborted {
				log.Print("Aborted")
				return
			} else {
				log.Print("Error reading line: ", err)
				return
			}

		}
	}()
	<-Stop
	return nil
}

func jshow(j []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, j, "", "\t")
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}
