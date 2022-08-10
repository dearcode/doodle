package etcd

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"go.etcd.io/etcd/client/v3"
)

var (
	testKey       = "/goapi/dbfree/exec/192.168.0.222:8080"
	testCluster   = []string{"192.168.180.104:12379", "192.168.180.104:22379", "192.168.180.104:32379"}
	testKeyPrefix = "/goapi/"
)

func TestAddKey(t *testing.T) {
	c, err := New(testCluster...)
	if err != nil {
		t.Fatal(err.Error())
	}

	_, err = c.Keepalive(testKey, "ok")
	if err != nil {
		t.Fatal(err.Error())
	}

}

func TestWatchKey(t *testing.T) {
	c, err := New(testCluster...)
	if err != nil {
		t.Fatal(err.Error())
	}

	l, err := c.Keepalive(testKey, "ok")
	if err != nil {
		t.Fatal(err.Error())
	}

	go func() {
		time.Sleep(time.Second)
		l.Close()
		fmt.Printf("close\n")

	}()

	ec := make(chan clientv3.Event)

	c.WatchPrefix(testKeyPrefix, ec)
	e := <-ec
	t.Logf("event:%v, key:%s", e.Type, e.Kv.Key)

}

func TestGetKey(t *testing.T) {
	c, err := New(testCluster...)
	if err != nil {
		t.Fatal(err.Error())
	}

	v := fmt.Sprintf("%v", time.Now().UnixNano())

	if err = c.Put(testKey, v); err != nil {
		t.Fatalf(err.Error())
	}

	v2, err := c.Get(testKey)
	if err != nil {
		t.Fatal(err.Error())
	}

	if v != v2 {
		t.Fatalf("key:%v, expect:%v, recv:%v", testKey, v, v2)
	}

}

func TestAddrs(t *testing.T) {
	data := []struct {
		input  []string
		output []string
	}{
		{[]string{"1.1.1.1,2.2.2.2", "3.3.3.3", "4.4.4.4"}, []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}},
		{[]string{"1.1.1.1,2.2.2.2,3.3.3.3, 4.4.4.4"}, []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}},
		{[]string{"1.1.1.1 , 2.2.2.2,3.3.3.3, 4.4.4.4"}, []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}},
		{[]string{"1.1.1.1,2.2.2.2 ", " 3.3.3.3, 4.4.4.4"}, []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4"}},
		{[]string{"1.1.1.1"}, []string{"1.1.1.1"}},
		{[]string{" 1.1.1.1 "}, []string{"1.1.1.1"}},
	}

	for _, d := range data {
		addrs := etcdAddrs(d.input...)
		t.Logf("addrs:%#v", addrs)
		if !reflect.DeepEqual(addrs, d.output) {
			t.Fatalf("input:%+v, expect:%+v, recv:%+v", d.input, d.output, addrs)
		}
	}
}
