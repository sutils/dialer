package dialer

import (
	"fmt"
	"testing"
)

func TestTCPDialer(t *testing.T) {
	tcp := NewTCPDialer()
	tcp.Bootstrap(nil)
	if !tcp.Matched("tcp://localhost:80") {
		t.Error("error")
		return
	}
	if tcp.Matched("https://xx/%EX%B8%AD%E8%AF%AD%E8%A8%80") {
		t.Error("error")
		return
	}
	_, err := tcp.Dial(10, "http://localhost")
	if err != nil {
		t.Error(err)
		return
	}
	_, err = tcp.Dial(10, "https://www.baidu.com")
	if err != nil {
		t.Error(err)
		return
	}
	fmt.Printf("%v test done...", tcp)
}
