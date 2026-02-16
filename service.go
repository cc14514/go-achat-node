// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 liangchuan

package chat

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"github.com/cc14514/go-alibp2p"
	"github.com/google/uuid"
	"io"
	"log"
	"sync"
	"time"
)

const (
	PID_NORMAL        = "/chat/normal/0.0.1"
	PID_GROUP         = "/chat/group/0.0.1"
	PID_MAILBOX       = "/chat/mailbox/put/0.0.1"
	PID_MAILBOX_QUERY = "/chat/mailbox/query/0.0.1"
	PID_MAILBOX_CLEAN = "/chat/mailbox/clean/0.0.1"

	PID_MAILBOX_GROUP_UPDATE = "/chat/mailbox/group/update/0.0.1"
	PID_MAILBOX_GROUP_DROP   = "/chat/mailbox/group/drop/0.0.1"
	PID_MAILBOX_GROUP_MSG    = "/chat/mailbox/group/msg/0.0.1"
	PID_MAILBOX_GROUP_MEMBER = "/chat/mailbox/group/member/0.0.1"
)

var (
	timeout = 10 * time.Second
	SUCCESS = []byte("success")
)

type ChatService struct {
	myid        JID // self id
	homedir     string
	ctx         context.Context
	p2pservice  alibp2p.Libp2pService
	recvMsgCh   chan Msg
	stop        chan struct{}
	handleMsgFn map[string]MsgHandle
	lock        *sync.Mutex
	mbox        *mailbox
}

func NewChatService(ctx context.Context, myid JID, homedir string, p2pservice alibp2p.Libp2pService) *ChatService {
	return &ChatService{
		ctx:         ctx,
		myid:        myid,
		homedir:     homedir,
		p2pservice:  p2pservice,
		recvMsgCh:   make(chan Msg, 128),
		stop:        make(chan struct{}),
		handleMsgFn: make(map[string]MsgHandle),
		lock:        new(sync.Mutex),
		mbox:        newMailbox(ctx, homedir, myid, p2pservice),
	}
}

func (c *ChatService) GetHomedir() string {
	return c.homedir
}

func (c *ChatService) GetMyid() JID {
	return c.myid
}
func (c *ChatService) GetLibp2pService() alibp2p.Libp2pService {
	return c.p2pservice
}

func (c *ChatService) AppendHandleMsg(fn MsgHandle) string {
	c.lock.Lock()
	defer c.lock.Unlock()
	fid := uuid.New().String()
	c.handleMsgFn[fid] = fn
	return fid
}

func (c *ChatService) DropHandleMsg(fid string) {
	c.lock.Lock()
	defer c.lock.Unlock()
	delete(c.handleMsgFn, fid)
}

func (c *ChatService) Stop() error {
	close(c.stop)
	return nil
}

func (c *ChatService) Start() error {
	c.normalService()
	go func() {
		for {
			select {
			case <-c.ctx.Done():
				return
			case <-c.stop:
				return
			case msg := <-c.recvMsgCh:
				if c.handleMsgFn != nil {
					for _, fn := range c.handleMsgFn {
						go fn(c, msg.(*Message))
					}
				}
			}
		}
	}()
	return c.mbox.Start()
}

func (c *ChatService) QueryMsg() (*MessageBag, error) {
	return c.mbox.QueryMsg(c.myid)
}

func (c *ChatService) CleanMsg(ids []string) error {
	if len(ids) > 0 {
		return c.mbox.CleanMsg(c.myid, ids)
	}
	return nil
}

func (c *ChatService) SendMsg(msg *Message) error {
	//log.Println("sendMsg", "msg", string(msg.Json()))
	switch msg.Envelope.Type {
	case NormalMsg:
		if _, err := c.p2pservice.RequestWithTimeout(msg.Envelope.To.Peerid(), PID_NORMAL, msg.Bytes(), timeout); err != nil {
			if _, err := c.p2pservice.RequestWithTimeout(msg.Envelope.To.Mailid(), PID_MAILBOX, msg.Bytes(), timeout); err != nil {
				log.Println("sendMsg error", "err", err, "msg", string(msg.Json()))
				return err
			}
		}
	case GroupMsg:
		if _, err := c.p2pservice.RequestWithTimeout(msg.Envelope.Gid.Peerid(), PID_GROUP, msg.Bytes(), timeout); err != nil {
			if _, err := c.p2pservice.RequestWithTimeout(msg.Envelope.Gid.Mailid(), PID_MAILBOX, msg.Bytes(), timeout); err != nil {
				log.Println("sendMsg error", "err", err, "msg", string(msg.Json()))
				return err
			}
		}
	default:
		return errors.New("not support yet")
	}
	return nil
}

func (c *ChatService) normalService() {
	c.p2pservice.SetHandler(PID_NORMAL, func(sessionId string, pubkey *ecdsa.PublicKey, rw io.ReadWriter) error {
		msg, err := new(Message).FromReader(rw)
		if err != nil {
			rw.Write([]byte(err.Error()))
			return err
		}

		select {
		case c.recvMsgCh <- msg:
		case <-c.stop:
		case <-c.ctx.Done():
		}

		rw.Write(SUCCESS)
		return nil
	})
}

// group service ====================================================
func (c *ChatService) CreateGroup(g *Group) (*GroupRsp, error) {
	return c.mbox.genGroup(g)
}
