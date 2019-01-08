package ssh

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime"
	"strings"

	"github.com/dearcode/crab/log"
	"github.com/juju/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

//Client ssh客户端，支持scp的.
type Client struct {
	server string
	conn   *ssh.Client
}

//NewClient 创建ssh客户端.
func NewClient(host string, port int, user, passwd, keyFile string) (*Client, error) {
	conf := ssh.ClientConfig{
		User:            user,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	if passwd != "" {
		conf.Auth = []ssh.AuthMethod{ssh.Password(passwd)}
	}

	if keyFile != "" {
		key, err := ioutil.ReadFile(keyFile)
		if err != nil {
			return nil, errors.Trace(err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, errors.Trace(err)
		}

		conf.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	}

	server := fmt.Sprintf("%s:%d", host, port)

	conn, err := ssh.Dial("tcp", server, &conf)
	if err != nil {
		return nil, errors.Annotatef(err, server)
	}

	return &Client{conn: conn, server: server}, nil
}

//Exec 执行命令并等待返回结果.
func (c *Client) Exec(cmd string) (string, error) {
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 1<<16)
			// 获取所有goroutine的stacktrace, 如果只获取当前goroutine的stacktrace, 第二个参数需要为 `false`
			runtime.Stack(buf, true)
			log.Errorf("panic err:%v", err)
			log.Errorf("panic stack:%v", string(buf))
		}
	}()

	session, err := c.conn.NewSession()
	if err != nil {
		return "", errors.Annotatef(err, c.server)
	}
	defer session.Close()

	var bufErr, bufOut bytes.Buffer

	session.Stdout = &bufOut
	session.Stderr = &bufErr

	if err = session.Run(cmd); err != nil {
		return "", errors.Annotatef(err, bufOut.String()+bufErr.String())
	}

	return strings.TrimSpace(bufOut.String() + bufErr.String()), nil
}

//ExecPipe 执行命令并设置输出流.
func (c *Client) ExecPipe(cmdStr string, setPipe func(stdOut, stdErr io.Reader)) error {
	defer func() {
		if err := recover(); err != nil {
			buf := make([]byte, 1<<16)
			// 获取所有goroutine的stacktrace, 如果只获取当前goroutine的stacktrace, 第二个参数需要为 `false`
			runtime.Stack(buf, true)
			log.Errorf("panic err:%v", err)
			log.Errorf("panic stack:%v", string(buf))
		}
	}()

	session, err := c.conn.NewSession()
	if err != nil {
		return errors.Annotatef(err, c.server)
	}
	defer session.Close()

	stdErr, err := session.StderrPipe()
	if err != nil {
		return errors.Trace(err)
	}

	stdOut, err := session.StdoutPipe()
	if err != nil {
		return errors.Trace(err)
	}

	setPipe(stdOut, stdErr)

	return errors.Trace(session.Run(cmdStr))
}

//Upload 上传文件.
func (c *Client) Upload(src, dest string) error {
	sftp, err := sftp.NewClient(c.conn)
	if err != nil {
		return errors.Trace(err)
	}
	defer sftp.Close()

	st, err := os.Stat(src)
	if err != nil {
		return errors.Annotatef(err, "stat file:%v", src)
	}

	out, err := sftp.Create(dest)
	if err != nil {
		return errors.Annotatef(err, "create dest file:%v", dest)
	}
	defer out.Close()

	in, err := os.Open(src)
	if err != nil {
		return errors.Annotatef(err, "open src file:%v", src)
	}
	defer in.Close()

	if _, err = io.Copy(out, in); err != nil {
		return errors.Annotatef(err, "io copy src:%v, dest:%v", src, dest)
	}

	cmd := fmt.Sprintf("chmod %o %v", st.Mode(), dest)
	if _, err = c.Exec(cmd); err != nil {
		return errors.Annotatef(err, "cmd:%v", cmd)
	}

	return nil
}
