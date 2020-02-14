package chat

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"errors"
	"github.com/cc14514/go-alibp2p"
	"github.com/cc14514/go-alibp2p-chat/ldb"
	"github.com/tendermint/go-amino"
	"io"
	"log"
	"path"
	"sort"
)

var kfilter = func(prefix, k []byte) bool {
	if k != nil && len(k) > len(prefix) {
		return bytes.Equal(k[:len(prefix)], prefix)
	}
	return false
}

type mailbox struct {
	ctx        context.Context
	stop       chan struct{}
	db         ldb.Database
	p2pservice alibp2p.Libp2pService
}

func newMailbox(ctx context.Context, homedir string, p2pservice alibp2p.Libp2pService) *mailbox {
	db, err := ldb.NewLDBDatabase(path.Join(homedir, "mailbox"), 0, 0)
	if err != nil {
		panic(err)
	}
	return &mailbox{
		ctx:        ctx,
		stop:       make(chan struct{}),
		db:         db,
		p2pservice: p2pservice,
	}
}

func (m *mailbox) verifyMsg(msg *Message) error {
	// TODO 验证绑定关系
	log.Println("TODO 验证绑定关系", msg)
	return nil
}

func (m *mailbox) putMsg(msg *Message) error {
	id := msg.Envelope.To.Peerid()
	tab := ldb.NewTable(m.db, id)
	return tab.Put([]byte(msg.Envelope.Id), msg.Bytes())
}

func (m *mailbox) doCleanMsg(cleanMsg *CleanMsg) {
	id := cleanMsg.Jid.Peerid()
	tab := ldb.NewTable(m.db, id)
	for _, mid := range cleanMsg.Ids {
		tab.Delete([]byte(mid))
	}
}

func (m *mailbox) doQueryMsg(jid JID) *MessageBag {
	id := jid.Peerid()
	tab := ldb.NewTable(m.db, id)
	it := tab.NewIterator()
	sl := make([]*Message, 0)
	for it.Next() {
		if kfilter([]byte(id), it.Key()) {
			if m, err := new(Message).FromBytes(it.Value()); err == nil {
				sl = append(sl, m.(*Message))
			}
		}
	}
	ml := MessageList(sl)
	sort.Sort(ml)
	return &MessageBag{Messages: ml}
}

func (m *mailbox) Stop() error {
	close(m.stop)
	return nil
}

func (m *mailbox) Start() error {
	m.queryService()
	m.msgService()
	m.cleanService()
	return nil
}

func (m *mailbox) CleanMsg(jid JID, ids []string) error {
	data, err := amino.MarshalBinaryLengthPrefixed(&CleanMsg{jid, ids})
	if err != nil {
		return err
	}
	_, err = m.p2pservice.RequestWithTimeout(jid.Mailid(), PID_MAILBOX_CLEAN, data, timeout)
	return err
}

func (m *mailbox) cleanService() {
	m.p2pservice.SetHandler(PID_MAILBOX_CLEAN, func(sessionId string, pubkey *ecdsa.PublicKey, rw io.ReadWriter) error {
		cleanMsg := new(CleanMsg)
		_, err := amino.UnmarshalBinaryLengthPrefixedReader(rw, cleanMsg, 2*1024*1024)
		if err != nil {
			log.Println("PID_MAILBOX_CLEAN error", "err", err)
			return err
		}
		m.doCleanMsg(cleanMsg)
		return err
	})
}

func (m *mailbox) QueryMsg(jid JID) (*MessageBag, error) {
	data, err := amino.MarshalBinaryLengthPrefixed(jid)
	if err != nil {
		return nil, err
	}
	rtn, err := m.p2pservice.RequestWithTimeout(jid.Mailid(), PID_MAILBOX_QUERY, data, timeout)
	if err != nil {
		return nil, err
	}
	var msgs = new(MessageBag)
	err = amino.UnmarshalBinaryLengthPrefixed(rtn, msgs)
	return msgs, err
}

func (m *mailbox) queryService() {
	m.p2pservice.SetHandler(PID_MAILBOX_QUERY, func(sessionId string, pubkey *ecdsa.PublicKey, rw io.ReadWriter) error {
		var k JID
		_, err := amino.UnmarshalBinaryLengthPrefixedReader(rw, &k, 128)
		if err != nil {
			log.Println("PID_MAILBOX_QUERY error", "err", err)
			rw.Write(new(MessageBag).Bytes())
			return err
		}
		_, err = rw.Write(m.doQueryMsg(k).Bytes())
		return err
	})
}

func (m *mailbox) msgService() {
	m.p2pservice.SetHandler(PID_MAILBOX, func(sessionId string, pubkey *ecdsa.PublicKey, rw io.ReadWriter) error {
		msg, err := new(Message).FromReader(rw)
		if err != nil {
			rw.Write([]byte(err.Error()))
			return err
		}
		message := msg.(*Message)
		if err := m.verifyMsg(message); err != nil {
			rw.Write([]byte(err.Error()))
			return err
		}
		switch message.Envelope.Type {
		case NormalMsg, GroupMsg:
			if err := m.putMsg(msg.(*Message)); err != nil {
				rw.Write([]byte(err.Error()))
				return err
			}
		default:
			return errors.New("not support yet")
		}
		rw.Write(SUCCESS)
		return nil
	})
}
