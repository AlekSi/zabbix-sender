package zabbix_sender_test

import (
	. "."
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

func TestConvert(t *testing.T) {
	var (
		sender         = NewSender("localhost")
		i      int     = 42
		f0     float32 = 0
		f1     float32 = 1
		f32    float32 = 3.141592653
		f64    float64 = 3.141592653
		s              = "string"
		e      error   = fmt.Errorf("error")
	)
	b, err := sender.Convert(map[string]interface{}{"int": i, "f0": f0, "f1": f1, "float32": f32, "float64": f64, "s": s, "e": e})
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", b)
	if len(b) != cap(b) {
		t.Errorf("Optimize: len(%d) != cap(%d)", len(b), cap(b))
	}

	if string(b[0:5]) != "ZBXD\x01" {
		t.Error("Wrong header")
	}
	if string(b[5:13]) != "\x64\x01\x00\x00\x00\x00\x00\x00" {
		t.Errorf("Wrong size: %x", b[5:13])
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
	data := m["data"].([]interface{})
	for i := range data {
		d := data[i].(map[string]interface{})
		if d["host"].(string) != "localhost" {
			t.Error("Wrong host")
		}
		switch d["key"].(string) {
		case "int":
			if d["value"].(string) != "42" {
				t.Errorf("Wrong value %#v", d)
			}
		case "f0":
			if d["value"].(string) != "0.000000" {
				t.Errorf("Wrong value %#v", d)
			}
		case "f1":
			if d["value"].(string) != "1.000000" {
				t.Errorf("Wrong value %#v", d)
			}
		case "float32":
			if d["value"].(string) != "3.141593" {
				t.Errorf("Wrong value %#v", d)
			}
		case "float64":
			if d["value"].(string) != "3.141593" {
				t.Errorf("Wrong value %#v", d)
			}
		case "s":
			if d["value"].(string) != "string" {
				t.Errorf("Wrong value %#v", d)
			}
		case "e":
			if d["value"].(string) != "error" {
				t.Errorf("Wrong value %#v", d)
			}
		default:
			t.Fatal("default reached")
		}
	}
}
