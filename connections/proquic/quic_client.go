package proquic

import (
	"net"
	"sync"

	"gitee.com/dark.H/ProxyZ/connections/base"
	"gitee.com/dark.H/gs"
	"github.com/quic-go/quic-go"
)

var (
	quicTunnels = gs.List[quic.Connection]{}
	lastUsedNum = 0
	maxNum      = 20
	lock        = sync.RWMutex{}
)

func getNum() int {
	defer lock.Unlock()
	lock.Lock()
	i := lastUsedNum
	lastUsedNum = (lastUsedNum + 1) % maxNum
	return i
}

func ConnectQUIC(addr string, config *base.ProtocolConfig) (con net.Conn, err error) {
	no := getNum()
	var conn quic.Connection
	if quicTunnels.Count() <= no {
		tlsconfig, _ := config.GetQuicConfig()
		conn, err = quic.DialAddr(addr, tlsconfig, nil)
		if err != nil {
			return nil, err
		}
		quicTunnels = quicTunnels.Add(conn)
	} else {
		conn = quicTunnels.Nth(no)
	}
	stream, err := conn.OpenStream()
	if err != nil {
		return nil, err
	}
	qq := WrapQuicNetConn(stream)
	return qq, err
}
