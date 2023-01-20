package prokcp

import (
	"crypto/sha1"
	"errors"
	"net"

	"gitee.com/dark.H/ProxyZ/connections/baseconnection"
	"gitee.com/dark.H/gs"
	"github.com/xtaci/kcp-go"
	"github.com/xtaci/smux"
	"golang.org/x/crypto/pbkdf2"
	// "github.com/cs8425/smux"
)

const (
	idType  = 0 // address type index
	idIP0   = 1 // ip address start index
	idDmLen = 1 // domain address length index
	idDm0   = 2 // domain address start index

	typeIPv4     = 1 // type is ipv4 address
	typeDm       = 3 // type is domain address
	typeIPv6     = 4 // type is ipv6 address
	typeRedirect = 9

	lenIPv4        = net.IPv4len + 2 // ipv4 + 2port
	lenIPv6        = net.IPv6len + 2 // ipv6 + 2port
	lenDmBase      = 2               // 1addrLen + 2port, plus addrLen
	AddrMask  byte = 0xf
	// lenHmacSha1 = 10
)

var (
	debug                 bool
	sanitizeIps           bool
	udp                   bool
	managerAddr           string
	smuxConfig            = smux.DefaultConfig()
	Socks5ConnectedRemote = []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x08, 0x43}
)

type Channel struct {
	stream net.Conn
	host   string
}

func newChannel(stream net.Conn, host string) Channel {
	return Channel{
		stream: stream,
		host:   host,
	}
}

// KcpServer used for server
type KcpServer struct {
	config *baseconnection.ProtocolConfig
	// RedirectMode  bool
	// TunnelChan     chan Channel
	// TcpListenPorts map[string]int
	AcceptConn int
	// RedirectBook  *utils.Config
}

func (ksever *KcpServer) Accept() (con net.Conn, err error) {
	listener := ksever.GetListener()
	if listener == nil {
		return nil, errors.New("get listener err! in kcp")
	}
	con, err = listener.Accept()
	if err != nil {
		return
	}
	// KeepAlive := 10
	// ScavengeTTL := 600
	// AutoExpire := 7
	// SmuxBuf := 4194304 * 2
	// StreamBuf := 2097152 * 2
	ksever.AcceptConn += 1
	return
}

func (kserver *KcpServer) DelCon(con net.Conn) {
	con.Close()
	kserver.AcceptConn -= 1
}

func (ksever *KcpServer) GetListener() net.Listener {
	_key := ksever.config.Password
	_salt := ksever.config.SALT

	key := pbkdf2.Key([]byte(_key), []byte(_salt), 4096, 32, sha1.New)
	block, _ := kcp.NewAESBlockCrypt(key)
	// var listener net.Listener
	serverAddr := gs.Str("%s:%d").F(ksever.config.Server, ksever.config.ServerPort)

	SocketBuf := 4194304 * 2
	DataShard := 10
	ParityShard := 3
	addr := serverAddr.Str()
	gs.Str(addr).Println("listen kcp")
	gs.Str("key:%s | salt: %s | ds:%d | pd: %d").F(_key, _salt, DataShard, ParityShard).Println("kcp config")
	if listener, err := kcp.ListenWithOptions(addr, block, DataShard, ParityShard); err == nil {
		listener.SetReadBuffer(SocketBuf)
		listener.SetWriteBuffer(SocketBuf)
		listener.SetDSCP(0)
		return listener
	} else {
		return nil
	}

}

func (kserver *KcpServer) GetConfig() *baseconnection.ProtocolConfig {
	return kserver.config
}

func NewKcpServer(config *baseconnection.ProtocolConfig) *KcpServer {
	k := new(KcpServer)
	config.ProxyType = "kcp"
	config.Type = "fast"
	k.config = config

	return k
}
