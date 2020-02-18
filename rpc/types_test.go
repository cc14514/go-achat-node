package rpc

import (
	"github.com/tendermint/go-amino"
	"testing"
)

func TestUser(t *testing.T) {
	u := &User{}
	u.Id = "hello"
	u.Name = "aaaaaaaa"
	buf, _ := amino.MarshalJSON(u)
	t.Log(string(buf))
}
