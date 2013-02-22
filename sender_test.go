package zabbix_sender_test

import (
	"fmt"
	"net"

	. "."
)

func ExampleSend() {
	data := map[string]interface{}{"rpm": 42.12, "errors": 1}
	di := MakeDataItems(data, "localhost")
	addr, _ := net.ResolveTCPAddr("tcp", "localhost:10051")
	res, _ := Send(addr, di)
	fmt.Print(res)
}
