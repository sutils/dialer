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
}

//NewTCPDialer will return new TCPDialer
func NewTCPDialer() *TCPDialer {
	return &TCPDialer{
		portMatcher: regexp.MustCompile("^.*:[0-9]+$"),
	}
}

//Name will return dialer name
func (t *TCPDialer) Name() string {
	return "TCP"
}

//Bootstrap the dialer.
func (t *TCPDialer) Bootstrap(options util.Map) error {
	return nil
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
		raw, err = net.Dial(network, host)
	}
	return
}

func (t *TCPDialer) String() string {
	return "TCPDialer"
}
