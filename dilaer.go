package dialer

import (
	"fmt"
	"io"

	"github.com/Centny/gwf/util"
)

// Dialer is the interface that wraps the dialer
type Dialer interface {
	Name() string
	//initial dialer
	Bootstrap(options util.Map) error
	//
	Options() util.Map
	//match uri
	Matched(uri string) bool
	//dial raw connection
	Dial(sid uint64, uri string) (r io.ReadWriteCloser, err error)
}

//Pool is the set of Dialer
type Pool struct {
	Dialers []Dialer
}

//NewPool will return new Pool
func NewPool() (pool *Pool) {
	pool = &Pool{}
	return
}

//AddDialer will append dialer which is bootstraped to pool
func (p *Pool) AddDialer(dialers ...Dialer) (err error) {
	p.Dialers = append(p.Dialers, dialers...)
	return
}

func (p *Pool) Bootstrap(options util.Map) error {
	dialerOptions := options.AryMapVal("dialers")
	for _, option := range dialerOptions {
		dtype := option.StrVal("type")
		dialer := NewDialer(dtype)
		if dialer == nil {
			return fmt.Errorf("create dialer fail by %v", util.S2Json(option))
		}
		err := dialer.Bootstrap(option)
		if err != nil {
			return err
		}
		p.Dialers = append(p.Dialers, dialer)
	}
	if options.IntValV("standard", 0) > 0 {
		p.Dialers = append(p.Dialers, NewCmdDialer(), NewEchoDialer(),
			NewWebDialer(), NewTCPDialer())
	}
	return nil
}

//Dial the uri by dialer poo
func (p *Pool) Dial(sid uint64, uri string) (r io.ReadWriteCloser, err error) {
	for _, dialer := range p.Dialers {
		if dialer.Matched(uri) {
			r, err = dialer.Dial(sid, uri)
			return
		}
	}
	err = fmt.Errorf("uri(%v) is not supported(not matched dialer)", uri)
	return
}

func DefaultDialerCreator(t string) (dialer Dialer) {
	switch t {
	case "balance":
		dialer = NewBalancedDialer()
	case "cmd":
		dialer = NewCmdDialer()
	case "echo":
		dialer = NewEchoDialer()
	case "socks":
		dialer = NewSocksProxyDialer()
	case "tcp":
		dialer = NewTCPDialer()
	case "web":
		dialer = NewWebDialer()
	}
	return
}

var NewDialer = DefaultDialerCreator
