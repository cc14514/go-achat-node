// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 liangchuan

package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	chat "github.com/cc14514/go-achat-node"
	"github.com/cc14514/go-achat-node/rpc"
	"github.com/cc14514/go-alibp2p"
	ma "github.com/multiformats/go-multiaddr"
	"github.com/urfave/cli"
	"log"
	"math/big"
	"os"
	"strings"
	"sync"
	"time"
)

func init() {
	// Consistent, timestamped logs help when correlating with libp2p/zap output.
	log.SetFlags(log.LstdFlags | log.Lmicroseconds | log.LUTC)
}

func pskFingerprint(networkID *big.Int) string {
	if networkID == nil {
		return ""
	}
	sum := sha256.Sum256(networkID.Bytes())
	// Print a short, stable identifier; avoid dumping full material.
	return hex.EncodeToString(sum[:])[:12]
}

func logMultiaddrs(label string, addrs []ma.Multiaddr) {
	if len(addrs) == 0 {
		log.Printf("achat: %s.count=0", label)
		return
	}
	log.Printf("achat: %s.count=%d", label, len(addrs))
	for i, a := range addrs {
		if a == nil {
			continue
		}
		log.Printf("achat: %s[%d]=%s", label, i, a.String())
	}
}

func logStrings(label string, ss []string) {
	if len(ss) == 0 {
		log.Printf("achat: %s.count=0", label)
		return
	}
	log.Printf("achat: %s.count=%d", label, len(ss))
	for i, s := range ss {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		log.Printf("achat: %s[%d]=%s", label, i, s)
	}
}

var DEFBOOTNODES = []string{
	"/ip4/82.157.104.202/tcp/24000/p2p/16Uiu2HAmThtRghjg2k2fK1Zau5GW1svxrc2RXuNk5wwGnwA9juUT",
	"/ip4/39.105.35.133/tcp/24000/p2p/16Uiu2HAmUudavpi7V1FUxWoLhXTGJ4i38ZMhAuL2Tc9LPCz8EzZa",
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
	log.Printf(
		"achat: cfg homedir=%s port=%d rpcport=%d networkid=%s psk_fp=%s discover=%t relay=%t nodiscover=%t",
		homedir, port, rpcport, cfg.Networkid.String(), pskFingerprint(cfg.Networkid), cfg.Discover, cfg.Relay, nodiscover,
	)
	if bootnodes != "" {
		log.Printf("achat: flag.bootnodes=%s", bootnodes)
		cfg.Bootnodes = strings.Split(bootnodes, ",")
	}
	if nodiscover {
		log.Printf("achat: nodiscover=true; clearing bootnodes")
		cfg.Bootnodes = nil
	}
	logStrings("cfg.bootnodes", cfg.Bootnodes)
	if muxport > 0 {
		cfg.MuxPort = big.NewInt(int64(muxport))
	}
	a := alibp2p.NewService(cfg)
	b := a.(*alibp2p.Service)
	p2pservice = b
	p2pservice.Start()
	logMultiaddrs("host.listen_addrs", b.Host().Network().ListenAddresses())
	logMultiaddrs("host.addrs", b.Host().Addrs())
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
