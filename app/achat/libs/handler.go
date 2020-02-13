package libs

import (
	"fmt"
	chat "github.com/cc14514/go-alibp2p-chat"
	"time"
)

func CmdMsgHandle(service *chat.ChatService, msg *chat.Message) {
	ts := time.Unix(msg.Envelope.Ct, 0).Format("2006-01-02 15:04:05")
	fmt.Println(msg.Envelope.From, ts, ">", msg.Payload.Content)
}

func AppMsgHandle(service *chat.ChatService, msg *chat.Message) {
	//fmt.Println("app >", string(msg.Json()))
}
