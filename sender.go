package zabbix_sender

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"regexp"
	"strconv"
)

// Unexpected header of Zabbix's response.
var ErrBadHeader = errors.New("bad header")

type Response struct {
	Response  string `json:"response"` // "success" on success
	Info      string `json:"info"`     // String like "Processed 2 Failed 1 Total 3 Seconds spent 0.000034"
	Processed int    // Filled by parsing Info
	Failed    int    // Filled by parsing Info
}

var infoRE = regexp.MustCompile(`(?i)Processed:? (\d+);? Failed:? (\d+)`)

type TlsParams struct {
	TLSCAFile            string
	TLSCertFile          string
	TLSKeyFile           string
	TLSServerName        string
	/* FIXME: support this
	TLSServerCertIssuer  string
	TLSServerCertSubject string
	*/
}

// Send DataItems to Zabbix server and wait for response.
// Returns encountered fatal error like I/O and marshalling/unmarshalling.
// Caller should inspect response (and in some situations also Zabbix server log)
// to check if all items are accepted.
func SendTls(addr *net.TCPAddr, di DataItems, tlsparams *TlsParams) (res *Response, err error) {
	// Load client cert
	cert, err := tls.LoadX509KeyPair(tlsparams.TLSCertFile, tlsparams.TLSKeyFile)
	if err != nil {
		return
	}
	// Load CA cert
	caCert, err := ioutil.ReadFile(tlsparams.TLSCAFile)
	if err != nil {
		return
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	config := tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs: caCertPool,
		InsecureSkipVerify: len(tlsparams.TLSServerName) == 0,
		ServerName: tlsparams.TLSServerName,
		/* FIXME!!
		CipherSuites: []uint16 {
			tls.TLS_RSA_WITH_RC4_128_SHA,
			tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
			tls.TLS_RSA_WITH_AES_128_CBC_SHA,
			tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		},
		*/
	}

	b, err := di.Marshal()
	if err != nil {
		return
	}

	// Zabbix doesn't support persistent connections, so open/close it every time.
	conn, err := tls.Dial(addr.Network(), addr.String(), &config)
	if err != nil {
		return
	}
	defer conn.Close()

	return send(conn, b)
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

	return send(conn, b)
}

func send(conn net.Conn, b []byte) (res *Response, err error) {
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
	if err == nil {
		m := infoRE.FindStringSubmatch(res.Info)
		if len(m) == 3 {
			p, _ := strconv.Atoi(m[1])
			f, _ := strconv.Atoi(m[2])
			res.Processed = p
			res.Failed = f
		}
	}
	return
}
