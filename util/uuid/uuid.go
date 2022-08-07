package uuid

import (
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sync/atomic"
)

var (
	inc uint32
	buf = make([]byte, 8)
)

func init() {
	addrs, _ := net.InterfaceAddrs()
	for _, addr := range addrs {
		if i, ok := addr.(*net.IPNet); ok {
			if !i.IP.IsLoopback() {
				i4 := i.IP.To4()
				if len(i4) == net.IPv4len {
					buf[0] = i.IP.To4()[2]
					buf[1] = i.IP.To4()[3]
					break
				}
			}
		}
	}
	binary.LittleEndian.PutUint16(buf[2:], uint16(os.Getpid()))
}

// UINT64 生成uint64的uuid
func UINT64() uint64 {
	binary.BigEndian.PutUint32(buf[4:], atomic.AddUint32(&inc, 1))
	return binary.BigEndian.Uint64(buf)
}

// String 生成字符串的uuid
func String() string {
	binary.BigEndian.PutUint32(buf[4:], atomic.AddUint32(&inc, 1))
	return fmt.Sprintf("%x", binary.BigEndian.Uint64(buf))
}
