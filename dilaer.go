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

//AddDialer will run Dialer.Bootstrap, then append dialer to pool.
func (d *Pool) AddDialer(options util.Map, dialers ...Dialer) (err error) {
	for _, dialer := range dialers {
		err = dialer.Bootstrap(options.MapVal(dialer.Name()))
		if err != nil {
			break
		}
		d.Dialers = append(d.Dialers, dialer)
	}
	return
}

//Dial the uri by dialer poo
func (d *Pool) Dial(sid uint64, uri string) (r io.ReadWriteCloser, err error) {
	for _, dialer := range d.Dialers {
		if dialer.Matched(uri) {
			r, err = dialer.Dial(sid, uri)
			return
		}
	}
	err = fmt.Errorf("uri(%v) is not supported(not matched dialer)", uri)
	return
}
