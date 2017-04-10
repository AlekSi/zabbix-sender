package zabbix_sender_test

import (
	. "."
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

var (
	f0   float32 = 0
	f1   float32 = 1
	f32  float32 = 3.141592653
	f64  float64 = 3.141592653
	i    int     = 42
	s    string  = "string"
	e    error   = fmt.Errorf("%v", "error")
	p0   *string
	data = map[string]interface{}{"f0": f0, "f1": f1, "float32": f32, "float64": f64,
		"int": i, "s": s, "e": e, "p0": p0}
)

func TestMakeDataItems(t *testing.T) {
	di := MakeDataItems(data, "localhost", time.Unix(0, 0), time.Unix(0, 0))
	t.Logf("%+v", di)
	for _, d := range di {
		if d.Hostname != "localhost" {
			t.Error("Wrong hostname")
		}
		if d.Timestamp != 0 {
			t.Error("Wrong timestamp")
		}
		switch d.Key {
		case "f0":
			if d.Value != "0.000000" {
				t.Errorf("Wrong value %#v", d)
			}
		case "f1":
			if d.Value != "1.000000" {
				t.Errorf("Wrong value %#v", d)
			}
		case "float32":
			if d.Value != "3.141593" {
				t.Errorf("Wrong value %#v", d)
			}
		case "float64":
			if d.Value != "3.141593" {
				t.Errorf("Wrong value %#v", d)
			}
		case "int":
			if d.Value != "42" {
				t.Errorf("Wrong value %#v", d)
			}
		case "s":
			if d.Value != "string" {
				t.Errorf("Wrong value %#v", d)
			}
		case "e":
			if d.Value != "error" {
				t.Errorf("Wrong value %#v", d)
			}
		case "p0":
			if d.Value != "<nil>" {
				t.Errorf("Wrong value %#v", d)
			}
		default:
			t.Fatal("default reached")
		}
	}
}

func TestMarshal(t *testing.T) {
	di := MakeDataItems(data, "localhost", time.Now(), time.Now())
	b, err := di.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", b)

	if len(b) != cap(b) {
		t.Errorf("Optimize: len(%d) != cap(%d)", len(b), cap(b))
	}

	if string(b[:5]) != "ZBXD\x01" {
		t.Error("Wrong header")
	}
	var datalen uint64
	err = binary.Read(bytes.NewBuffer(b[5:13]), binary.LittleEndian, &datalen)
	if err != nil {
		t.Fatal(err)
	}
	if datalen != uint64(len(b))-13 {
		t.Errorf("Wrong size: %d (expected %d)", datalen, len(b)-13)
	}

	b = b[13:]
	str := string(b)
	if !strings.HasPrefix(str, `{"request":"sender data","data":[{`) ||
		strings.Contains(str, `, `) ||
		strings.Contains(str, `: `) {
		t.Error("Zabbix's JSON parser will not parse it:", str)
	}

	var m map[string]interface{}
	err = json.Unmarshal(b, &m)
	if err != nil {
		t.Fatal(err)
	}
}
