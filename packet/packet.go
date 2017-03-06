package packet

import (
	"github.com/pquerna/ffjson/ffjson"
)

type Packet struct {
	Data []byte
}

func NewPacket(data []byte) *Packet {
	return &Packet{Data: data}
}

//序列化
func MashallPackets(datas []*Packet) ([]byte, error) {
	return ffjson.Marshal(datas)
}

//反序列化
func UnmashallPackets(b []byte) (p []*Packet, err error) {
	err = ffjson.Unmarshal(b, p)
	return
}
