// Package zabbix_sender provides interface to send data to Zabbix server.
//
// It works similar to Zabbix's own zabbix_sender
// (https://www.zabbix.com/documentation/1.8/manpages/zabbix_sender).
package zabbix_sender

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"
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
	Conn *net.TCPConn
	host string
	m    sync.Mutex
}

// Creates new Sender for reporting data from given hostname.
// If hostname is empty, tries get it from os.Hostname(). Panics as last resport.
func NewSender(hostname string) *Sender {
	var err error
	if hostname == "" {
		hostname, err = os.Hostname()
	}
	if hostname == "" {
		panic(err)
	}
	return &Sender{host: hostname}
}

// Converts value to format accepted by Zabbix server.
func convertValue(i interface{}) string {
	switch v := i.(type) {
	case float32, float64:
		return fmt.Sprintf("%.6f", v)
	default:
		return fmt.Sprint(v)
	}
	panic("not reached")
}

// Converts data to format accepted by Zabbix server.
// It's like dense JSON with binary header, somewhat documented there:
// https://www.zabbix.com/documentation/1.8/protocols/agent
func (s *Sender) Convert(kv map[string]interface{}) (b []byte, err error) {
	// convert data - order is not important
	data := make([]dataItem, len(kv))
	i := 0
	for k, v := range kv {
		data[i] = dataItem{s.host, k, convertValue(v)}
		i++
	}

	d, err := json.Marshal(data)
	if err == nil {
		// there order of fields in "JSON" is important - request should be before data
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

// TODO
func (s *Sender) Dial(addr string) (err error) {
	a, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		return
	}

	c, err := net.DialTCP("tcp", nil, a)
	if err != nil {
		return
	}

	s.m.Lock()
	defer s.m.Unlock()
	s.Conn = c
	return
}

// TODO
func (s *Sender) Close() (err error) {
	s.m.Lock()
	defer s.m.Unlock()
	if s.Conn != nil {
		err = s.Conn.Close()
		if err != nil {
			s.Conn = nil
		}
	}
	return
}

// TODO
func (s *Sender) Send(kv map[string]interface{}) (err error) {
	b, err := s.Convert(kv)
	if err != nil {
		return
	}
	n, err := s.Conn.Write(b)
	if err != nil {
		return
	}
	if n < len(b) {
		return io.ErrShortWrite
	}
	return
}
