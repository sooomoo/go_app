package hubs

import (
	"fmt"

	"github.com/sooomo/niu"
)

func handleReceivedMsg(msg *niu.HubMessage) {
	msgProto := niu.NewMsgPackProtocol(nil, nil)
	var mp map[string]any
	if msgType, err := msgProto.DecodeReq(msg.Data, &mp); err == nil {
		fmt.Printf("recv msg type:%v, val is:%v", msgType, mp)
	}
	// TODO
}

func handleLineRegistered(r *niu.Line) {
	fmt.Printf("line registered: userid->%v, platform->%v", r.UserId(), r.Platform())
}

func handleLineUnegistered(u *niu.Line) {
	fmt.Printf("line unregistered: userid->%v, platform->%v", u.UserId(), u.Platform())
}

func handleLineError(e *niu.LineError) {
	fmt.Printf("line error: userid->%v, platform->%v, err:%v", e.UserId, e.Platform, e.Error)
}
