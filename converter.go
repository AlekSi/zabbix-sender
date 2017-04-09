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
	Hostname  string `json:"host"`
	Key       string `json:"key"`
	Timestamp int64  `json:"clock,omitempty"` // UNIX timestamp, 0 is ignored
	Value     string `json:"value"`           // Use ConvertValue() to fill
}

type DataItems []DataItem

// Convert key/value pairs to DataItems using ConvertValue().
// Each DataItem's Host is set to hostname, in case Timestamp is 0 - it gonna be omitted.
func MakeDataItems(kv map[string]interface{}, hostname string, timestamp time.Time) DataItems {
	di := make(DataItems, len(kv))
	i := 0
	for k, v := range kv {
		di[i] = DataItem{hostname, k, timestamp.Unix(), ConvertValue(v)}
		i++
	}

	return di
}

// Converts filled DataItems to format accepted by Zabbix server.
// It's like dense JSON with binary header, somewhat documented there:
// https://www.zabbix.com/documentation/2.0/manual/appendix/items/activepassive
func (di DataItems) Marshal() (b []byte, err error) {
	d, err := json.Marshal(di)
	if err == nil {
		// the order of fields in this "JSON" is important - request should be before data
		now := fmt.Sprint(time.Now().Unix())
		datalen := uint64(len(d) + len(now) + 42) // 32 + d + 9 + now + 1
		b = make([]byte, 0, datalen+13)           // datalen + 5 + 8
		buf := bytes.NewBuffer(b)
		buf.Write(header)                                     // 5
		err = binary.Write(buf, binary.LittleEndian, datalen) // 8
		buf.WriteString(`{"request":"sender data","data":`)   // 32
		buf.Write(d)                                          // d
		buf.WriteString(`,"clock":`)                          // 9
		buf.WriteString(now)                                  // now
		buf.WriteByte('}')                                    // 1
		b = buf.Bytes()
	}
	return
}
