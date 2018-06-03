package dialer

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"runtime"
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
	BASH     string
	PS1      string
	Prefix   []byte
}

//NewCmdDialer will return new CmdDialer
func NewCmdDialer() *CmdDialer {
	return &CmdDialer{
		Replace:  []byte("\r"),
		CloseTag: CmdCtrlC,
		BASH:     "bash",
	}
}

//Bootstrap the dilaer
func (c *CmdDialer) Bootstrap() error {
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
	if runnable == "bash" {
		cmd := NewCmd(c.BASH, c.PS1, c.BASH)
		if len(c.Prefix) > 0 {
			cmd.Prefix = bytes.NewBuffer(c.Prefix)
		}
		cmd.PS1 = c.PS1
		cmd.Cols, cmd.Rows = 80, 60
		util.ValidAttrF(`cols,O|I,R:0;rows,O|I,R:0;`, remote.Query().Get, true, &cmd.Cols, &cmd.Rows)
		err = cmd.Start()
		raw = cmd
		return
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("cmd", "/C", runnable)
	default:
		cmd = exec.Command(c.BASH, "-c", runnable)
	}
	retReader, stdWriter, err := os.Pipe()
	if err != nil {
		return
	}
	stdin, _ := cmd.StdinPipe()
	cmd.Stdout = stdWriter
	cmd.Stderr = stdWriter
	cmdWriter := &CmdStdinWriter{
		Writer:   stdin,
		Replace:  c.Replace,
		CloseTag: c.CloseTag,
	}
	combined := &CombinedReadWriterCloser{
		Writer: cmdWriter,
		Reader: retReader,
		Closer: func() error {
			log.D("CmdDialer will kill the cmd(%v)", sid)
			stdWriter.Close()
			stdin.Close()
			cmd.Process.Kill()
			return nil
		},
	}
	//
	switch remote.Query().Get("LC") {
	case "zh_CN.GBK":
		combined.Reader = transform.NewReader(combined.Reader, simplifiedchinese.GBK.NewDecoder())
		cmdWriter.Writer = transform.NewWriter(cmdWriter.Writer, simplifiedchinese.GBK.NewEncoder())
	case "zh_CN.GB18030":
		combined.Reader = transform.NewReader(combined.Reader, simplifiedchinese.GB18030.NewDecoder())
		cmdWriter.Writer = transform.NewWriter(cmdWriter.Writer, simplifiedchinese.GB18030.NewEncoder())
	default:
	}
	raw = combined
	err = cmd.Start()
	if err == nil {
		go func() {
			cmd.Wait()
			combined.Close()
		}()
	}
	return
}

func (c *CmdDialer) String() string {
	return "CmdDialer"
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
