// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 liangchuan
//
// NOTE: This file previously contained a GPL-3.0-or-later header.
// If any content in this file is derived from third-party GPL code,
// relicensing may require upstream permission. Verify provenance if needed.

package rpc

import (
	"encoding/json"
	"fmt"
	chat "github.com/cc14514/go-achat-node"
	"github.com/cc14514/go-achat-node/ldb"
	"github.com/tendermint/go-amino"
	"path"
)

type groupBean chat.Group

func (c *groupBean) FromJson(data []byte) (*chat.Group, error) {
	err := amino.UnmarshalJSON(data, c)
	return (*chat.Group)(c), err
}

func (c *groupBean) FromMap(m map[string]interface{}) (*chat.Group, error) {
	d, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return c.FromJson(d)
}

type GroupService struct {
	chatservice *chat.ChatService
	db          ldb.Database
}

func NewGroupService(chatservice *chat.ChatService) Service {
	db, err := ldb.NewLDBDatabase(path.Join(chatservice.GetHomedir(), "group"), 0, 0)
	if err != nil {
		panic(err)
	}
	return &GroupService{chatservice: chatservice, db: db}
}

func (g GroupService) Create(req *Req) *Rsp {
	fmt.Println("group.create -->", req)
	if len(req.Params) < 1 {
		return NewRsp(req.Id, nil, &RspError{Code: "10000", Message: "group not nil"})
	}
	var (
		err   error
		group = new(chat.Group)
	)
	if j, ok := req.Params[0].(string); ok {
		group, err = new(groupBean).FromJson([]byte(j))
	} else if m, ok := req.Params[0].(map[string]interface{}); ok {
		group, err = new(groupBean).FromMap(m)
	}

	grsp, err := g.chatservice.CreateGroup(group)
	rsp := NewRsp(req.Id, grsp, nil)
	if err != nil {
		rsp = NewRsp(req.Id, nil, &RspError{
			Code:    "10001",
			Message: err.Error(),
		})
	}
	fmt.Println("group.create <--", err, rsp)

	if putFn := getRpcFn("user_put"); putFn != nil {
		putReq := &Req{Params: []interface{}{
			map[string]interface{}{
				"gid":     grsp.Group.Id,
				"name":    grsp.Group.Name,
				"comment": grsp.Group.Comment,
				"lastlog": grsp.Group.Lastlog,
			},
		}}
		putFn(putReq)
	}
	return rsp

}

func (g GroupService) APIs() *API {
	return &API{
		Namespace: "group",
		Api: map[string]RpcFn{
			"create": g.Create,
		},
	}
}
