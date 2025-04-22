package hubs

import (
	"fmt"

	"github.com/sooomo/niu"
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

var chatProtocal = niu.NewMsgPackProtocol(nil, nil)

func handleReceivedMsg(msg *niu.LineMessage) {
	meta, err := chatProtocal.GetMeta(msg.Data)
	if err != nil {
		// log err
		return
	}
	if meta.MsgType == byte(ChatMsgTypePing) {
		resp, err := chatProtocal.EncodeResp(int32(ChatMsgTypePong), meta.RequestId, byte(ChatRespCodeOk), nil)
		if err != nil {
			pushToUserLine(msg.UserId, msg.LineId, resp)
		}
	}
}

func pushToUserLine(userId, lineId string, data []byte) {
	uls := chatHub.GetUserLines(userId)
	if uls == nil {
		// log
		return
	}
	uls.PushMessageToLines(data, lineId)
}

func handleLineRegistered(r *niu.Line) {
	fmt.Printf("line registered: userid->%v, platform->%v", r.UserId(), r.Platform())
	resp, err := chatProtocal.EncodeResp(int32(ChatMsgTypeReady), 0, byte(ChatRespCodeOk), nil)
	if err != nil {
		pushToUserLine(r.UserId(), r.Id(), resp)
	}
}

func handleLineUnegistered(u *niu.Line) {
	fmt.Printf("line unregistered: userid->%v, platform->%v", u.UserId(), u.Platform())
}

func handleLineError(e *niu.LineError) {
	fmt.Printf("line error: userid->%v, platform->%v, err:%v", e.UserId, e.Platform, e.Error)
}
