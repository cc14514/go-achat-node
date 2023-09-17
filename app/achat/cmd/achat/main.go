package main

import (
	"context"
	"fmt"
	"github.com/cc14514/go-alibp2p"
	chat "github.com/cc14514/go-alibp2p-chat"
	"github.com/cc14514/go-alibp2p-chat/rpc"
	"github.com/urfave/cli"
	"log"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"
)

var DEFBOOTNODES = []string{
	"/ip4/82.157.104.202/tcp/24000/p2p/16Uiu2HAmThtRghjg2k2fK1Zau5GW1svxrc2RXuNk5wwGnwA9juUT",
}

var (
	tspool                                          = alibp2p.NewAsyncRunner(context.Background(), 100, 1024)
	tpscounter                                      = new(sync.Map)
	homedir, bootnodes, capwd, leader, pwd, mailbox string
	port, networkid, rpcport, muxport               int
	nodiscover                                      bool
	p2pservice                                      alibp2p.Libp2pService
	app                                             = cli.NewApp()
	chatservice                                     *chat.ChatService
)

func init() {
	app.Name = os.Args[0]
	app.Usage = "基于 go-alibp2p 的 chat"
	app.Version = "0.0.1"
	app.Author = "liangc"
	app.Email = "cc14514@icloud.com"
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:        "rpcport",
			Usage:       "RPC server listening `PORT`",
			Value:       9990,
			Destination: &rpcport,
		},
		cli.IntFlag{
			Name:        "port",
			Usage:       "service tcp port",
			Value:       24000,
			Destination: &port,
		},
		cli.IntFlag{
			Name:        "networkid",
			Usage:       "network id",
			Value:       1,
			Destination: &networkid,
		},
		cli.StringFlag{
			Name:        "homedir,d",
			Usage:       "home dir",
			Value:       "/tmp",
			Destination: &homedir,
		},
		cli.StringFlag{
			Name:        "pwd",
			Usage:       "passwd for subcmd attach",
			Destination: &pwd,
		},
		cli.StringFlag{
			Name:        "mailbox",
			Usage:       "recv offline message",
			Destination: &mailbox,
		},
		cli.StringFlag{
			Name:        "bootnodes",
			Usage:       "bootnode list split by ','",
			Destination: &bootnodes,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:   "attach",
			Usage:  "attach to console",
			Action: AttachCmd,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:        "pwd",
					Usage:       "ashell node's passwd for attach",
					Destination: &pwd,
				},
				cli.IntFlag{
					Name:        "rpcport,p",
					Usage:       "RPC server's `PORT`",
					Value:       9990,
					Destination: &rpcport,
				},
			},
		},
		{
			Name:  "console",
			Usage: "start with console",
			Action: func(ctx *cli.Context) error {
				fmt.Println("rpcport", rpcport)
				fmt.Println("homedir", homedir)
				go achat(ctx)
				<-time.After(2 * time.Second)
				return AttachCmd(ctx)
			},
		},
		{
			Name:  "bootnode",
			Usage: "start as a bootnode",
			Action: func(ctx *cli.Context) error {
				nodiscover = true
				return achat(ctx)
			},
		},
	}

	app.Before = func(ctx *cli.Context) error {
		return nil
	}
	app.Action = achat
}

func achat(_ *cli.Context) error {
	if homedir == "" {
		panic("homedir can not empty.")
	}
	_ctx := context.Background()
	cfg := alibp2p.Config{
		Ctx:       _ctx,
		Homedir:   homedir,
		Port:      uint64(port),
		Discover:  !nodiscover,
		Networkid: big.NewInt(int64(networkid)),
		Loglevel:  4, // 3 INFO, 4 DEBUG, 5 TRACE -> 3-4 INFO, 5 DEBUG
		Bootnodes: DEFBOOTNODES,
		Relay:     true,
	}
	if bootnodes != "" {
		log.Println("bootnodes=", bootnodes)
		cfg.Bootnodes = strings.Split(bootnodes, ",")
	}
	if nodiscover {
		cfg.Bootnodes = nil
	}
	if muxport > 0 {
		cfg.MuxPort = big.NewInt(int64(muxport))
	}
	a := alibp2p.NewService(cfg)
	b := a.(*alibp2p.Service)
	p2pservice = b
	p2pservice.Start()
	myid, _ := p2pservice.Myid()
	chatservice = chat.NewChatService(_ctx, chat.NewJID(myid, mailbox), homedir, p2pservice)
	chatservice.AppendHandleMsg(func(service *chat.ChatService, msg *chat.Message) {
		// log handler
		log.Println("-->", msg)
	})

	chatservice.Start()
	log.Println(">> Action on port =", port)
	log.Println(">> myid =", myid)
	rpc.StartRPC(pwd, rpcport, chatservice)
	return nil
}

func main() {
	if err := app.Run(os.Args); err != nil {
		os.Exit(-1)
	}
}
