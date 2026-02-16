// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 liangchuan

package chat

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/cc14514/go-achat-node/ldb"
	"github.com/cc14514/go-alibp2p"
	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/tendermint/go-amino"
	"io"
	"log"
)

// group struct =====================
type (
	Group struct {
		Id      GID          `json:"id,omitempty"`
		Owner   *GroupMember `json:"owner,omitempty"`
		Name    string       `json:"name,omitempty"`
		Comment string       `json:"comment,omitempty"`
		Lastlog string       `json:"lastlog,omitempty"`
	}
	GroupRsp struct {
		Group *Group
		Err   string
	}

	GroupMemberReq struct {
		Gid     GID
		Id      JID
		Action  MemberAction
		Members []*GroupMember
	}

	GroupMemberRsp struct {
		Action MemberAction
		Result []byte
		Err    string
	}

	GroupMessage struct {
		Msg        *Message
		ParentHash []byte
	}

	GroupMember struct {
		Id     JID    `json:"id,omitempty"`
		Name   string `json:"name,omitempty"`
		Gid    GID
		action MemberAction
	}

	// member 存储的时候，用链式存储，方便查找
	MemberItem struct {
		Id, Prve, Next JID
		Member         *GroupMember
	}

	MemberLog struct {
		Id, Prve, Next string
		Action         MemberAction
		Gid            GID
		MemberId       JID
	}
	groupdb struct {
		db, groupTab, memberTab ldb.Database
	}
	MemberAction string
)

const (
	ADD  MemberAction = "+"
	SUB  MemberAction = "-"
	FROM MemberAction = "f"
	TO   MemberAction = "t"
)

func (g *GroupRsp) FromBytes(dat []byte) (*GroupRsp, error) {
	err := amino.UnmarshalBinaryLengthPrefixed(dat, g)
	return g, err
}

func (g *GroupRsp) Json() []byte {
	j, _ := amino.MarshalJSON(g)
	return j
}

func (g *Group) FromReader(rw io.ReadWriter) (*Group, error) {
	_, err := amino.UnmarshalBinaryLengthPrefixedReader(rw, g, 10*MAX_PKG)
	return g, err
}

//func (g *Group) ToBytes() []byte {
//	ret, _ := amino.MarshalBinaryLengthPrefixed(g)
//	return ret
//}
//

const (
	group_prefix  = "GROUP"
	member_prefix = "GROUP_MEMBER"
)

var (
	memberK = func(gid GID, id JID) []byte { return []byte(fmt.Sprintf("%s_%s_member", gid, id)) }
	// 为什么只记录 last 呢？因为 owner 就是 first
	memberLastK    = func(gid GID) []byte { return []byte(fmt.Sprintf("%s_member_last", gid)) }
	memberLogK     = func(gid GID, id string) []byte { return []byte(fmt.Sprintf("%s_%s_memberlog", gid, id)) }
	memberLastlogK = func(gid GID) []byte { return []byte(fmt.Sprintf("%s_memberlog_last", gid)) }
)

func newGroupDB(db ldb.Database) *groupdb {
	gdb := &groupdb{db: db}
	gdb.groupTab = ldb.NewTable(db, group_prefix)
	gdb.memberTab = ldb.NewTable(db, member_prefix)
	return gdb
}

func (g *groupdb) _setLastlog(group *Group) *Group {
	if last, err := g.memberTab.Get(memberLastlogK(group.Id)); err == nil {
		var lastLog = new(MemberLog)
		if err = amino.UnmarshalBinaryLengthPrefixed(last, lastLog); err == nil {
			group.Lastlog = lastLog.Id
		}
	}
	return group
}

func (g *groupdb) saveGroup(group *Group) error {
	log.Println("saveGroup-start", "gid", group.Id, "gname", group.Name, "owner", group.Owner.Id)
	dat, _ := toByte(group)
	err := g.groupTab.Put([]byte(group.Id), dat)
	if err != nil {
		log.Println("saveGroup-error", "gid", group.Id, "err", err)
		return err
	}
	// 初始化
	if buf, err := g.memberTab.Get(memberLastK(group.Id)); err != nil && buf == nil {
		log.Println("saveGroup-init-member-start", "gid", group.Id)
		g.handleMember(&GroupMember{
			Id:     group.Owner.Id,
			Gid:    group.Id,
			Name:   group.Owner.Name,
			action: ADD,
		})
		log.Println("saveGroup-init-member-end", "gid", group.Id)
	}
	log.Println("saveGroup-end", "gid", group.Id, "gname", group.Name, "owner", group.Owner.Id)
	return nil
}

func (g *groupdb) getGroup(id GID) (*Group, error) {
	d, err := g.groupTab.Get([]byte(id))
	if err != nil {
		return nil, err
	}
	group := new(Group)
	err = amino.UnmarshalBinaryLengthPrefixed(d, group)
	if err != nil {
		return nil, err
	}
	return g._setLastlog(group), nil
}

func (g *groupdb) queryMember(gid GID, action MemberAction, id JID) []*GroupMember {
	var gml = make([]*GroupMember, 0)
	fn := func(id JID) (*GroupMember, JID, error) {
		m, err := g.getMember(gid, id)
		if err != nil {
			return nil, id, err
		}
		if action == FROM {
			return m.Member, m.Next, nil
		} else if action == TO {
			return m.Member, m.Prve, nil
		}
		return nil, id, errors.New("error action")
	}
	for {
		m, nid, err := fn(id)
		if err != nil {
			break
		}
		gml, id = append(gml, m), nid
	}
	return gml
}

func (g *groupdb) getMember(gid GID, id JID) (*MemberItem, error) {
	buf, err := g.memberTab.Get(memberK(gid, id))
	if err != nil {
		return nil, err
	}
	m := new(MemberItem)
	err = amino.UnmarshalBinaryLengthPrefixed(buf, m)
	return m, err
}

// save or delete , append memberlog
func (g *groupdb) handleMember(gm *GroupMember) error {
	log.Println("handleMember-start", "gid", gm.Gid, "mid", gm.Id, "action", gm.action)
	var (
		gid       = gm.Gid
		tab       = g.memberTab
		memberLog = &MemberLog{Id: uuid.New().String(), Action: gm.action}
		lastMlog  []byte
	)
	memberLog.MemberId = gm.Id

	lastlogBytes, _ := tab.Get(memberLastlogK(gid))
	var lastLog = new(MemberLog)
	// 更新 lastlog link to new memberLog >>>>
	if lastlogBytes != nil {
		amino.UnmarshalBinaryLengthPrefixed(lastlogBytes, lastLog)
		lastLog.Next = memberLog.Id
		memberLog.Prve = lastLog.Id
		tab.Put(memberLogK(gid, lastLog.Id), mustToByte(lastLog))
		log.Println("handleMember-lastlog-link",
			"gid", gm.Gid, "mid", gm.Id,
			"lastLog.Next", memberLog.Id,
			"memberLog.Prve", lastLog.Id,
			"action", memberLog.Action)
	}

	lastMlog = mustToByte(memberLog)
	// 更新 lastlog <<<<
	tab.Put(memberLogK(gid, memberLog.Id), lastMlog)
	log.Println("handleMember-memberLog-put", "logid", memberLog.Id)
	switch gm.action {
	case ADD:
		log.Println("handleMember-add-start", "mid", gm.Id)
		itm := &MemberItem{Id: gm.Id, Member: gm}
		// 构建链
		lastBytes, _ := tab.Get(memberLastK(gid))
		if lastBytes != nil {
			last := new(MemberItem)
			amino.UnmarshalBinaryLengthPrefixed(lastBytes, last)
			last.Next = itm.Id
			itm.Prve = last.Id
			tab.Put(memberK(gid, last.Id), mustToByte(last))
			log.Println("handleMember-add-last-link", "mid", gm.Id, "lastid", last.Id)
		}
		tab.Put(memberK(gid, gm.Id), mustToByte(itm))
		tab.Put(memberLastK(gid), mustToByte(itm))
		log.Println("handleMember-add-end", "mid", gm.Id)
	case SUB:
		//TODO fix link
		//tab.Delete(memberK(gid, gm.Id))
		itm, err := g.getMember(gm.Gid, gm.Id)
		if err != nil {
			log.Println("handleMember-del-error-1", "mid", gm.Id, "err", err)
			return err
		}
		log.Println("handleMember-del-start", "mid", gm.Id, "next", itm.Next, "prve", itm.Prve)
		itmPrve, err := g.getMember(gm.Gid, itm.Prve)
		if err != nil {
			// 上一个必须要有
			log.Println("handleMember-del-error-2", "mid", gm.Id, "err", err)
			return err
		}
		itmNext, err := g.getMember(gm.Gid, itm.Next)
		if err == nil {
			itmPrve.Next = itmNext.Id
			itmNext.Prve = itmPrve.Id
			tab.Put(memberK(gid, itmNext.Id), mustToByte(itmNext))
			log.Println("handleMember-del-fix-link-1", "mid", gm.Id, "prve.next", itmNext.Id, "next.prve", itmPrve.Id)
		} else {
			itmPrve.Next = ""
			// 如果删除的是最后一个，则需要把 last 更新了
			tab.Put(memberLastK(gid), mustToByte(itmPrve))
			log.Println("handleMember-del-fix-link-2", "mid", gm.Id, "prve.next", "nil")
		}
		tab.Put(memberK(gid, itmPrve.Id), mustToByte(itmPrve))
		tab.Delete(memberK(gid, itm.Id))
		log.Println("handleMember-del-end", "mid", gm.Id)
	default:
		log.Println("handleMember-error", "gid", gm.Gid, "mid", gm.Id, "action", gm.action, "err", "not support opt")
		return errors.New("not support opt")
	}
	log.Println("handleMember-end", "gid", gm.Gid, "mid", gm.Id, "action", gm.action)
	return tab.Put(memberLastlogK(gid), lastMlog)
}

func toByte(o interface{}) ([]byte, error) {
	return amino.MarshalBinaryLengthPrefixed(o)
}

func mustToByte(o interface{}) []byte {
	d, err := amino.MarshalBinaryLengthPrefixed(o)
	if err != nil {
		panic(err)
	}
	return d
}

func hash(data []byte) []byte {
	s1 := sha1.New()
	s1.Write(data)
	return s1.Sum(nil)
}

func resp(rw io.ReadWriter, o interface{}) error {
	_, err := amino.MarshalBinaryLengthPrefixedWriter(rw, o)
	return err
}

func (m *mailbox) newGID() GID {
	_, pub, _ := crypto.GenerateSecp256k1Key(rand.Reader)
	k0 := pub.(*crypto.Secp256k1PublicKey)
	pubkey := (*ecdsa.PublicKey)(k0)
	id, _ := alibp2p.ECDSAPubEncode(pubkey)
	return GID(id + m.myid.Mailid())
}

// 创建一个群
func (m *mailbox) genGroup(g *Group) (*GroupRsp, error) {
	to := m.myid.Mailid()
	if to == "" {
		return nil, errors.New("mailbox not found")
	}
	if g.Id == "" {
		g.Id = m.newGID()
	}
	pkg, err := toByte(g)
	if err != nil {
		return nil, err
	}
	rsp, err := m.p2pservice.RequestWithTimeout(to, PID_MAILBOX_GROUP_UPDATE, pkg, timeout)
	if err != nil {
		return nil, err
	}
	grsp, err := new(GroupRsp).FromBytes(rsp)
	if err != nil {
		return nil, err
	}
	return grsp, nil
}

func (m *mailbox) groupService() {
	var gdb = newGroupDB(m.db)
	// 添加/减少 成员
	m.p2pservice.SetHandler(PID_MAILBOX_GROUP_MEMBER, func(sessionId string, pubkey *ecdsa.PublicKey, rw io.ReadWriter) error {
		var (
			req = new(GroupMemberReq)
			rsp = new(GroupMemberRsp)
		)
		defer resp(rw, rsp)
		_, err := amino.UnmarshalBinaryLengthPrefixedReader(rw, req, 4096)
		if err != nil {
			rsp.Err = err.Error()
			return err
		}
		rsp.Action = req.Action
		if _, err := gdb.getGroup(req.Gid); err != nil {
			rsp.Err = "group not found"
			return err
		}
		switch rsp.Action {
		case ADD, SUB:
			for _, r := range req.Members {
				gdb.handleMember(r)
			}
			rsp.Result = SUCCESS
		case FROM, TO: // query
			req.Members = gdb.queryMember(req.Gid, req.Action, req.Id)
			result, err := amino.MarshalBinaryLengthPrefixed(req)
			if err != nil {
				rsp.Err = err.Error()
				return err
			}
			rsp.Result = result
		default:
			rsp.Err = "action not support"
		}
		return nil
	})
	// 创建和修改
	m.p2pservice.SetHandler(PID_MAILBOX_GROUP_UPDATE, func(sessionId string, pubkey *ecdsa.PublicKey, rw io.ReadWriter) error {
		log.Println("PID_MAILBOX_GROUP_UPDATE-start", "session", sessionId)
		var req, err = new(Group).FromReader(rw)
		if err != nil {
			resp(rw, GroupRsp{Err: err.Error()})
			log.Println("PID_MAILBOX_GROUP_UPDATE-error-1", "session", sessionId, "err", err)
			return err
		}
		if req.Id == "" {
			err = errors.New("req.Id not be nil")
			resp(rw, GroupRsp{Err: err.Error()})
			log.Println("PID_MAILBOX_GROUP_UPDATE-error-2", "session", sessionId, "err", err)
			return err
		}

		myid, _ := alibp2p.ECDSAPubEncode(pubkey)
		req.Owner = &GroupMember{Id: JID(myid)}
		err = gdb.saveGroup(req)
		if err != nil {
			resp(rw, GroupRsp{Err: err.Error()})
			log.Println("PID_MAILBOX_GROUP_UPDATE-error-3", "session", sessionId, "err", err)
			return err
		}
		g, err := gdb.getGroup(req.Id)
		if err != nil {
			resp(rw, GroupRsp{Err: err.Error()})
			log.Println("PID_MAILBOX_GROUP_UPDATE-error-4", "session", sessionId, "err", err)
			return err
		}
		resp(rw, GroupRsp{Group: g})
		log.Println("PID_MAILBOX_GROUP_UPDATE-end", "session", sessionId, "err", err)
		return err
	})

}
