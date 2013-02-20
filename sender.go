package zabbix_sender

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
)

var (
	header = []byte("ZBXD\x01")
)

type dataItem struct {
	Host  string `json:"host"`
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Sender struct {
	Host string
}

func convertValue(i interface{}) string {
	switch v := i.(type) {
	case float32, float64:
		return fmt.Sprintf("%.6f", v)
	default:
		return fmt.Sprint(v)
	}
	panic("not reached")
}

func (s *Sender) Convert(kv map[string]interface{}) (b []byte, err error) {
	data := make([]dataItem, len(kv))
	i := 0
	for k, v := range kv {
		data[i] = dataItem{s.Host, k, convertValue(v)}
		i++
	}

	d, err := json.Marshal(data)
	if err == nil {
		l := uint64(len(d))
		b = make([]byte, 0, 46+l) // 5 + 8 + 32 + l + 1
		buf := bytes.NewBuffer(b)
		buf.Write(header)                                   // 5
		err = binary.Write(buf, binary.LittleEndian, l)     // 8
		buf.WriteString(`{"request":"sender data","data":`) // 32
		buf.Write(d)                                        // l
		buf.WriteByte('}')                                  // 1
		b = buf.Bytes()
	}
	return
}
