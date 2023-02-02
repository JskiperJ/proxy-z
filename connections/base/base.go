package base

import (
	"io"
	"math/rand"
	"net"
	"runtime"
	"strings"

	"gitee.com/dark.H/gs"
	"github.com/fatih/color"
)

const bufSize = 4096

func ErrToFile(label string, err error) {
	c := gs.Str("[%s]:" + err.Error() + "\n").F(label)
	c.Color("r").Print()
	c.ToFile("/tmp/z-proxy.err.log")
}

// const bufSize = 8192

// Memory optimized io.Copy function specified for this library
func Copy(dst io.Writer, src io.Reader) (written int64, err error) {
	// If the reader has a WriteTo method, use it to do the copy.
	// Avoids an allocation and a copy.
	if wt, ok := src.(io.WriterTo); ok {
		return wt.WriteTo(dst)
	}
	// Similarly, if the writer has a ReadFrom method, use it to do the copy.
	if rt, ok := dst.(io.ReaderFrom); ok {
		return rt.ReadFrom(src)
	}

	// fallback to standard io.CopyBuffer
	buf := make([]byte, bufSize)
	return io.CopyBuffer(dst, src, buf)
}

func Pipe(p1, p2 net.Conn) (err error) {
	// start tunnel & wait for tunnel termination
	streamCopy := func(dst io.Writer, src io.ReadCloser, fr, to net.Addr) error {
		// startAt := time.Now()
		_, err := Copy(dst, src)

		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
			} else if strings.Contains(err.Error(), "EOF") {
			} else if strings.Contains(err.Error(), "read/write on closed pipe") {
			} else {
				r := color.New(color.FgRed)
				r.Println("error : ", err)
			}

		}
		// endAt := time.Now().Sub(startAt)
		// log.Println("passed:", FGCOLORS[1](n), FGCOLORS[0](p1.RemoteAddr()), "->", FGCOLORS[0](p2.RemoteAddr()), "Used:", endAt)
		p1.Close()
		p2.Close()
		return err
		// }()
	}
	go streamCopy(p1, p2, p2.RemoteAddr(), p1.RemoteAddr())
	err = streamCopy(p2, p1, p1.RemoteAddr(), p2.RemoteAddr())
	return
}

func OpenPortUFW(port int) {
	if runtime.GOOS == "linux" {
		gs.Str("open port :%d").F(port).Color("y").Println()
		gs.Str("ufw allow %d").F(port).Exec()
	}
}

func ClosePortUFW(port int) {
	if runtime.GOOS == "linux" {
		gs.Str("ufw delete allow %d").F(port).Exec()
		gs.Str("close port :%d").F(port).Color("y", "B").Println("Close")

	}
}

func GiveAPort() (port int) {
	for {
		port = 40000 + rand.Int()%10000
		ln, err := net.Listen("tcp", ":"+gs.S(port).Str())
		if err == nil {
			ln.Close()
			OpenPortUFW(port)
			return port
		}
	}

}
