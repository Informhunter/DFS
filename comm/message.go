package comm

import (
	"bytes"
	"encoding/gob"
)

type MessageType int8

const (
	MessageTypeStatus MessageType = iota
	MessageTypeRequestUploadToken
	MessageTypeUploadToken
	MessageTypeRequestDownloadToken
	MessageTypeDownloadToken
)

func (mt MessageType) String() string {
	switch mt {
	case MessageTypeStatus:
		return "MessageTypeStatus"
	case MessageTypeRequestUploadToken:
		return "MessageTypeRequestUploadToken"
	case MessageTypeUploadToken:
		return "MessageTypeUploadToken"
	case MessageTypeRequestDownloadToken:
		return "MessageTypeRequestDownloadToken"
	case MessageTypeDownloadToken:
		return "MessageTypeDownloadToken"
	}
	return "Unknown"
}

type Message struct {
	Type       MessageType
	SourceNode string
	Data       []byte
}

/*EncodeData encodes data and stores it into byte array inside message struct.
Data must have a fixed-size type or be a struct of fixed-size types. */
func (msg *Message) EncodeData(data interface{}) error {
	buf := &bytes.Buffer{}
	enc := gob.NewEncoder(buf)
	err := enc.Encode(data)
	if err != nil {
		return err
	}
	msg.Data = buf.Bytes()
	return nil
}

/*DecodeData decodes data inside message and stores it in output variable.
Output variable's type must be pointer to type, that was originally encoded.
*/
func (msg *Message) DecodeData(output interface{}) error {
	buf := bytes.NewBuffer(msg.Data)
	dec := gob.NewDecoder(buf)
	err := dec.Decode(output)
	if err != nil {
		return err
	}
	return nil
}
