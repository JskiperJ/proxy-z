package protls

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"log"
	"net"

	"gitee.com/dark.H/ProxyZ/asset"
	"gitee.com/dark.H/ProxyZ/connections/baseconnection"
	"gitee.com/dark.H/gs"
)

var (
	CERT              = "Resources/pem/cert.pem"
	KEYPEM            = "Resources/pem/key.pem"
	SHARED_TLS_CONFIG *tls.Config
	SHARED_TLS_KEY    = ""
)

// KcpServer used for server
type TlsServer struct {
	config    *baseconnection.ProtocolConfig
	tlsconfig *tls.Config
	// RedirectMode  bool
	// TunnelChan     chan Channel
	// TcpListenPorts map[string]int
	AcceptConn int
	// RedirectBook  *utils.Config
}

func GetTlsConfig() *tls.Config {
	if SHARED_TLS_CONFIG == nil {
		cerPEM, err := asset.Asset(CERT)
		if err != nil {
			log.Fatal(err)
		}
		keyPEM, err := asset.Asset(KEYPEM)
		if err != nil {
			log.Fatal(err)
		}
		SHARED_TLS_KEY = (gs.S(cerPEM) + "|" + gs.S(keyPEM)).Str()

		// Load the certificate and private key
		cert, err := tls.X509KeyPair(cerPEM, keyPEM)
		if err != nil {
			panic(err)
		}
		certpool := x509.NewCertPool()
		certpool.AppendCertsFromPEM(cerPEM)

		SHARED_TLS_CONFIG = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            certpool,
			ClientCAs:          certpool,
			InsecureSkipVerify: true,
		}
	}
	return SHARED_TLS_CONFIG

}

func (tlsserver *TlsServer) Accept() (con net.Conn, err error) {
	listener := tlsserver.GetListener()
	if listener == nil {
		return nil, errors.New("get listener err! in kcp")
	}
	con, err = listener.Accept()
	if err != nil {
		return
	}
	tlsserver.AcceptConn += 1
	return
}

func (kserver *TlsServer) DelCon(con net.Conn) {
	con.Close()
	kserver.AcceptConn -= 1
}

func (tlsserver *TlsServer) GetListener() net.Listener {
	address := gs.Str("%s:%d").F(tlsserver.config.Server, tlsserver.config.ServerPort).Str()
	listenr, err := tls.Listen("tcp", address, tlsserver.tlsconfig)
	if err != nil {
		return nil
	}
	return listenr
}

func (kserver *TlsServer) GetConfig() *baseconnection.ProtocolConfig {
	return kserver.config
}

func NewTlsServer(config *baseconnection.ProtocolConfig) *TlsServer {
	k := new(TlsServer)

	k.tlsconfig = GetTlsConfig()
	config.Password = SHARED_TLS_KEY
	config.Method = "tls"
	k.config = config

	return k
}
