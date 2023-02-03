package proquic

import (
	"errors"
	"net"

	"gitee.com/dark.H/ProxyZ/connections/base"
	"github.com/quic-go/quic-go"
)

type QuicClient struct {
	addr     string
	isclosed bool
	qcon     quic.Connection
}

func NewQuicClient(config *base.ProtocolConfig) (qc *QuicClient) {
	qc = new(QuicClient)
	qc.addr = config.RemoteAddr()
	tlsconfig, _ := config.GetQuicConfig()
	conn, err := quic.DialAddr(qc.addr, tlsconfig, nil)
	if err != nil {
		qc.isclosed = true
		return qc
	}
	qc.qcon = conn
	return
}

func (qc *QuicClient) IsClosed() bool {
	return qc.isclosed
}

func (q *QuicClient) NewConnnect() (con net.Conn, err error) {
	if q.IsClosed() || q.qcon == nil {
		return nil, errors.New("dia quic err")
	}
	conn := q.qcon
	stream, err := conn.OpenStream()
	if err != nil {
		return nil, err
	}
	qq := WrapQuicNetConn(stream)
	return qq, err
}

func (q *QuicClient) Close() error {
	q.isclosed = true
	if q.qcon != nil {
		return q.qcon.CloseWithError(quic.ApplicationErrorCode(0), "closd")
	} else {
		return errors.New("no qcon")
	}

}
