package prokcp

import (
	"crypto/sha1"
	"net"

	"gitee.com/dark.H/ProxyZ/connections/base"
	"github.com/xtaci/kcp-go"
	"golang.org/x/crypto/pbkdf2"
)

func ConnectKcp(addr string, config *base.ProtocolConfig) (conn net.Conn, err error) {
	_key := config.Password
	_salt := config.SALT
	key := pbkdf2.Key([]byte(_key), []byte(_salt), 4096, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	DataShard := 10
	ParityShard := 3
	// gs.Str("key:%s | salt: %s | ds:%d | pd: %d | mode:%s ").F(_key, _salt, DataShard, ParityShard, config.Type).Println("kcp config")
	kcpconn, err := kcp.DialWithOptions(addr, block, DataShard, ParityShard)

	if err != nil {
		return nil, err
	}

	return kcpconn, nil
}

func ConnectKcpFirstBuf(dst string, config *base.ProtocolConfig, firstbuf ...[]byte) (con net.Conn, reply []byte, err error) {
	con, err = ConnectKcp(dst, config)

	if firstbuf != nil {

		con.Write(firstbuf[0])
		buf := make([]byte, 8096)
		n, err := con.Read(buf)

		if err != nil {
			return nil, nil, err
		}
		reply = make([]byte, n)
		copy(reply, buf[:n])
		return con, reply, nil
	}

	return
}
