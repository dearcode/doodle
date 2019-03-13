package distributor

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/dearcode/crab/log"
	"github.com/dearcode/crab/orm"
	"github.com/juju/errors"

	"github.com/dearcode/doodle/distributor/config"
	"github.com/dearcode/doodle/util"
	"github.com/dearcode/doodle/util/ssh"
	"github.com/dearcode/doodle/util/uuid"
)

const (
	stateCompileBegin = iota + 1
	stateComplieSuccess
	stateComplieFailed
	stateInstallBegin
	stateInstallSuccess
	stateInstallFailed

	sqlWriteLogs   = "update distributor_logs set info = concat(info, ?) , state = ? where id=?"
	sqlUpdateState = "update distributor set state = ? where id=?"
)

var (
	scripts   = []string{"build.sh", "Dockerfile.tpl", "install.sh"}
	taskIDInc = uint32(0)
)

type task struct {
	wg      sync.WaitGroup
	ID      string
	path    string
	db      *sql.DB
	service service
	d       distributor
	state   int
	logID   int64
}

const (
	buildPathFormat = "20060102_150405"
)

func newTask(serviceID int64) (*task, error) {
	var p service

	db, err := mdb.GetConnection()
	if err != nil {
		return nil, errors.Trace(err)
	}

	if err = orm.NewStmt(db, "service").Where("service.id=%v", serviceID).Query(&p); err != nil {
		return nil, errors.Trace(err)
	}

	if err = orm.NewStmt(db, "node").Where("cluster_id=%d", p.Cluster.ID).Query(&p.Cluster.Node); err != nil {
		return nil, errors.Trace(err)
	}

	log.Debugf("service:%#v", p)

	path := fmt.Sprintf("%s/%v_%v", config.Distributor.Server.BuildPath, time.Now().Format(buildPathFormat), atomic.AddUint32(&taskIDInc, 1))
	if err = os.MkdirAll(path, os.ModePerm); err != nil {
		return nil, errors.Annotatef(err, path)
	}

	for _, f := range scripts {
		of := fmt.Sprintf("%s/%v", config.Distributor.Server.Script, f)
		nf := fmt.Sprintf("%s/%s", path, f)
		if err = os.Link(of, nf); err != nil {
			return nil, errors.Annotatef(err, "old:%v, new:%v", of, nf)
		}
	}

	d := distributor{
		ServiceID: serviceID,
		Server:    util.LocalAddr(),
	}

	if d.ID, err = orm.NewStmt(db, "distributor").Insert(&d); err != nil {
		log.Errorf("insert distributor:%v error:%v", d, err)
		return nil, errors.Trace(err)
	}

	return &task{db: db, service: p, d: d, ID: uuid.String(), path: path}, nil
}

func (t *task) updateState(state int) {
	t.state = state
	if _, err := orm.NewStmt(t.db, "").Exec(sqlUpdateState, state, t.d.ID); err != nil {
		log.Errorf("%v sqlUpdateState:%v, %v, %v, error:%v", t, sqlUpdateState, state, t.d.ID, errors.ErrorStack(err))
		return
	}
}

func (t *task) String() string {
	return t.ID
}

func (t *task) logStream(reader io.ReadCloser) {
	defer t.wg.Done()

	r := bufio.NewReader(reader)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			if err == io.EOF || strings.Contains(err.Error(), "file already closed") {
				return
			}
			log.Debugf("%v ReadLine %v", t, err)
			return
		}
		log.Infof("%v %s", t, line)
		if _, err := orm.NewStmt(t.db, "").Exec(sqlWriteLogs, string(line)+"\n", t.state, t.logID); err != nil {
			log.Errorf("%v update logs sql:%v, %s, %v, error:%v", t, sqlWriteLogs, line, t.logID, err)
			continue
		}
	}
}

func (t *task) writeSSHLogs(stdOut, stdErr io.Reader) {
	t.writeLogs(0, ioutil.NopCloser(stdOut), ioutil.NopCloser(stdErr))
}

func (t *task) writeLogs(pid int, stdOut, stdErr io.ReadCloser) {
	dl := distributorLogs{
		DistributorID: t.d.ID,
		PID:           pid,
		State:         t.state,
	}

	id, err := orm.NewStmt(t.db, "distributor_logs").Insert(&dl)
	if err != nil {
		log.Errorf("%v insert distributor_logs error:%v", t, errors.ErrorStack(err))
		return
	}

	t.logID = id

	t.wg.Add(2)
	go t.logStream(stdOut)
	go t.logStream(stdErr)
}

func execSystemCmdWait(cmdStr string, stdPipe func(pid int, out, err io.ReadCloser)) error {
	cmd := exec.Command("/bin/bash", "-c", cmdStr)
	errPipe, err := cmd.StderrPipe()
	if err != nil {
		return errors.Trace(err)
	}
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		return errors.Trace(err)
	}

	if err := cmd.Start(); err != nil {
		return errors.Trace(err)
	}

	stdPipe(cmd.Process.Pid, outPipe, errPipe)

	return errors.Trace(cmd.Wait())
}

func (t *task) install() error {
	var err error

	t.updateState(stateInstallBegin)

	defer func() {
		if err != nil {
			t.updateState(stateInstallFailed)
			return
		}
		t.updateState(stateInstallSuccess)
	}()

	//切换工作目录.
	oldPath, _ := os.Getwd()
	if err = os.Chdir(t.path); err != nil {
		return errors.Annotatef(err, t.path)
	}
	defer os.Chdir(oldPath)

	tarFile := t.service.Name + ".tar.gz"

	cmd := fmt.Sprintf("tar -C bin -czf %s %s", tarFile, t.service.Name)
	if err := execSystemCmdWait(cmd, t.writeLogs); err != nil {
		return errors.Annotatef(err, cmd)
	}

	key := t.service.Source[7:]

	for _, n := range t.service.Cluster.Node {
		app := w.get(key, n.Server)
		cmd = fmt.Sprintf("./install.sh %s %d %s", t.service.Name, app.PID, config.Distributor.ETCD.Hosts)
		sc, err := ssh.NewClient(n.Server, 22, "jeduser", "", config.Distributor.SSH.Key)
		if err != nil {
			return errors.Trace(err)
		}

		log.Debugf("%v begin, upload install script", t)
		if err = sc.Upload("./install.sh", "install.sh"); err != nil {
			return errors.Annotatef(err, "install.sh")
		}
		log.Debugf("%v end, upload install script", t)

		log.Debugf("%v begin, upload file:%v", t, tarFile)
		if err = sc.Upload(tarFile, tarFile); err != nil {
			return errors.Trace(err)
		}
		log.Debugf("%v end, upload file:%v", t, tarFile)

		log.Debugf("%v begin, ssh exec:%v", t, cmd)
		if err = sc.ExecPipe(cmd, t.writeSSHLogs); err != nil {
			return errors.Annotatef(err, cmd)
		}
		log.Debugf("%v end, ssh exec:%v", t, cmd)
		log.Debugf("%v deploy %s success", t, n.Server)
	}
	t.wg.Wait()

	log.Debugf("%v deploy all success", t)

	return nil
}

//compile 使用脚本编译指定应用.
func (t *task) compile() error {
	var err error

	t.updateState(stateCompileBegin)

	defer func() {
		if err != nil {
			t.updateState(stateComplieFailed)
			return
		}
		t.updateState(stateComplieSuccess)
	}()

	oldPath, _ := os.Getwd()
	if err = os.Chdir(t.path); err != nil {
		return errors.Annotatef(err, t.path)
	}
	defer os.Chdir(oldPath)

	cmd := fmt.Sprintf("./build.sh %s %s %s", t.service.Source, t.service.key(), t.service.Name)

	if err = execSystemCmdWait(cmd, t.writeLogs); err != nil {
		return errors.Annotatef(err, cmd)
	}
	t.wg.Wait()

	log.Debugf("path:%v", oldPath)

	return nil
}
