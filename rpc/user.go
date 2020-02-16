package rpc

import (
	"fmt"
	chat "github.com/cc14514/go-alibp2p-chat"
	"github.com/cc14514/go-alibp2p-chat/ldb"
	"path"
)

type UserService struct {
	chatservice *chat.ChatService
	db          ldb.Database
}

func NewUserService(chatservice *chat.ChatService) Service {
	db, err := ldb.NewLDBDatabase(path.Join(chatservice.GetHomedir(), "user"), 0, 0)
	if err != nil {
		panic(err)
	}
	return &UserService{chatservice: chatservice, db: db}
}

func (u *UserService) put(user *User) error {
	return u.db.Put([]byte(user.Id.Peerid()), user.Bytes())
}

func (u *UserService) get(id chat.JID) (*User, error) {
	d, err := u.db.Get([]byte(id))
	if err != nil {
		return nil, err
	}
	return new(User).FromBytes(d)
}

func (u *UserService) del(id chat.JID) error {
	err := u.db.Delete([]byte(id))
	if err != nil {
		return err
	}
	return nil
}

func (u *UserService) query() ([]*User, error) {
	it := u.db.NewIterator()
	sl := make([]*User, 0)
	for it.Next() {
		if m, err := new(User).FromBytes(it.Value()); err == nil {
			sl = append(sl, m)
		}
	}
	return sl, nil
}

func (u *UserService) Put(req *Req) *Rsp {
	fmt.Println("user.put -->", req)
	if len(req.Params) < 1 {
		return NewRsp(req.Id, nil, &RspError{Code: "10000", Message: "user not nil"})
	}
	var (
		err  error
		user = new(User)
	)
	if j, ok := req.Params[0].(string); ok {
		user, err = user.FromJson([]byte(j))
	} else if m, ok := req.Params[0].(map[string]interface{}); ok {
		user, err = user.FromMap(m)
	}
	if err != nil {
		return NewRsp(req.Id, nil, &RspError{Code: "10001", Message: err.Error()})
	}
	if user.Id == "" {
		return NewRsp(req.Id, nil, &RspError{Code: "10002", Message: "userid not nil"})
	}
	if err = u.put(user); err != nil {
		return NewRsp(req.Id, nil, &RspError{Code: "10003", Message: err.Error()})
	}
	rsp := NewRsp(req.Id, "success", nil)
	fmt.Println("user.put <--", rsp)
	return rsp
}

func (u *UserService) Get(req *Req) *Rsp {
	fmt.Println("user.get -->", req)
	if len(req.Params) < 1 {
		return NewRsp(req.Id, nil, &RspError{Code: "20001", Message: "userid not nil"})
	}
	user, err := u.get(chat.JID(req.Params[0].(string)))
	if err != nil {
		return NewRsp(req.Id, nil, &RspError{Code: "20002", Message: err.Error()})
	}
	rsp := NewRsp(req.Id, user, nil)
	fmt.Println("user.get <--", rsp)
	return rsp
}

func (u *UserService) Del(req *Req) *Rsp {
	fmt.Println("user.del -->", req)
	if len(req.Params) < 1 {
		return NewRsp(req.Id, nil, &RspError{Code: "30001", Message: "userid not nil"})
	}
	err := u.del(chat.JID(req.Params[0].(string)))
	if err != nil {
		return NewRsp(req.Id, nil, &RspError{Code: "30002", Message: err.Error()})
	}
	rsp := NewRsp(req.Id, "success", nil)
	fmt.Println("user.del <--", rsp)
	return rsp
}

func (u *UserService) Query(req *Req) *Rsp {
	fmt.Println("user.query -->", req)
	us, err := u.query()
	if err != nil {
		return NewRsp(req.Id, nil, &RspError{Code: "40001", Message: err.Error()})
	}
	rsp := NewRsp(req.Id, us, nil)
	fmt.Println("user.query <--", rsp)
	return rsp
}

func (u *UserService) APIs() *API {
	return &API{
		Namespace: "user",
		Api: map[string]RpcFn{
			"put":   u.Put,
			"get":   u.Get,
			"del":   u.Del,
			"query": u.Query,
		},
	}
}
