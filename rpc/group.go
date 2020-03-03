/*************************************************************************
 * Copyright (C) 2016-2019 PDX Technologies, Inc. All Rights Reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 * @Time   : 2020/2/24 4:29 下午
 * @Author : liangc
 *************************************************************************/

package rpc

import (
	"encoding/json"
	"fmt"
	chat "github.com/cc14514/go-alibp2p-chat"
	"github.com/cc14514/go-alibp2p-chat/ldb"
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
