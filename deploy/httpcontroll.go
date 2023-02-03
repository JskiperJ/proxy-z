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
	ChangeProxyType(tp string)
}

type HTTPAPIConfig struct {
	ClientConf ClientInterface
	Routes     gs.List[*Onevps]
	Logined    bool
}

var (
	globalClient = &HTTPAPIConfig{}
	LOCAL_PORT   = 1091
)

func LoadPage(name string, data any) []byte {
	buf, _ := asset.Asset("Resources/pages/" + name)
	text := string(buf)
	buffer := bytes.NewBuffer([]byte{})
	t, _ := template.New(name).Parse(text)
	// gs.S(data).Println()
	t.Execute(buffer, data)
	return buffer.Bytes()
}

func localSetupHandler() http.Handler {
	mux := http.NewServeMux()

	go func() {
		inter := time.NewTicker(10 * time.Minute)
		for {
			select {
			case <-inter.C:
				if globalClient.Routes.Count() > 0 {
					globalClient.Routes = TestRoutes(globalClient.Routes)
				}

			default:
				time.Sleep(12 * time.Second)
			}
		}
	}()

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if globalClient.Routes.Count() == 0 {

			http.Redirect(w, r, "/z-login", http.StatusSeeOther)

			return
		}

		if r.Method == "GET" {
			// globalClient.Routes.Every(func(no int, i *Onevps) {
			// 	i.Println()
			// })
			w.Write(LoadPage("route.html", globalClient.Routes))
			return
		}
	})

	mux.HandleFunc("/z-login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			w.Write(LoadPage("login.html", nil))
		} else {
			// fmt.Println(r.Body)
			d, err := Recv(r.Body)
			if err != nil {
				w.WriteHeader(400)
				Reply(w, err, false)
				return
			}
			user := d["name"]
			pwd := d["password"]
			// gs.Str(user + ":" + pwd).Println()
			if vpss := GitGetAccount("https://"+string(gs.Str("55594657571e515d5f1f5653405b1c7a1d53541c555946").Derypt("2022")), user.(string), pwd.(string)); vpss.Count() > 0 {
				globalClient.Routes = vpss
				gs.Str("start test route").Println("login")
				go TestRoutes(globalClient.Routes)
				Reply(w, "", true)
				return
			} else {
				w.WriteHeader(400)
				Reply(w, "", false)
			}
		}
	})

	mux.HandleFunc("/z-api", func(w http.ResponseWriter, r *http.Request) {
		if globalClient.Routes.Count() == 0 {
			http.Redirect(w, r, "/z-login", http.StatusSeeOther)
			return
		}
		// if globalClient.ClientConf == nil {
		// 	http.Redirect(w, r, "/z-login", http.StatusSeeOther)
		// 	return
		// }
		d, err := Recv(r.Body)
		if err != nil {
			w.WriteHeader(400)
			Reply(w, err, false)
			return
		}
		if d == nil {
			Reply(w, "alive", true)
			return
		}
		// gs.S(d).Println("API")
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
			case "change":
				if proxyTp, ok := d["proxy-type"]; ok {
					go globalClient.ClientConf.ChangeProxyType(proxyTp.(string))
					Reply(w, "change proxy :"+proxyTp.(string), true)
				} else {
					Reply(w, "faled", false)
				}
				return
			case "switch":
				if host, ok := d["host"]; ok && host != nil {
					gs.Str(host.(string)).Color("g", "B").Println("Swtich")
					if globalClient.ClientConf == nil {
						globalClient.ClientConf = clientcontroll.NewClientControll(host.(string), LOCAL_PORT)
						go globalClient.ClientConf.Socks5Listen()
					} else {
						gs.Str("Close Old!").Color("g", "B").Println("Swtich")
						globalClient.ClientConf.TryClose()
						go globalClient.ClientConf.ChangeRoute(host.(string))
					}
					Reply(w, "ok", true)
				} else {
					Reply(w, "no host", false)
				}
				return
			case "check":
				if globalClient.ClientConf != nil {
					Reply(w, globalClient.ClientConf.GetRoute(), true)
				} else {
					Reply(w, "err", false)
				}
				return
			case "test":
				Reply(w, globalClient.Routes, true)
				return

			}
		}
		Reply(w, "err", false)

	})
	return mux

}

func LocalAPI(openbrowser bool) {
	server := &http.Server{
		Handler: localSetupHandler(),
		Addr:    "0.0.0.0:35555",
	}
	if !openbrowser {
		go func() {
			time.Sleep(2 * time.Second)
			if runtime.GOOS == "windows" {
				gs.Str("start http://localhost:35555/").Exec()
			} else if runtime.GOOS == "darwin" {
				gs.Str("open http://localhost:35555/").Exec()
			}
		}()
	}
	err := server.ListenAndServe()
	if err != nil {
		gs.Str(err.Error()).Color("r").Println("Err")
	}
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
	// fmt.Println(gs.S(buf))
	if d := gs.Str(buf).Json(); len(d) > 0 {
		return d, nil
	}
	return nil, nil
}
