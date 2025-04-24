package hubs

import (
	"fmt"
	"goapp/pkg/bytes"
	"goapp/pkg/hub"
)

type ChatMsgType byte

const (
	ChatMsgTypeReady ChatMsgType = 1
	ChatMsgTypePing  ChatMsgType = 5
	ChatMsgTypePong  ChatMsgType = 6
)

type ChatRespCode byte

const (
	ChatRespCodeOk ChatRespCode = 1
)

var chatProtocal = bytes.NewMsgPackProtocol(nil, nil)

func handleReceivedMsg(msg *hub.LineMessage) {
	meta, err := chatProtocal.GetMeta(msg.Data)
	if err != nil {
		// log err
		return
	}
	if meta.MsgType == byte(ChatMsgTypePing) {
		resp, err := chatProtocal.EncodeResp(int32(ChatMsgTypePong), meta.RequestId, byte(ChatRespCodeOk), nil)
		if err == nil {
			chatHub.PushToUserLines(msg.UserId, resp, msg.LineId)
		}
	}
}

func handleLineRegistered(r *hub.Line) {
	fmt.Printf("line registered: userid->%v, platform->%v", r.UserId(), r.Platform())
	resp, err := chatProtocal.EncodeResp(int32(ChatMsgTypeReady), 0, byte(ChatRespCodeOk), nil)
	if err == nil {
		chatHub.PushToUserLines(r.UserId(), resp, r.Id())
	}
}

func handleLineUnegistered(u *hub.Line) {
	fmt.Printf("line unregistered: userid->%v, platform->%v", u.UserId(), u.Platform())
}

func handleLineError(e *hub.LineError) {
	fmt.Printf("line error: userid->%v, platform->%v, err:%v", e.UserId, e.Platform, e.Error)
}
