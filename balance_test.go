package dialer

import (
	"fmt"
	"io"
	"testing"

	"github.com/Centny/gwf/util"
)

type OnceDialer struct {
	ID     string
	dialed int
	conf   util.Map
}

func (o *OnceDialer) Name() string {
	return o.ID
}

//initial dialer
func (o *OnceDialer) Bootstrap(options util.Map) error {
	o.ID = options.StrVal("id")
	if len(o.ID) < 1 {
		return fmt.Errorf("id is required")
	}
	o.conf = options
	return nil
}

//
func (o *OnceDialer) Options() util.Map {
	return o.conf
}

//match uri
func (o *OnceDialer) Matched(uri string) bool {
	return uri == "once"
}

//dial raw connection
func (o *OnceDialer) Dial(sid uint64, uri string) (r io.ReadWriteCloser, err error) {
	r = o
	o.dialed++
	if o.dialed > 1 {
		err = fmt.Errorf("dialed")
	}
	return
}

func (o *OnceDialer) Read(p []byte) (n int, err error) {
	return
}

func (o *OnceDialer) Write(p []byte) (n int, err error) {
	return
}

func (o *OnceDialer) Close() error {
	return nil
}

func TestBalancedDialerDefaul(t *testing.T) {
	NewDialer = func(t string) Dialer {
		return &OnceDialer{}
	}
	dialer := NewBalancedDialer()
	err := dialer.Bootstrap(util.Map{
		"id":      "t1",
		"matcher": ".*",
		"timeout": 500,
		"delay":   100,
		"dialers": []util.Map{
			{
				"id":          "i0",
				"type":        "once",
				"fail_remove": 2,
			},
			{
				"id":          "i1",
				"type":        "once",
				"fail_remove": 3,
			},
			{
				"id":          "i2",
				"type":        "once",
				"fail_remove": 4,
			},
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	_, err = dialer.Dial(uint64(4), "not")
	if err == nil {
		t.Error(err)
		return
	}
	for i := 0; i < 3; i++ {
		_, err = dialer.Dial(uint64(i), "once")
		if err != nil {
			t.Error(err)
			return
		}
	}
	_, err = dialer.Dial(uint64(4), "once")
	if err == nil {
		t.Error(err)
		return
	}
	NewDialer = DefaultDialerCreator
	//
	//
	dialer = NewBalancedDialer()
	dialer.Name()
	dialer.Options()
	dialer.AddDialer(NewTCPDialer())
	//
	//test error

	//id not found
	err = dialer.Bootstrap(util.Map{})
	if err == nil {
		t.Error(err)
		return
	}
	//policy error
	err = dialer.Bootstrap(util.Map{
		"id": "t0",
		"policy": []util.Map{
			{
				"matcher": "[",
				"limit":   []int64{},
			},
		},
	})
	if err == nil {
		t.Error(err)
		return
	}
	//dialer type error
	err = dialer.Bootstrap(util.Map{
		"id": "t0",
		"dialers": []util.Map{
			{
				"type": "xx",
			},
		},
	})
	if err == nil {
		t.Error(err)
		return
	}
	//dialer bootstrap error
	err = dialer.Bootstrap(util.Map{
		"id": "t0",
		"dialers": []util.Map{
			{
				"type": "balance",
			},
		},
	})
	if err == nil {
		t.Error(err)
		return
	}

	//
	err = dialer.AddPolicy(".*", []int64{})
	if err == nil {
		t.Error(err)
		return
	}
	err = dialer.AddPolicy("[.*", []int64{})
	if err == nil {
		t.Error(err)
		return
	}
}

type TimeDialer struct {
	ID     string
	dialed int
	conf   util.Map
	last   int64
}

func (o *TimeDialer) Name() string {
	return o.ID
}

//initial dialer
func (o *TimeDialer) Bootstrap(options util.Map) error {
	o.ID = options.StrVal("id")
	if len(o.ID) < 1 {
		return fmt.Errorf("id is required")
	}
	o.conf = options
	return nil
}

//
func (o *TimeDialer) Options() util.Map {
	return o.conf
}

//match uri
func (o *TimeDialer) Matched(uri string) bool {
	return uri == "time"
}

//dial raw connection
func (t *TimeDialer) Dial(sid uint64, uri string) (r io.ReadWriteCloser, err error) {
	if util.Now()-t.last < 100 {
		panic("too fast")
	}
	r = t
	t.last = util.Now()
	return
}

func (o *TimeDialer) Read(p []byte) (n int, err error) {
	return
}

func (o *TimeDialer) Write(p []byte) (n int, err error) {
	return
}

func (o *TimeDialer) Close() error {
	return nil
}

func TestBalancedDialerPolicy(t *testing.T) {
	NewDialer = func(t string) Dialer {
		return &TimeDialer{}
	}
	defer func() {
		NewDialer = DefaultDialerCreator
	}()
	dialer := NewBalancedDialer()
	err := dialer.Bootstrap(util.Map{
		"id":      "t1",
		"matcher": ".*",
		"timeout": 1000,
		"delay":   100,
		"dialers": []util.Map{
			{
				"id":          "i0",
				"type":        "time",
				"fail_remove": 2,
			},
			{
				"id":          "i1",
				"type":        "time",
				"fail_remove": 3,
			},
			{
				"id":          "i2",
				"type":        "time",
				"fail_remove": 4,
			},
		},
		"policy": []util.Map{
			{
				"matcher": ".*",
				"limit":   []int{110, 1},
			},
		},
	})
	if err != nil {
		t.Error(err)
		return
	}
	_, err = dialer.Dial(uint64(4), "not")
	if err == nil {
		t.Error(err)
		return
	}
	for i := 0; i < 10; i++ {
		_, err = dialer.Dial(uint64(i), "time")
		if err != nil {
			t.Errorf("%v->%v", i, err)
			return
		}
	}
}
