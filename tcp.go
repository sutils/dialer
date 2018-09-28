package dialer

import (
	"io"
	"net"
	"net/url"
	"regexp"

	"github.com/Centny/gwf/util"
)

//TCPDialer is an implementation of the Dialer interface for dial tcp connections.
type TCPDialer struct {
	portMatcher *regexp.Regexp
	conf        util.Map
}

//NewTCPDialer will return new TCPDialer
func NewTCPDialer() *TCPDialer {
	return &TCPDialer{
		portMatcher: regexp.MustCompile("^.*:[0-9]+$"),
		conf:        util.Map{},
	}
}

//Name will return dialer name
func (t *TCPDialer) Name() string {
	return "tcp"
}

//Bootstrap the dialer.
func (t *TCPDialer) Bootstrap(options util.Map) error {
	t.conf = options
	return nil
}

func (t *TCPDialer) Options() util.Map {
	return t.conf
}

//Matched will return whether the uri is invalid tcp uri.
func (t *TCPDialer) Matched(uri string) bool {
	_, err := url.Parse(uri)
	return err == nil
}

//Dial one connection by uri
func (t *TCPDialer) Dial(sid uint64, uri string) (raw io.ReadWriteCloser, err error) {
	remote, err := url.Parse(uri)
	if err == nil {
		var dialer net.Dialer
		bind := remote.Query().Get("bind")
		if len(bind) > 0 {
			dialer.LocalAddr, err = net.ResolveTCPAddr("tcp", bind)
			if err != nil {
				return
			}
		}
		network := remote.Scheme
		host := remote.Host
		switch network {
		case "http":
			network = "tcp"
			if !t.portMatcher.MatchString(host) {
				host += ":80"
			}
		case "https":
			network = "tcp"
			if !t.portMatcher.MatchString(host) {
				host += ":443"
			}
		}
		raw, err = dialer.Dial(network, host)
	}
	return
}

func (t *TCPDialer) String() string {
	return "TCPDialer"
}
