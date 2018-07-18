package dialer

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/Centny/gwf/util"
)

//SocksProxyDialer is an implementation of the Dialer interface for dial by socks proxy.
type SocksProxyDialer struct {
	Address string
	matcher *regexp.Regexp
}

//NewSocksProxyDialer will return new SocksProxyDialer
func NewSocksProxyDialer() *SocksProxyDialer {
	return &SocksProxyDialer{
		matcher: regexp.MustCompile("^.*:[0-9]+$"),
	}
}

//Name will return dialer name
func (s *SocksProxyDialer) Name() string {
	return "Socks"
}

//Bootstrap the dialer.
func (s *SocksProxyDialer) Bootstrap(options util.Map) error {
	if options != nil {
		s.Address = options.StrVal("address")
	}
	return nil
}

//Matched will return whether the uri is invalid tcp uri.
func (s *SocksProxyDialer) Matched(uri string) bool {
	remote, err := url.Parse(uri)
	return err == nil && s.matcher.MatchString(remote.Host)
}

//Dial one connection by uri
func (s *SocksProxyDialer) Dial(sid uint64, uri string) (raw io.ReadWriteCloser, err error) {
	remote, err := url.Parse(uri)
	if err != nil {
		return
	}
	parts := strings.SplitAfterN(remote.Host, ":", 2)
	if len(parts) < 2 {
		err = fmt.Errorf("not supported address:%v", remote.Host)
		return
	}
	host := parts[0]
	port, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		err = fmt.Errorf("parse address:%v error:%v", remote.Host, err)
		return
	}
	conn, err := net.Dial("tcp", s.Address)
	if err != nil {
		return
	}
	_, err = conn.Write([]byte{0x05, 0x01, 0x00})
	if err != nil {
		conn.Close()
		return
	}
	buf := make([]byte, 1024*64)
	err = fullBuf(conn, buf, 2, nil)
	if err != nil {
		conn.Close()
		return
	}
	if buf[0] != 0x05 || buf[1] != 0x00 {
		err = fmt.Errorf("unsupported %x", buf)
		conn.Close()
		return
	}
	blen := len(host) + 7
	buf[0], buf[1], buf[2] = 0x05, 0x01, 0x00
	buf[3], buf[4] = 0x03, byte(len(host))
	copy(buf[5:], []byte(host))
	buf[blen-2] = byte(port * 256)
	buf[blen-1] = byte(port % 256)
	_, err = conn.Write(buf[:blen])
	if err != nil {
		conn.Close()
		return
	}
	err = fullBuf(conn, buf, 5, nil)
	if err != nil {
		conn.Close()
		return
	}
	switch buf[3] {
	case 0x01:
		err = fullBuf(conn, buf, 5, nil)
	case 0x03:
		err = fullBuf(conn, buf, uint32(buf[4])+2, nil)
	case 0x04:
		err = fullBuf(conn, buf, 17, nil)
	default:
		err = fmt.Errorf("reply address type is not supported:%v", buf[3])
	}
	if err != nil {
		conn.Close()
		return
	}
	raw = conn
	return
}

func (s *SocksProxyDialer) String() string {
	return "SocksProxyDialer"
}
