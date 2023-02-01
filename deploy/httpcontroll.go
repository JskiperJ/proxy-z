package deploy

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"runtime"
	"text/template"
	"time"

	"gitee.com/dark.H/ProxyZ/asset"
	"gitee.com/dark.H/ProxyZ/clientcontroll"
	"gitee.com/dark.H/gs"
)

type ClientInterface interface {
	TryClose()
	ChangeRoute(string)
	Socks5Listen() error
	ChangePort(int)
	GetRoute() string
}

type HTTPAPIConfig struct {
	ClientConf ClientInterface
	Routes     gs.List[*Onevps]
	Logined    bool
}

var (
	globalClient = &HTTPAPIConfig{}
)

func LoadPage(name string, data any) []byte {
	buf, _ := asset.Asset("Resources/pages/" + name)
	text := string(buf)
	buffer := bytes.NewBuffer([]byte{})
	t, _ := template.New(name).Parse(text)
	t.Execute(buffer, data)
	return buffer.Bytes()
}

func localSetupHandler() http.Handler {
	mux := http.NewServeMux()

	go func() {
		inter := time.NewTicker(1 * time.Minute)
		for {
			select {
			case <-inter.C:
				if globalClient.Routes.Count() > 0 {
					TestRoutes(globalClient.Routes)
				}

			default:
				time.Sleep(12 * time.Second)
			}
		}
	}()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if globalClient.Routes.Count() == 0 {
			http.Redirect(w, r, "/z-login", http.StatusSeeOther)
		}
		if r.Method == "GET" {
			w.Write(LoadPage("route.html", globalClient.Routes))
		}
	})

	mux.HandleFunc("/z-login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {

			w.Write(LoadPage("login.html", nil))
		} else {
			r.ParseForm()
			user := r.FormValue("name")
			pwd := r.FormValue("pwd")
			// gs.Str(name + ":" + pwd).Println()
			if vpss := GitGetAccount("https://"+string(gs.Str("55594657571e515d5f1f5653405b1c7a1d53541c555946").Derypt("2022")), user, pwd); vpss.Count() > 0 {
				globalClient.Routes = vpss
				http.Redirect(w, r, "/", http.StatusSeeOther)
			}
		}
	})
	mux.HandleFunc("/z-api", func(w http.ResponseWriter, r *http.Request) {
		if globalClient.Routes.Count() == 0 {
			http.Redirect(w, r, "/z-login", http.StatusSeeOther)
		}
		if globalClient.ClientConf == nil {
			http.Redirect(w, r, "/z-login", http.StatusSeeOther)
		}
		d, err := Recv(r.Body)
		if err != nil {
			w.WriteHeader(400)
			Reply(w, err, false)
		}
		if d == nil {
			Reply(w, "alive", true)
			return
		}
		if op, ok := d["op"]; ok {
			switch op {
			case "connect":
				if user, ok := d["user"]; ok && user != nil {
					if pwd, ok := d["pwd"]; ok && pwd != nil {
						go func() {
							if vpss := GitGetAccount("https://"+string(gs.Str("55594657571e515d5f1f5653405b1c7a1d53541c555946").Derypt("2022")), user.(string), pwd.(string)); vpss.Count() > 0 {
								globalClient.Routes = vpss
							}
						}()
						Reply(w, "ok", true)
						return
					}
				}
			case "switch":
				if host, ok := d["host"]; ok && host != nil {
					if globalClient.ClientConf == nil {
						globalClient.ClientConf = clientcontroll.NewClientControll(host.(string), 3080)
						go globalClient.ClientConf.Socks5Listen()
					} else {
						globalClient.ClientConf.TryClose()
						go globalClient.ClientConf.ChangeRoute(host.(string))
					}
					Reply(w, "ok", true)
				} else {
					Reply(w, "no host", false)
				}
			case "check":
				Reply(w, globalClient.ClientConf.GetRoute(), true)
			case "test":
				Reply(w, globalClient.Routes, true)

			}
		}
		Reply(w, "err", false)

	})
	return mux

}

func LocalAPI() {
	server := &http.Server{
		Handler: localSetupHandler(),
		Addr:    "0.0.0.0:55555",
	}
	go func() {
		time.Sleep(2 * time.Second)
		if runtime.GOOS == "windows" {
			gs.Str("start http://localhost:55555/").Exec()
		} else if runtime.GOOS == "darwin" {
			gs.Str("open http://localhost:55555/").Exec()
		}
	}()
	server.ListenAndServe()
}

func Reply(w io.Writer, msg any, status bool) {
	if status {
		fmt.Fprintf(w, string(gs.Dict[any]{
			"status": "ok",
			"msg":    msg,
		}.Json()))
	} else {
		fmt.Fprintf(w, string(gs.Dict[any]{
			"status": "fail",
			"msg":    msg,
		}.Json()))

	}
}

func Recv(r io.Reader) (d gs.Dict[any], err error) {
	buf, err := ioutil.ReadAll(r)
	if err != io.EOF && err != nil {
		// w.WriteHeader(400)
		return nil, err
	}
	if len(buf) == 0 {
		return nil, nil
	}
	if d := gs.S(buf).Json(); len(d) > 0 {
		return d, nil
	}
	return nil, nil
}
