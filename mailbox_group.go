package chat

import (
	"crypto/ecdsa"
	"crypto/sha1"
	"errors"
	"fmt"
	"github.com/cc14514/go-alibp2p"
	"github.com/cc14514/go-alibp2p-chat/ldb"
	"github.com/google/uuid"
	"github.com/tendermint/go-amino"
	"io"
)

// group struct =====================
type (
	Group struct {
		Id      GID
		Owner   *GroupMember
		Name    string
		Comment string
		Lastlog string
	}
	GroupRsp struct {
		Group *Group
		Err   string
	}

	GroupMemberReq struct {
		Gid     GID
		Action  string // + / -
		Members []*GroupMember
	}

	GroupMemberRsp struct {
		Result string
		Err    string
	}

	GroupMessage struct {
		Msg        *Message
		ParentHash []byte
	}

	GroupMember struct {
		Id     JID
		Gid    GID
		Name   string
		action string
	}

	// member 存储的时候，用链式存储，方便查找
	MemberItem struct {
		Id, Prve, Next JID
		Member         *GroupMember
	}

	MemberLog struct {
		Id, Prve, Next, Action string
		Gid                    GID
		MemberId               JID
	}
	groupdb struct {
		db, groupTab, memberTab ldb.Database
	}
)

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
	dat, _ := toByte(group)
	err := g.groupTab.Put([]byte(group.Id), dat)
	if err != nil {
		return err
	}
	// 初始化
	if buf, err := g.memberTab.Get(memberLastK(group.Id)); err != nil && buf == nil {
		g.handleMember(&GroupMember{
			Id:     group.Owner.Id,
			Gid:    group.Id,
			Name:   group.Owner.Name,
			action: "+",
		})
	}
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
	var (
		gid       = gm.Gid
		tab       = g.memberTab
		memberLog = &MemberLog{Id: uuid.New().String(), Action: gm.action}
		lastMlog  []byte
	)
	memberLog.MemberId = gm.Id

	lastlogBytes, _ := tab.Get(memberLastlogK(gid))
	var lastLog = new(MemberLog)
	// 更新 lastlog >>>>
	if lastlogBytes != nil {
		amino.UnmarshalBinaryLengthPrefixed(lastlogBytes, lastLog)
		lastLog.Next = memberLog.Id
		memberLog.Prve = lastLog.Id
		tab.Put(memberLogK(gid, lastLog.Id), mustToByte(lastLog))
	}

	lastMlog = mustToByte(memberLog)
	// 更新 lastlog <<<<
	tab.Put(memberLogK(gid, memberLog.Id), lastMlog)
	switch gm.action {
	case "+":
		itm := &MemberItem{Id: gm.Id, Member: gm}
		// 构建链
		lastBytes, _ := tab.Get(memberLastK(gid))
		if lastBytes != nil {
			last := new(MemberItem)
			amino.UnmarshalBinaryLengthPrefixed(lastBytes, last)
			last.Next = itm.Id
			itm.Prve = last.Id
			tab.Put(memberK(gid, last.Id), mustToByte(last))
		}
		tab.Put(memberK(gid, gm.Id), mustToByte(itm))
		tab.Put(memberLastK(gid), mustToByte(itm))
	case "-":
		//TODO fix link
		//tab.Delete(memberK(gid, gm.Id))
		itm, err := g.getMember(gm.Gid, gm.Id)
		if err != nil {
			return err
		}
		itmPrve, err := g.getMember(gm.Gid, itm.Prve)
		if err != nil {
			// 上一个必须要有
			return err
		}
		itmNext, err := g.getMember(gm.Gid, itm.Next)
		if err == nil {
			itmPrve.Next = itmNext.Id
			itmNext.Prve = itmPrve.Id
			tab.Put(memberK(gid, itmNext.Id), mustToByte(itmNext))
		} else {
			itmPrve.Next = ""
			// 如果删除的是最后一个，则需要把 last 更新了
			tab.Put(memberLastK(gid), mustToByte(itmPrve))
		}
		tab.Put(memberK(gid, itmPrve.Id), mustToByte(itmPrve))
		tab.Delete(memberK(gid, itm.Id))
	default:
		return errors.New("not support opt")
	}
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

func (m *mailbox) groupService() {
	var gdb = newGroupDB(m.db)
	// 添加/减少 成员
	m.p2pservice.SetHandler(PID_MAILBOX_GROUP_MEMBER, func(sessionId string, pubkey *ecdsa.PublicKey, rw io.ReadWriter) error {
		var req = new(GroupMemberReq)
		_, err := amino.UnmarshalBinaryLengthPrefixedReader(rw, req, 4096)
		if err != nil {
			resp(rw, GroupMemberRsp{Err: err.Error()})
			return err
		}
		if _, err := gdb.getGroup(req.Gid); err != nil {
			resp(rw, GroupMemberRsp{Err: "group not found"})
			return err
		}
		for _, r := range req.Members {
			gdb.handleMember(r)
		}
		return nil
	})
	// 创建和修改
	m.p2pservice.SetHandler(PID_MAILBOX_GROUP_UPDATE, func(sessionId string, pubkey *ecdsa.PublicKey, rw io.ReadWriter) error {
		var req = new(Group)
		_, err := amino.UnmarshalBinaryLengthPrefixedReader(rw, req, 4096)
		if err != nil {
			resp(rw, GroupRsp{Err: err.Error()})
			return err
		}
		if req.Id == "" {
			req.Id = GID(uuid.New().String())
		}

		myid, _ := alibp2p.ECDSAPubEncode(pubkey)
		req.Owner.Id = JID(myid)
		err = gdb.saveGroup(req)
		if err != nil {
			resp(rw, GroupRsp{Err: err.Error()})
			return err
		}

		resp(rw, GroupRsp{Group: req})
		return err
	})

}
