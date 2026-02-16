// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 liangchuan

package rpc

import (
	"github.com/tendermint/go-amino"
	"strings"
	"testing"
)

func TestUser(t *testing.T) {
	u := &User{}
	u.Id = "hello"
	u.Name = "aaaaaaaa"
	buf, _ := amino.MarshalJSON(u)
	t.Log(string(buf))
}

func TestMap(t *testing.T) {
	s1 := "abc@123"
	s2 := "def@123"

	m := make(map[string]any)
	m[strings.Split(s1, "@")[1]] = "aaa"
	m[strings.Split(s2, "@")[1]] = "bbb"
	t.Log(m)
}
