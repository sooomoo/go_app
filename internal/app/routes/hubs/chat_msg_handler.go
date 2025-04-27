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
	fmt.Printf("[HUB] receive msg: userid->%v, platform->%v, line->%v, msg type->%v, requestId->%v, timestamp(relative)->%v\n", msg.UserId, msg.Platform, msg.LineId, meta.MsgType, meta.RequestId, meta.GetConvertedTimestamp())
	if meta.MsgType == byte(ChatMsgTypePing) {
		resp, err := chatProtocal.EncodeResp(int32(ChatMsgTypePong), meta.RequestId, byte(ChatRespCodeOk), nil)
		if err == nil {
			chatHub.PushToUserLines(msg.UserId, resp, msg.LineId)
		}
	}
}

func handleLineRegistered(r *hub.Line) {
	fmt.Printf("[HUB] line registered: userid->%v, platform->%v, line->%v\n", r.UserId(), r.Platform(), r.Id())
	resp, err := chatProtocal.EncodeResp(int32(ChatMsgTypeReady), 0, byte(ChatRespCodeOk), nil)
	if err == nil {
		chatHub.PushToUserLines(r.UserId(), resp, r.Id())
	}
}

func handleLineUnegistered(u *hub.Line) {
	fmt.Printf("[HUB] line unregistered: userid->%v, platform->%v, line->%v\n", u.UserId(), u.Platform(), u.Id())
}

func handleLineError(e *hub.LineError) {
	fmt.Printf("[HUB] line error: userid->%v, platform->%v, line->%v, err:%v\n", e.UserId, e.Platform, e.LineId, e.Error)
}
