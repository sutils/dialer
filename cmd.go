package dialer

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"sync/atomic"

	"github.com/Centny/gwf/log"
	"github.com/Centny/gwf/util"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

//CmdCtrlC is the ctrl-c command on telnet
var CmdCtrlC = []byte{255, 244, 255, 253, 6}

//CmdStdinWriter is writer to handler charset replace and close command.
type CmdStdinWriter struct {
	io.Writer
	Replace  []byte
	CloseTag []byte
}

func (c *CmdStdinWriter) Write(p []byte) (n int, err error) {
	if len(c.CloseTag) > 0 {
		newp := bytes.Replace(p, c.CloseTag, []byte{}, -1)
		if len(newp) != len(p) {
			err = fmt.Errorf("closed")
			return 0, err
		}
	}
	n = len(p)
	if len(c.Replace) > 0 {
		p = bytes.Replace(p, c.Replace, []byte{}, -1)
	}
	_, err = c.Writer.Write(p)
	return
}

//CmdDialer is an implementation of the Dialer interface for dial command
type CmdDialer struct {
	Replace  []byte
	CloseTag []byte
	PS1      string
	Dir      string
	LC       string
	Prefix   string
}

//NewCmdDialer will return new CmdDialer
func NewCmdDialer() *CmdDialer {
	return &CmdDialer{
		Replace:  []byte("\r"),
		CloseTag: CmdCtrlC,
	}
}

//Name will return dialer name
func (c *CmdDialer) Name() string {
	return "Cmd"
}

//Bootstrap the dilaer
func (c *CmdDialer) Bootstrap(options util.Map) error {
	if options != nil {
		c.PS1 = options.StrVal("PS1")
		c.Dir = options.StrVal("Dir")
		c.LC = options.StrVal("LC")
		c.Prefix = options.StrVal("Prefix")
	}
	return nil
}

//Matched will return wheter uri is invalid uril.
func (c *CmdDialer) Matched(uri string) bool {
	target, err := url.Parse(uri)
	return err == nil && target.Scheme == "tcp" && target.Host == "cmd"
}

//Dial will start command and pipe to stdin/stdout
func (c *CmdDialer) Dial(sid uint64, uri string) (raw io.ReadWriteCloser, err error) {
	remote, err := url.Parse(uri)
	if err != nil {
		return
	}
	runnable := remote.Query().Get("exec")
	log.D("CmdDialer dial to cmd:%v", runnable)
	cmd := NewCmd("Cmd", c.PS1, runnable)
	if len(c.Prefix) > 0 {
		cmd.Prefix = bytes.NewBuffer([]byte(c.Prefix))
	}
	cmd.PS1 = c.PS1
	cmd.Dir = c.Dir
	ps1 := remote.Query().Get("PS1")
	if len(ps1) > 0 {
		cmd.PS1 = ps1
	}
	cmd.Dir = remote.Query().Get("dir")
	cmd.Cols, cmd.Rows = 80, 60
	util.ValidAttrF(`cols,O|I,R:0;rows,O|I,R:0;`, remote.Query().Get, true, &cmd.Cols, &cmd.Rows)
	lc := remote.Query().Get("LC")
	if len(lc) < 1 {
		lc = c.LC
	}
	switch lc {
	case "zh_CN.GBK":
		raw = &CombinedReadWriterCloser{
			Reader: transform.NewReader(cmd, simplifiedchinese.GBK.NewDecoder()),
			Writer: transform.NewWriter(cmd, simplifiedchinese.GBK.NewEncoder()),
			Closer: cmd.Close,
		}
	case "zh_CN.GB18030":
		raw = &CombinedReadWriterCloser{
			Reader: transform.NewReader(cmd, simplifiedchinese.GB18030.NewDecoder()),
			Writer: transform.NewWriter(cmd, simplifiedchinese.GB18030.NewEncoder()),
			Closer: cmd.Close,
		}
	default:
		raw = cmd
	}
	err = cmd.Start()
	return
}

func (c *CmdDialer) String() string {
	return "Cmd"
}

//CombinedReadWriterCloser is an implementation of io.ReadWriteClose to combined reader/writer/closer
type CombinedReadWriterCloser struct {
	io.Reader
	io.Writer
	Closer func() error
	closed uint32
}

//Close will call closer only once
func (c *CombinedReadWriterCloser) Close() (err error) {
	if !atomic.CompareAndSwapUint32(&c.closed, 0, 1) {
		return fmt.Errorf("closed")
	}
	if c.Closer != nil {
		err = c.Closer()
	}
	return
}
