package protls

import (
	"crypto/tls"
	"net"

	"gitee.com/dark.H/ProxyZ/connections/base"
)

func ConnectTls(config *base.ProtocolConfig) (con net.Conn, err error) {
	dst := config.RemoteAddr()
	tlsconfig, _ := config.GetTlsConfig()
	return tls.Dial("tcp", dst, tlsconfig)
}
