package dialer

import (
	"fmt"
	"io"
)

// Dialer is the interface that wraps the dialer
type Dialer interface {
	//initial dialer
	Bootstrap() error
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
