// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 liangchuan

package chat

import (
	"github.com/google/uuid"
	"github.com/tendermint/go-amino"
	"testing"
)

var msgJson = `
{
	"envelope": {
		"id": "uuid",
		"from": "16Uiu2HAkzRux7XYhYfmTDY2C7xuBapitNp25DvKvpvVnCf9bRne716Uiu2HAkzRux7XYhYfmTDY2C7xuBapitNp25DvKvpvVnCf9bRne7",
		"to": "16Uiu2HAkzRux7XYhYfmTDY2C7xuBapitNp25DvKvpvVnCf9bRne716Uiu2HAkzRux7XYhYfmTDY2C7xuBapitNp25DvKvpvVnCf9bRne7",
		"type": "1",
		"ack": "1",
		"ct": "13位时间戳，由服务器来补充此值",
		"pwd": "只有当 type=0 时，即 opensession 时，才会使用此属性",
		"gid": "群ID，只在 type=2 时会用到此属性"
	},
	"vsn": "0.2",
	"payload": {
		"attrs": [{ "key": "k","val":"v" }],
		"content": "Hello world."
	}
}
`

func TestMsg(t *testing.T) {
	msg, err := new(Message).FromJson([]byte(msgJson))
	t.Log(err)
	t.Log(len(msg.Json()), string(msg.Json()))
	t.Log(len(msg.Bytes()))
}

func TestJID(t *testing.T) {
	jid := JID("16Uiu2HAkzRux7XYhYfmTDY2C7xuBapitNp25DvKvpvVnCf9bRne716Uiu2HAkzRux7XYhYfmTDY2C7xuBapitNp25DvKvpvVnCf9bRne7")
	t.Log(jid.Peerid())
	t.Log(jid.Mailid())
	buf, err := amino.MarshalBinaryLengthPrefixed(jid)
	t.Log(err, buf)
	var jid2 JID
	err = amino.UnmarshalBinaryLengthPrefixed(buf, &jid2)
	t.Log(err, jid2)
}

func TestUUID(t *testing.T) {
	for i := 0; i < 10; i++ {
		t.Log(i, uuid.New().String())
	}
}

func TestMessages(t *testing.T) {
	msgs := []*Message{NewSysMessage("a"), NewSysMessage("b")}
	buf, err := amino.MarshalBinaryLengthPrefixed(&MessageBag{Messages: msgs})
	t.Log(err, buf)
	msgs2 := new(MessageBag)
	err = amino.UnmarshalBinaryLengthPrefixed(buf, msgs2)
	t.Log(err, msgs2.Messages[0])

	d := new(MessageBag).Bytes()
	t.Log(d)
	err = amino.UnmarshalBinaryLengthPrefixed(d, msgs2)
	t.Log(err, msgs2)

}
