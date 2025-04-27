package bytes

import (
	"errors"
	"fmt"
	"goapp/pkg/cryptos"
	"time"
)

type PacketMetaData struct {
	MsgType   byte  // 1字节
	RequestId int32 // 4字节
	Timestamp int32 // 4字节，从2025-01-01 00:00:00开始的秒数
}

func (m *PacketMetaData) GetConvertedTimestamp() time.Time {
	return protocolStartTime.Add(time.Duration(m.Timestamp) * time.Second)
}

type RequestPacket struct {
	PacketMetaData
	Payload any
}

var protocolStartTime = time.Date(2025, 1, 1, 0, 0, 0, 0, time.Local)

const (
	metaLength         = 9
	responseMetaLength = 10
)

type PacketProtocol struct {
	signer    cryptos.Signer
	cryptor   cryptos.Cryptor
	marshaler PayloadMarshaler
}

func NewMsgPackProtocol(signer cryptos.Signer, cryptor cryptos.Cryptor) *PacketProtocol {
	return &PacketProtocol{
		signer:    signer,
		cryptor:   cryptor,
		marshaler: msgpackMarshaler,
	}
}

func NewJsonProtocol(signer cryptos.Signer, cryptor cryptos.Cryptor) *PacketProtocol {
	return &PacketProtocol{
		signer:    signer,
		cryptor:   cryptor,
		marshaler: jsonMarshaler,
	}
}

func (m *PacketProtocol) GetMeta(data []byte) (*PacketMetaData, error) {
	if len(data) < metaLength {
		return nil, errors.New("bad data format")
	}
	requestId := int64(data[1])<<24 | int64(data[2])<<16 | int64(data[3])<<8 | int64(data[4])
	ts := int64(data[5])<<24 | int64(data[6])<<16 | int64(data[7])<<8 | int64(data[8])
	fmt.Println("requestId:", requestId, "timestamp:", ts)
	return &PacketMetaData{data[0], int32(requestId), int32(ts)}, nil
}

func (m *PacketProtocol) EncodeResp(msgType, requestId int32, code byte, payload any) ([]byte, error) {
	var body []byte
	if payload != nil {
		var err error
		body, err = m.marshaler.Marshal(payload)
		if err != nil {
			return nil, err
		}
	}

	timestamp := int32(time.Since(protocolStartTime).Seconds())
	out := []byte{byte(msgType)}
	out = append(out, byte(requestId>>24&0x00FF), byte(requestId>>16&0x00FF), byte(requestId>>8&0x00FF), byte(requestId&0x00FF))
	out = append(out, byte(timestamp>>24&0x00FF), byte(timestamp>>16&0x00FF), byte(timestamp>>8&0x00FF), byte(timestamp&0x00FF))
	out = append(out, code)

	if len(body) > 0 && m.cryptor != nil {
		var err error
		body, err = m.cryptor.Encrypt(body)
		if err != nil {
			return nil, err
		}
		out = append(out, body...)
	}

	if m.signer != nil {
		signature, err := m.signer.Sign(out)
		if err != nil {
			return nil, err
		}

		out = append(out, signature...)
	}

	return out, nil
}

func (m *PacketProtocol) DecodeReq(data []byte) (*RequestPacket, error) {
	meta, err := m.GetMeta(data)
	if err != nil {
		return nil, err
	}
	body := data[metaLength:]

	if m.signer != nil {
		signStart := len(data) - m.signer.SignatureLen()
		if signStart >= len(data) || signStart < metaLength {
			return nil, errors.New("bad data format: no sign")
		}
		signature := data[signStart:]
		body = data[metaLength:signStart]
		dataToVerify := data[:signStart]
		if !m.signer.Verify(dataToVerify, signature) {
			return nil, errors.New("sign verify fail")
		}
	}

	if len(body) > 0 {
		if m.cryptor != nil {
			body, err = m.cryptor.Decrypt(body)
			if err != nil {
				return nil, err
			}
		}

		var payload any
		if err = m.marshaler.Unmarshal(body, &payload); err != nil {
			return nil, err
		}

		return &RequestPacket{*meta, payload}, nil
	}

	return &RequestPacket{*meta, nil}, nil
}
