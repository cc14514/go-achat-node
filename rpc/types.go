package rpc

import (
	"encoding/json"
	"fmt"
	chat "github.com/cc14514/go-alibp2p-chat"
	"github.com/google/uuid"
	"github.com/tendermint/go-amino"
	"io"
	"strings"
)

type (
	RpcFn func(*Req) *Rsp

	API struct {
		Namespace string
		Api       map[string]RpcFn
	}

	Service interface {
		APIs() *API
	}

	Req struct {
		Id     string        `json:"id,omitempty"`
		Token  string        `json:"token"`
		Method string        `json:"method"`
		Params []interface{} `json:"params,omitempty"`
	}
	/*
		{
			"result": "Hello JSON-RPC",
		    "error": null,
			"id": 1
		}
	*/
	RspError struct {
		Code    string `json:"code,omitempty"`
		Message string `json:"message,omitempty"`
	}
	Rsp struct {
		Result interface{} `json:"result,omitempty"`
		Error  *RspError   `json:"error,omitempty"`
		Id     string      `json:"id,omitempty"`
	}

	Peerinfo struct {
		ID    string
		Addrs []string
	}

	User struct {
		Id      chat.JID `json:"id,omitempty"`
		Name    string   `json:"name,omitempty"`
		Age     int      `json:"age,omitempty"`
		Gender  int      `json:"gender,omitempty"`
		Icon    []byte   `json:"icon,omitempty"`
		Comment string   `json:"comment,omitempty"`
	}
)

func (c *User) FromJson(data []byte) (*User, error) {
	err := amino.UnmarshalJSON(data, c)
	return c, err
}

func (c *User) FromMap(m map[string]interface{}) (*User, error) {
	d, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return c.FromJson(d)
}

func (c *User) FromBytes(data []byte) (*User, error) {
	return c, amino.UnmarshalBinaryLengthPrefixed(data, c)
}

func (c *User) FromReader(r io.Reader) (*User, error) {
	_, err := amino.UnmarshalBinaryLengthPrefixedReader(r, c, 1024*1024)
	return c, err
}

func (c *User) Bytes() []byte {
	data, _ := amino.MarshalBinaryLengthPrefixed(c)
	return data
}

func NewRsp(id string, result interface{}, err *RspError) *Rsp {
	rsp := &Rsp{Id: id}
	if err != nil {
		rsp.Error = err
	} else {
		rsp.Result = result
	}
	return rsp
}

func (r *Rsp) WriteTo(rw io.Writer) error {
	_, err := rw.Write(r.Bytes())
	return err
}

func (r *Req) WriteTo(rw io.Writer) error {
	fmt.Println("request =>", string(r.Bytes()))
	_, err := rw.Write(r.Bytes())
	return err
}

func (r *Rsp) String() string {
	return string(r.Bytes())
}

func (r *Rsp) FromBytes(data []byte) (*Rsp, error) {
	err := json.Unmarshal(data, r)
	return r, err
}

func (r *Rsp) Bytes() []byte {
	buf, _ := json.Marshal(r)
	return buf
}

func NewReq(token, m string, p []interface{}) *Req {
	adapt := func(a []interface{}) *Req {
		as := make([]string, 0)
		for _, _a := range a {
			__a, ok := _a.(string)
			if !ok {
				return nil
			}
			as = append(as, __a)
		}
		j := strings.Join(as, " ")
		d := make(map[string]interface{})
		if err := json.Unmarshal([]byte(j), &d); err == nil {
			return &Req{uuid.New().String(), token, m, []interface{}{d}}
		}
		return nil
	}
	if req := adapt(p); req != nil {
		return req
	}
	return &Req{uuid.New().String(), token, m, p}
}

func (r *Req) String() string {
	return string(r.Bytes())
}

func (r *Req) FromBytes(data []byte) *Req {
	json.Unmarshal(data, r)
	return r
}

func (r *Req) Bytes() []byte {
	buf, _ := json.Marshal(r)
	return buf
}
