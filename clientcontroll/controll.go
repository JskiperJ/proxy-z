package clientcontroll

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"gitee.com/dark.H/ProxyZ/connections/baseconnection"
	"gitee.com/dark.H/ProxyZ/connections/prokcp"
	"gitee.com/dark.H/ProxyZ/connections/prosmux"
	"gitee.com/dark.H/ProxyZ/connections/prosocks5"
	"gitee.com/dark.H/ProxyZ/connections/protls"
	"gitee.com/dark.H/ProxyZ/servercontroll"
	"gitee.com/dark.H/gs"
)

var (
	errInvalidWrite = errors.New("invalid write result")
	ErrRouteISBreak = errors.New("route is break")
)

func RunLocal(server string, l int) {

	if r := servercontroll.TestServer(server); r > time.Minute {
		os.Exit(0)
		return
	} else {
		gs.Str("server build time: %s ").F(r).Println("test")
	}

	cli := NewClientControll(server, l)
	cli.Socks5Listen()
}

type ClientControl struct {
	SmuxClients []*prosmux.SmuxConfig

	nowconf    *baseconnection.ProtocolConfig
	ClientNum  int
	ListenPort int
	ErrCount   int
	AliveCount int
	lastUse    int
	lock       sync.RWMutex
	Addr       gs.Str
	closed     bool
	closeFlag  bool
}

func NewClientControll(addr string, listenport int) *ClientControl {
	if !gs.Str(addr).In(":") {
		addr += ":55443"
	}
	if !gs.Str(addr).In("://") {
		addr = "https://" + addr
	}

	c := &ClientControl{
		Addr:       gs.Str(addr),
		ListenPort: listenport,
		ClientNum:  40,
		lastUse:    -1,
	}
	return c
}

func RecvMsg(reply gs.Str) (di any, o bool) {
	d := reply.Json()
	if c, ok := d["status"]; ok {
		if c.(string) == "ok" {
			o = true
		}

		di = d["msg"]
		return
	} else {
		o = false
		return
	}
}

func (c *ClientControl) TryClose() {
	c.closeFlag = true
}

func (c *ClientControl) ChangeRoute(host string) {

	if c.closeFlag {
		c.Addr = gs.Str(host)
	} else {
		gs.Str("server is not closed !").Color("r").Println()
	}
	for {
		time.Sleep(1 * time.Second)
		if c.closed {
			break
		}
	}
	c.Socks5Listen()
}

func (c *ClientControl) ChangePort(port int) {
	c.ListenPort = port
}

func (c *ClientControl) ReportErrorProxy() (conf *baseconnection.ProtocolConfig) {

	var addr string
	useTls := false
	if c.Addr.StartsWith("tls://") {
		addr = c.Addr.Split("://")[1].Str()
		useTls = true
	} else if c.Addr.In("https://") {
		addr = c.Addr.Split("://")[1].Str()
		useTls = true
	} else if c.Addr.In("://") {
		addr = c.Addr.Split("://")[1].Str()
	} else {
		addr = c.Addr.Str()
	}
	var reply gs.Str
	if useTls {
		reply = servercontroll.HTTPSPost("https://"+addr+"/proxy-err", gs.Dict[any]{
			"ID": c.nowconf.ID,
		})
	} else {
		reply = servercontroll.HTTP3Post("https://"+addr+"/proxy-err", gs.Dict[any]{
			"ID": c.nowconf.ID,
		})
	}

	if reply == "" {
		return nil
	}
	if obj, ok := RecvMsg(reply); ok {
		// fmt.Println(obj)
		buf, err := json.Marshal(obj)
		if err != nil {
			gs.Str(err.Error()).Println("Err Tr")
			return nil
		}
		conf = new(baseconnection.ProtocolConfig)

		if err := json.Unmarshal(buf, conf); err != nil {
			gs.Str("get aviable proxy client err :" + err.Error()).Println("Err")
			return nil
		}
		if conf.Server == "0.0.0.0" {
			conf.Server = gs.Str(addr).Split(":")[0].Trim()
		}
		c.nowconf = conf
	}

	return
}

func (c *ClientControl) GetAviableProxy() (conf *baseconnection.ProtocolConfig) {
	if c.nowconf != nil {
		return c.nowconf
	}
	var addr string
	useTls := false
	if c.Addr.StartsWith("tls://") {
		addr = c.Addr.Split("://")[1].Str()
		useTls = true
	} else if c.Addr.StartsWith("https://") {
		addr = c.Addr.Split("://")[1].Str()
		useTls = true
	} else if c.Addr.In("://") {
		addr = c.Addr.Split("://")[1].Str()
	} else {
		addr = c.Addr.Str()
	}
	var reply gs.Str
	if useTls {
		reply = servercontroll.HTTPSPost("https://"+addr+"/proxy-get", nil)
	} else {
		reply = servercontroll.HTTP3Post("https://"+addr+"/proxy-get", nil)
	}

	if reply == "" {
		return nil
	}
	if obj, ok := RecvMsg(reply); ok {
		// fmt.Println(obj)
		buf, err := json.Marshal(obj)
		if err != nil {
			gs.Str(err.Error()).Println("Err Tr")
			return nil
		}
		conf = new(baseconnection.ProtocolConfig)

		if err := json.Unmarshal(buf, conf); err != nil {
			gs.Str("get aviable proxy client err :" + err.Error()).Println("Err")
			return nil
		}
		if conf.Server == "0.0.0.0" {
			conf.Server = gs.Str(addr).Split(":")[0].Trim()
		}
		c.nowconf = conf
	}

	return
}

/*
**************************************************************
**************************************************************
CORE ！！！！！！！！
*/
func (c *ClientControl) Socks5Listen() (err error) {
	c.InitializationTunnels()
	if c.ListenPort != 0 {
		l, err := net.Listen("tcp", "0.0.0.0:"+gs.S(c.ListenPort).Str())
		if err != nil {
			log.Fatal(err)
		}
		for {
			if c.ErrCount > 7 {
				c.ReportErrorProxy()
				c.ErrCount = 0
			}
			if c.closeFlag {
				break
			}
			socks5con, err := l.Accept()
			if err != nil {
				gs.S(err.Error()).Println("accept err")
				time.Sleep(3 * time.Second)
				continue
			}

			go func(socks5con net.Conn) {
				defer socks5con.Close()
				err := prosocks5.Socks5HandShake(&socks5con)
				if err != nil {
					gs.Str(err.Error()).Println("socks5 handshake")
					return
				}

				raw, host, _, err := prosocks5.GetLocalRequest(&socks5con)
				if err != nil {
					gs.Str(err.Error()).Println("socks5 get host")
					return
				}
				for tryTime := 0; tryTime < 3; tryTime += 1 {
					remotecon, err := c.ConnectRemote()
					if err != nil {
						gs.Str(err.Error()).Println("connect proxy server err")
						return
					}
					defer remotecon.Close()
					_, err = remotecon.Write(raw)
					if err != nil {
						gs.Str(err.Error()).Println("connecting write|" + host)
						c.lock.Lock()
						c.ErrCount += 1
						c.lock.Unlock()
						return
					}
					// gs.Str(host).Color("g").Println("connect|write")
					_buf := make([]byte, len(prosocks5.Socks5Confirm))
					remotecon.SetReadDeadline(time.Now().Add(1 * time.Minute))
					_, err = remotecon.Read(_buf)

					if err != nil {
						gs.Str(err.Error()).Println("connecting read|" + host)
						if err.Error() != "timeout" {
							baseconnection.ErrToFile("err in client controll.go :160", err)
						}

						c.lock.Lock()
						c.ErrCount += 1
						c.lock.Unlock()
						continue
					}
					if bytes.Equal(_buf, prosocks5.Socks5Confirm) {
						_, err = socks5con.Write(_buf)
						if err != nil {
							gs.Str(err.Error()).Println("connecting reply|" + host)
							return
						}
					}

					c.lock.Lock()
					c.AliveCount += 1
					if c.ErrCount > 0 {
						c.ErrCount -= 1
					}
					c.lock.Unlock()
					gs.Str("[%s] %s").F("connecting|"+gs.S(c.AliveCount), host).Color("g").Add("\r").Print()
					remotecon.SetReadDeadline(time.Now().Add(30 * time.Minute))
					c.Pipe(socks5con, remotecon)
					socks5con.Close()
					remotecon.Close()
					c.lock.Lock()
					c.AliveCount -= 1
					c.lock.Unlock()
					break

				}

			}(socks5con)

		}
	}
	c.closed = true
	return
}

func (c *ClientControl) RebuildSmux(no int) (err error) {
	proxyConfig := c.GetAviableProxy()
	if proxyConfig == nil {
		return ErrRouteISBreak
	}
	var singleTunnelConn net.Conn
	switch proxyConfig.ProxyType {
	case "tls":
		singleTunnelConn, err = protls.ConnectTls(proxyConfig.RemoteAddr(), proxyConfig)
	case "kcp":
		singleTunnelConn, err = prokcp.ConnectKcp(proxyConfig.RemoteAddr(), proxyConfig)
	default:
		singleTunnelConn, err = prokcp.ConnectKcp(proxyConfig.RemoteAddr(), proxyConfig)
	}
	// gs.Str("--> "+proxyConfig.RemoteAddr()).Color("y", "B").Println(proxyConfig.ProxyType)
	if singleTunnelConn != nil {
		if len(c.SmuxClients) <= no {
			c.SmuxClients = append(c.SmuxClients, prosmux.NewSmuxClient(singleTunnelConn))
		} else {
			c.lock.Lock()
			c.SmuxClients[no].Session.Close()
			c.SmuxClients[no] = nil
			c.SmuxClients[no] = prosmux.NewSmuxClient(singleTunnelConn)
			c.lock.Unlock()
		}

	} else {
		if err == nil {
			err = errors.New("tls/kcp only :  now method is :" + proxyConfig.ProxyType)
		}
		return err
	}
	return nil
}

func (c *ClientControl) GetSession() (con net.Conn, err error) {
	c.lock.Lock()
	c.lastUse += 1
	c.lastUse = c.lastUse % c.ClientNum
	c.lock.Unlock()
	if c.lastUse >= len(c.SmuxClients) && len(c.SmuxClients) > 0 {
		e := c.SmuxClients[len(c.SmuxClients)-1]
		if e.Session.IsClosed() {
			err = c.RebuildSmux(c.lastUse)
		} else {
			con, err = e.NewConnnect()
		}

	} else {
		if len(c.SmuxClients) == 0 {
			err = c.RebuildSmux(c.lastUse)
			if err != nil {
				return nil, err
			}
			e := c.SmuxClients[c.lastUse]
			con, err = e.NewConnnect()
		} else {
			e := c.SmuxClients[c.lastUse]
			if e.Session.IsClosed() {
				err = c.RebuildSmux(c.lastUse)
			} else {
				con, err = e.NewConnnect()
			}

		}

	}

	return
}

func (c *ClientControl) InitializationTunnels() {
	wait := sync.WaitGroup{}
	l := sync.RWMutex{}
	msgs := gs.Str("*").Color("r").Add("|").Repeat(c.ClientNum).Slice(0, -1).Split("|")
	for i := 0; i < c.ClientNum; i++ {
		wait.Add(1)
		go func(no int, w *sync.WaitGroup) {
			defer wait.Done()
			for {
				err := c.RebuildSmux(no)
				if err != nil {
					gs.Str("rebuild smux err:" + err.Error()).Println("Err")
					// return nil, err
				} else {
					l.Lock()
					msgs[no] = gs.Str('*').Color("g")
					l.Unlock()
					gs.Str("%s >> %s \r").F(c.Addr, msgs.Join("")).Print()
					break
				}

			}

		}(i, &wait)
	}

	wait.Wait()
	time.Sleep(1 * time.Second)
	gs.Str("\n").Print()
}

func (c *ClientControl) ConnectRemote() (con net.Conn, err error) {

	// connted := false

	con, err = c.GetSession()
	if err != nil {
		gs.Str("rebuild smux").Println("connect remote")
		con, err = c.GetSession()
	}
	// gs.Str("smxu connect ").Println()
	return
}

func (c *ClientControl) Pipe(p1, p2 net.Conn) {
	Pipe(p1, p2)
}

func Pipe(p1, p2 net.Conn) {
	var wg sync.WaitGroup
	var wait int = 1800
	wg.Add(1)
	streamCopy := func(dst net.Conn, src net.Conn, fr, to net.Addr) {
		// startAt := time.Now()
		Copy(dst, src, wait)
		p1.Close()
		p2.Close()
		// }()
	}

	go func(p1, p2 net.Conn) {
		wg.Done()
		streamCopy(p1, p2, p2.RemoteAddr(), p1.RemoteAddr())
	}(p1, p2)
	streamCopy(p2, p1, p1.RemoteAddr(), p2.RemoteAddr())
	wg.Wait()
}

// Memory optimized io.Copy function specified for this library
func Copy(dst io.Writer, src io.Reader, timeout ...int) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		if timeout != nil {
			src.(net.Conn).SetReadDeadline(time.Now().Add(time.Duration(timeout[0]) * time.Second))
		}
		return rt.ReadFrom(src)
	}

	// fallback to standard io.CopyBuffer
	buf := make([]byte, 4096)
	return copyBuffer(dst, src, buf, timeout...)
}

func copyBuffer(dst io.Writer, src io.Reader, buf []byte, timeout ...int) (written int64, err error) {
	if buf != nil && len(buf) == 0 {
		panic("empty buffer in CopyBuffer")
	}
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		if timeout != nil {
			src.(net.Conn).SetReadDeadline(time.Now().Add(time.Duration(timeout[0]) * time.Second))
		}
		return rt.ReadFrom(src)
	}
	if buf == nil {
		size := 32 * 1024
		if l, ok := src.(*io.LimitedReader); ok && int64(size) > l.N {
			if l.N < 1 {
				size = 1
			} else {
				size = int(l.N)
			}
		}
		buf = make([]byte, size)
	}
	for {
		if timeout != nil {
			src.(net.Conn).SetReadDeadline(time.Now().Add(time.Duration(timeout[0]) * time.Second))
		}
		nr, er := src.Read(buf)
		if nr > 0 {
			nw, ew := dst.Write(buf[0:nr])
			if nw < 0 || nr < nw {
				nw = 0
				if ew == nil {
					ew = errInvalidWrite
				}
			}
			written += int64(nw)
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err
}
