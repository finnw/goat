//  ---------------------------------------------------------------------------
//
//  msgMsg.go
//
//  This file is auto-generated by the net message code generator and should 
//  NOT be edited by hand unless you know what you are doing. Changes to the
//  source object definition will be automatically reflected in the this 
//  generated code the next time genproc is run.
//
//  -----------
package chat

// External imports.
import (
	"github.com/xaevman/goat/core/net"
	"github.com/xaevman/goat/lib/buffer"
)

// Stdlin imports.
import (
	"errors"
	"fmt"
)

// Generated imports.
import (
	"github.com/xaevman/goat/prod"
)

// MsgHandler is an empty function container.
type MsgHandler struct {}

// Close is called when a message signature is unregistered from a protocol.
func (this *MsgHandler) Close() {}

// Init is called when the message signature is first registered in a protocol.
func (this *MsgHandler) Init(proto *net.Protocol) {}

// DeserializeMsg is called by the protocol after an incoming network message has been 
// validated, decrypted, and uncompressed.
func (this *MsgHandler) DeserializeMsg(msg *net.Msg, access byte) (interface{}, error) {
	var err error

	cursor := 0
	data   := msg.GetPayload()
	nMsg   := new(Msg)
	
	nMsg.ChannelId, err = buffer.ReadUint32(data, &cursor)
	if err != nil { return nil, err }
	
	nMsg.From, err = buffer.ReadString(data, &cursor)
	if err != nil { return nil, err }
	
	nMsg.Subtype, err = buffer.ReadByte(data, &cursor)
	if err != nil { return nil, err }
	
	nMsg.Text, err = buffer.ReadString(data, &cursor)
	if err != nil { return nil, err }
	
	return nMsg, nil
}

// SerializeMsg is called by the protocol after a Msg object has been validated,
// compressed, and encrypted, in order to prepare a network message for transmission.
func (this *MsgHandler) SerializeMsg(data interface{}) (*net.Msg, error) {
	cursor      := 0
	nMsg, ok := data.(*Msg)
	if !ok {
		return nil, errors.New(fmt.Sprintf("Cannot serialize type %T", data))
	}

	dataLen := 0
	
	dataLen += buffer.LenUint32()
	dataLen += buffer.LenString(nMsg.From)
	dataLen += buffer.LenByte()
	dataLen += buffer.LenString(nMsg.Text)

	dataBuffer := make([]byte, dataLen)
	
	buffer.WriteUint32(nMsg.ChannelId, dataBuffer, &cursor)
	buffer.WriteString(nMsg.From, dataBuffer, &cursor)
	buffer.WriteByte(nMsg.Subtype, dataBuffer, &cursor)
	buffer.WriteString(nMsg.Text, dataBuffer, &cursor)

	msg := net.NewMsg()
	msg.SetMsgType(this.Signature())
	msg.SetPayload(dataBuffer)

	return msg, nil
}

// Signature returns Msg's network signature (prod.CHAT_MSG).
func (this *MsgHandler) Signature() uint16 {
	return prod.CHAT_MSG
}
