package plato

import (
	"encoding/binary"
	"fmt"
)

//go:generate protoc --go_out=./ plato.proto

const (
	MsgTypeMessageUpLink = 1 // 消息上行
	MsgTypeMessageDownLink = 2 // 消息下行
	MsgTypeOpenSession = 3 // 打开会话
	MsgTypeJoinSession = 4 // 加入会话
	MsgTypeLeaveSession = 5 // 离开会话
	MsgTypeCreateConn = 6 // 创建连接


)

type FixHeaderProtocol struct {
	version      [1]byte
	msgType      [1]byte
	varHeaderLen [4]byte
	bodyLen      [4]byte
}




func (p *FixHeaderProtocol) Check() error {
	fmt.Println("version:", p.GetVersion())
	fmt.Println("msgType:", p.GetMsgType())
	fmt.Println("varHeaderLen:", p.GetVarHeaderLen())
	fmt.Println("bodyLen:", p.GetBodyLen())
	return nil
}

func Marshal(version int8, msgType int8, header []byte, body []byte) []byte {
	msg := make([]byte, 10+len(header)+len(body))
	copy(msg[:1], []byte{byte(version)}[:])
	copy(msg[1:2], []byte{byte(msgType)}[:])
	int32ToBytes(int32(len(header)), msg[2:6])
	int32ToBytes(int32(len(body)), msg[6:10])
	copy(msg[10:10+len(header)], header)
	copy(msg[10+len(header):10+len(header)+len(body)], body)
	return msg
}


func int32ToBytes(value int32, bytes []byte) {
	binary.BigEndian.PutUint32(bytes, uint32(value))
}

func (p *FixHeaderProtocol) Unmarshal(data []byte) error {
	copy(p.version[:], data[:1])
	copy(p.msgType[:], data[1:2])
	copy(p.varHeaderLen[:], data[2:6])
	copy(p.bodyLen[:], data[6:10])

	return p.Check()
}

func (p *FixHeaderProtocol) GetVarHeaderLen() int {
	return int(binary.BigEndian.Uint32(p.varHeaderLen[:]))
}

func (p *FixHeaderProtocol) GetBodyLen() int {
	return int(binary.BigEndian.Uint32(p.bodyLen[:]))
}

func (p *FixHeaderProtocol) GetVersion() int {
	return int(p.version[0])
}

func (p *FixHeaderProtocol) GetMsgType() int {
	return int(p.msgType[0])
}
