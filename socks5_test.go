package dialer

import (
	"testing"

	"github.com/Centny/gwf/util"
)

func TestSocksProxy(t *testing.T) {
	dailer := NewSocksProxyDialer()
	dailer.Bootstrap(util.Map{
		"address": "127.0.0.1:1080",
	})
	remote := "tcp://www.google.com:80"
	if !dailer.Matched(remote) {
		t.Error("error")
		return
	}
	raw, err := dailer.Dial(100, remote)
	if err != nil {
		t.Error(err)
		return
	}
	raw.Close()
}
