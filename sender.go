package zabbix_sender

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"net"
)

// Unexpected header of Zabbix's response.
var ErrBadHeader = errors.New("bad header")

type Response struct {
	Response string `json:"response"` // "success" on success
	Info     string `json:"info"`     // String like ""Processed 2 Failed 0 Total 2 Seconds spent 0.000034""
}

// Send DataItems to Zabbix server and wait for response.
// Returns encountered fatal error like I/O and marshalling/unmarshalling.
// Caller should inspect response (and in some situations also Zabbix server log)
// to check if all items are accepted.
func Send(addr *net.TCPAddr, di DataItems) (res *Response, err error) {
	b, err := di.Marshal()
	if err != nil {
		return
	}

	// Zabbix doesn't support persistent connections, so open/close it every time.
	conn, err := net.DialTCP(addr.Network(), nil, addr)
	if err != nil {
		return
	}
	defer conn.Close()

	_, err = conn.Write(b)
	if err != nil {
		return
	}

	buf := make([]byte, 8)
	_, err = io.ReadFull(conn, buf[:5])
	if err != nil {
		return
	}
	if !bytes.Equal(buf[:5], header) {
		err = ErrBadHeader
		return
	}

	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return
	}
	var datalen uint64
	err = binary.Read(bytes.NewBuffer(buf), binary.LittleEndian, &datalen)
	if err != nil {
		err = ErrBadHeader
		return
	}

	buf = make([]byte, datalen)
	_, err = io.ReadFull(conn, buf)
	if err != nil {
		return
	}

	res = new(Response)
	err = json.Unmarshal(buf, res)
	return
}
