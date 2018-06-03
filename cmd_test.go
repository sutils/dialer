package dialer

import (
	"fmt"
	"io"
	"os"
	"testing"
	"time"
)

func TestCmdDialer(t *testing.T) {
	cmd := NewCmdDialer()
	cmd.PS1 = "CmdDialer"
	cmd.Prefix = []byte(`echo testing`)
	cmd.Bootstrap()
	if !cmd.Matched("tcp://cmd?exec=/bin/bash") {
		t.Error("error")
		return
	}
	raw, err := cmd.Dial(10, "tcp://cmd?exec=/bin/bash")
	if err != nil {
		t.Error(err)
		return
	}
	go io.Copy(os.Stdout, raw)
	fmt.Fprintf(raw, "ls\n")
	fmt.Fprintf(raw, "ls /tmp/\n")
	fmt.Fprintf(raw, "echo abc\n")
	time.Sleep(200 * time.Millisecond)
	raw.Write(CmdCtrlC)
	time.Sleep(200 * time.Millisecond)
	raw.Close()
	time.Sleep(200 * time.Millisecond)
	//for cover
	fmt.Printf("%v\n", cmd)
	//
	//test encoding
	raw, err = cmd.Dial(10, "tcp://cmd?exec=/bin/bash&LC=zh_CN.GBK")
	if err != nil {
		t.Error(err)
		return
	}
	go io.Copy(os.Stdout, raw)
	fmt.Fprintf(raw, "ls\n")
	time.Sleep(200 * time.Millisecond)
	raw.Close()
	//
	raw, err = cmd.Dial(10, "tcp://cmd?exec=/bin/bash&LC=zh_CN.GB18030")
	if err != nil {
		t.Error(err)
		return
	}
	go io.Copy(os.Stdout, raw)
	fmt.Fprintf(raw, "ls\n")
	time.Sleep(200 * time.Millisecond)
	raw.Close()
}

func TestCmdDialer2(t *testing.T) {
	cmd := NewCmdDialer()
	cmd.PS1 = "CmdDialer"
	cmd.Prefix = []byte(`echo testing`)
	cmd.Bootstrap()
	if !cmd.Matched("tcp://cmd?exec=bash") {
		t.Error("error")
		return
	}
	raw, err := cmd.Dial(10, "tcp://cmd?exec=bash")
	if err != nil {
		t.Error(err)
		return
	}
	go io.Copy(os.Stdout, raw)
	fmt.Fprintf(raw, "ls\n")
	fmt.Fprintf(raw, "ls /tmp/\n")
	fmt.Fprintf(raw, "echo abc\n")
	time.Sleep(200 * time.Millisecond)
	raw.Write(CmdCtrlC)
	time.Sleep(200 * time.Millisecond)
	raw.Close()
	time.Sleep(200 * time.Millisecond)
	//for cover
	fmt.Printf("%v\n", cmd)
	//
	//test encoding
	raw, err = cmd.Dial(10, "tcp://cmd?exec=bash&LC=zh_CN.GBK")
	if err != nil {
		t.Error(err)
		return
	}
	go io.Copy(os.Stdout, raw)
	fmt.Fprintf(raw, "ls\n")
	time.Sleep(200 * time.Millisecond)
	raw.Close()
	//
	raw, err = cmd.Dial(10, "tcp://cmd?exec=bash&LC=zh_CN.GB18030")
	if err != nil {
		t.Error(err)
		return
	}
	go io.Copy(os.Stdout, raw)
	fmt.Fprintf(raw, "ls\n")
	time.Sleep(200 * time.Millisecond)
	raw.Close()
	//
	//test error
	_, err = cmd.Dial(100, "://cmd")
	if err == nil {
		t.Error("error")
		return
	}
}
