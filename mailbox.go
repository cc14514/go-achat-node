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

func (m *mailbox) verifyMsg(msg *Message) error {
	// TODO 验证绑定关系
	return nil
}

func (m *mailbox) putMsg(msg *Message) error {
	// TODO 验证绑定关系
	id := msg.Envelope.To.Peerid()
	tab := ldb.NewTable(m.db, id)
	return tab.Put([]byte(msg.Envelope.Id), msg.Bytes())
}

func (m *mailbox) queryMsg(jid JID) Messages {
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
	return sl
}

func (m *mailbox) Stop() error {
	close(m.stop)
	return nil
}

func (m *mailbox) queryService() {
	m.p2pservice.SetHandler(PID_MAILBOX_QUERY, func(sessionId string, pubkey *ecdsa.PublicKey, rw io.ReadWriter) error {
		var k string
		_, err := amino.UnmarshalBinaryLengthPrefixedReader(rw, &k, 128)
		if err != nil {
			amino.MarshalBinaryLengthPrefixedWriter(rw, err)
			return err
		}
		_, err = rw.Write(m.queryMsg(JID(k)).Bytes())
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
		//TODO
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
