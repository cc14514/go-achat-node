package main

import (
	"achat/libs"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/peterh/liner"
	"github.com/urfave/cli"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func AttachCmd(ctx *cli.Context) error {
	var (
		token  string
		rpcurl = fmt.Sprintf("ws://localhost:%d/chat", rpcport)
	)

	auth := func() error {
		ws, err := websocket.Dial(rpcurl, "", "*")
		if err != nil {
			return err
		}
		defer ws.Close()
		req := libs.NewReq(token, "auth", []string{pwd})
		ws.Write(req.Bytes())
		tk, err := ioutil.ReadAll(ws)
		if err != nil {
			return err
		}
		rsp, err := new(libs.Rsp).FromBytes(tk)
		if err != nil {
			fmt.Println("aaaaaaaaaaa", err)
			return err
		}
		if rsp.Error != nil {
			return errors.New(rsp.Error.Code + " : " + rsp.Error.Message)
		}
		fmt.Println(string(tk), rsp)
		if tkarr := strings.Split(rsp.Result.(string), " "); len(tkarr) == 2 && tkarr[0] == "token" {
			token = tkarr[1]
			return nil
		}
		fmt.Println(string(tk))
		return errors.New("error pwd req / token resp")
	}
	if err := auth(); err != nil {
		return err
	}
	fmt.Println(len(os.Args), os.Args, token)
	if len(os.Args) == 3 {
		rp, err := strconv.Atoi(os.Args[2])
		if err == nil {
			rpcport = rp
		}
	}
	<-time.After(time.Second)
	//wsC, err := websocket.Dial(fmt.Sprintf("ws://localhost:%d/cancel", rpcport), "", "*")
	func() {
		defer func() {
			close(Stop)
			if chatservice != nil {
				chatservice.Stop()
			}
		}()
		fmt.Println("----------------------------------")
		fmt.Println("hello dshell, rpcport", rpcport)
		fmt.Println("----------------------------------")
		var (
			targetId      = ""
			ws            *websocket.Conn
			inCh          = make(chan string)
			history_fn    = filepath.Join(homedir, ".history")
			line          = liner.NewLiner()
			instructNames = libs.Instructs
		)
		defer func() {
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
			label := "chat$> "
			if targetId != "" {
				label = fmt.Sprintf("chat@%s$> ", targetId[len(targetId)-6:])
			}
			if cmd, err := line.Prompt(label); err == nil {
				if cmd == "" {
					continue
				}
				line.AppendHistory(cmd)
				if ws != nil {
					ws.Close()
				}
				ws, err = websocket.Dial(rpcurl, "", "*")
				if err != nil {
					fmt.Println("error", err)
					return
				}
				cmd = strings.Trim(cmd, " ")
				//app = app[:len([]byte(app))-1]
				// TODO 用正则表达式拆分指令和参数
				cmdArg := strings.Split(cmd, " ")
				//wsC.Write([]byte(cmdArg[0]))
				switch cmdArg[0] {
				case "opensession":
					//TODO verify id must be online
					targetId = cmdArg[1]
				case "exit":
					if targetId != "" {
						targetId = ""
					} else {
						fmt.Println("bye bye ^_^ ")
						return
					}
				default:
					down := make(chan struct{})
					go func() {
						defer func() {
							fmt.Println()
							close(down)
						}()
						req := libs.NewReq(token, cmdArg[0], cmdArg[1:])
						if targetId != "" {
							req = libs.NewReq(token, "sendmsg", append([]string{targetId}, cmdArg[:]...))
						}
						//fmt.Println(req)
						_, err = ws.Write(req.Bytes())
						if err != nil {
							fmt.Println("error", err)
							return
						}
						// TODO read message
						data, err := ioutil.ReadAll(ws)
						if err != nil {
							return
						}
						rsp, err := new(libs.Rsp).FromBytes(data)
						fmt.Println(">>>>>>>>>>>>", err, rsp)
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
