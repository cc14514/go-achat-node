package chat

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"github.com/cc14514/go-alibp2p"
	"io"
	"log"
	"time"
)

const (
	PID_NORMAL        = "/chat/normal/0.0.1"
	PID_GROUP         = "/chat/group/0.0.1"
	PID_MAILBOX       = "/chat/mailbox/put/0.0.1"
	PID_MAILBOX_QUERY = "/chat/mailbox/query/0.0.1"
)

var (
	timeout = 10 * time.Second
	SUCCESS = []byte("success")
)

type ChatService struct {
	myid        JID // self id
	ctx         context.Context
	p2pservice  alibp2p.Libp2pService
	recvMsgCh   chan Msg
	stop        chan struct{}
	handleMsgFn []func(*ChatService, *Message)
}

func NewChatService(ctx context.Context, myid JID, p2pservice alibp2p.Libp2pService) *ChatService {
	return &ChatService{
		ctx:        ctx,
		myid:       myid,
		p2pservice: p2pservice,
		recvMsgCh:  make(chan Msg, 128),
		stop:       make(chan struct{}),
	}
}

func (c *ChatService) GetMyid() JID {
	return c.myid
}
func (c *ChatService) GetLibp2pService() alibp2p.Libp2pService {
	return c.p2pservice
}

func (c *ChatService) AppendHandleMsg(fn func(service *ChatService, msg *Message)) *ChatService {
	c.handleMsgFn = append(c.handleMsgFn, fn)
	return c
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