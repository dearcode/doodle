package service

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/coreos/etcd/clientv3"
	"github.com/dearcode/crab/log"
	"github.com/juju/errors"

	"github.com/dearcode/doodle/meta"
	"github.com/dearcode/doodle/service/debug"
	"github.com/dearcode/doodle/util"
	"github.com/dearcode/doodle/util/etcd"
)

const (
	apigatePrefix = "/api/"
)

type keepalive struct {
	etcd  *etcd.Client
	lease clientv3.Lease
}

//apiKey 为当前项目名及IP端口
func apiKey(local string, port int) string {
	return fmt.Sprintf("%s%s/%s/%d", apigatePrefix, debug.Project, local, port)
}

func bindInfo(bind string) (string, int) {
	// 获取本机服务地址
	local := util.LocalAddr()
	port := bind[strings.LastIndex(bind, ":")+1:]
	p, _ := strconv.Atoi(port)

	return local, p
}

// newKeepalive 服务上线，注册到接口平台的etcd.
func newKeepalive(etcdAddr, bind string) (*keepalive, error) {
	if etcdAddr == "" {
		return nil, nil
	}
	c, err := etcd.New(etcdAddr)
	if err != nil {
		return nil, errors.Annotatef(err, etcdAddr)
	}

	local, port := bindInfo(bind)

	key := apiKey(local, port)
	val := meta.NewMicroAPP(local, port, debug.ServiceKey, os.Getpid(), debug.GitHash, debug.GitTime, debug.GitMessage).String()

	lease, err := c.Keepalive(key, val)
	if err != nil {
		log.Errorf("etcd Keepalive key:%v, val:%v, error:%v", key, val, errors.ErrorStack(err))
		c.Close()
		return nil, errors.Trace(err)
	}

	log.Debugf("etcd put key:%v val:%v", key, val)

	return &keepalive{etcd: c, lease: lease}, nil
}

func (k *keepalive) stop() {
	if k == nil {
		return
	}

	k.lease.Close()
	k.etcd.Close()
}
