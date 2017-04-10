package zabbix_sender

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"
)

var (
	header = []byte("ZBXD\x01")
)

// Converts value to format accepted by Zabbix server.
// It uses "%.6f" format for floats,
// and fmt.Sprint() (which will try String() and Error() methods) for other types.
// Keep in mind that Zabbix doesn't support negative integers, use floats instead.
func ConvertValue(i interface{}) string {
	switch v := i.(type) {
	case float32:
		return fmt.Sprintf("%.6f", v)
	case float64:
		return fmt.Sprintf("%.6f", v)
	default:
		return fmt.Sprint(v)
	}
	panic("not reached")
}

// Single Zabbix data item.
type DataItem struct {
	Hostname   string `json:"host"`
	Key        string `json:"key"`
	Value      string `json:"value"`           // Use ConvertValue() to fill
	Timestamp  int64  `json:"clock,omitempty"` // UNIX timestamp, 0 is ignored
	Nanosecond int    `json:"ns,omitempty"`    // UNIX nanoseconds timestamp, 0 is ignored
}

type DataItems []DataItem

// Convert key/value pairs to DataItems using ConvertValue().
// Each DataItem's Host is set to hostname, in case Timestamp is 0 - it gonna be omitted.
func MakeDataItems(kv map[string]interface{}, hostname string, timestamp time.Time, nanoseconds time.Time) DataItems {
	di := make(DataItems, len(kv))
	i := 0
	for k, v := range kv {
		di[i] = DataItem{hostname, k, ConvertValue(v), timestamp.Unix(), nanoseconds.Nanosecond()}
		i++
	}

	return di
}

// Converts filled DataItems to format accepted by Zabbix server.
// It's like dense JSON with binary header, somewhat documented there:
// https://www.zabbix.com/documentation/3.2/manual/appendix/items/activepassive
// and here: https://www.zabbix.org/wiki/Docs/protocols/zabbix_agent/3.0
func (di DataItems) Marshal() (b []byte, err error) {
	d, err := json.Marshal(di)
	if err == nil {
		// the order of fields in this "JSON" is important - request should be before data
		now := fmt.Sprint(time.Now().Unix())
		nowNs := fmt.Sprint(time.Now().Nanosecond())
		datalen := uint64(len(d) + len(now) + len(nowNs) + 48) // 32 + d + 9 + now + 6 + nowNs + 1
		b = make([]byte, 0, datalen+13)                        // datalen + 5 + 8
		buf := bytes.NewBuffer(b)
		buf.Write(header)                                     // 5
		err = binary.Write(buf, binary.LittleEndian, datalen) // 8
		buf.WriteString(`{"request":"sender data","data":`)   // 32
		buf.Write(d)                                          // d
		buf.WriteString(`,"clock":`)                          // 9
		buf.WriteString(now)                                  // now
		buf.WriteString(`,"ns":`)                             // 6
		buf.WriteString(nowNs)
		buf.WriteByte('}') // 1
		b = buf.Bytes()
	}
	return
}
