package servercontroll

import (
	"net/http"

	"gitee.com/dark.H/ProxyZ/connections/baseconnection"
	"gitee.com/dark.H/ProxyZ/update"
	"gitee.com/dark.H/gs"
)

func setupHandler(www string) http.Handler {
	mux := http.NewServeMux()
	if len(www) > 0 {
		mux.HandleFunc("/z-files", func(w http.ResponseWriter, r *http.Request) {
			fs := gs.List[any]{}
			gs.Str(www).Ls().Every(func(no int, i gs.Str) {
				isDir := i.IsDir()
				name := i.Basename()
				size := i.FileSize()
				fs = fs.Add(gs.Dict[any]{
					"name":  name,
					"isDir": isDir,
					"size":  size,
				})
			})
			Reply(w, fs, true)

		})
		mux.Handle("/z-files-d/", http.StripPrefix("/z-files-d/", http.FileServer(http.Dir(www))))
		mux.HandleFunc("/z-files-u", uploadFileFunc(www))
	}
	mux.HandleFunc("/z-info", func(w http.ResponseWriter, r *http.Request) {
		d, err := Recv(r.Body)
		if err != nil {
			w.WriteHeader(400)
			Reply(w, err, false)
		}
		if d == nil {
			Reply(w, "alive", true)
		}
	})
	mux.HandleFunc("/proxy-info", func(w http.ResponseWriter, r *http.Request) {
		ids := []string{}
		Tunnels.Every(func(no int, i *baseconnection.ProxyTunnel) {
			ids = append(ids, i.GetConfig().ID)
		})
		Reply(w, gs.Dict[[]string]{
			"ids": ids}, true)
	})

	mux.HandleFunc("/z-dns", func(w http.ResponseWriter, r *http.Request) {

	})

	mux.HandleFunc("/proxy-get", func(w http.ResponseWriter, r *http.Request) {
		tu := GetProxy()
		if !tu.On {
			afterID := tu.GetConfig().ID
			err := tu.Start(func() {
				DelProxy(afterID)
			})
			if err != nil {
				Reply(w, err, false)
				return
			}
		}
		str := tu.GetConfig()
		Reply(w, str, true)
	})

	mux.HandleFunc("/z11-update", func(w http.ResponseWriter, r *http.Request) {
		ids := gs.List[string]{}
		Tunnels.Every(func(no int, i *baseconnection.ProxyTunnel) {
			ids = append(ids, i.GetConfig().ID)
		})

		ids.Every(func(no int, i string) {
			DelProxy(i)
		})

		update.Update(func(info string, ok bool) {
			Reply(w, info, ok)
		})
		// }
	})

	mux.HandleFunc("/proxy-err", func(w http.ResponseWriter, r *http.Request) {

		d, err := Recv(r.Body)

		if err != nil {
			w.WriteHeader(400)
			Reply(w, err, false)
		}
		if id, ok := d["ID"]; ok && id != nil {
			idstr := id.(string)
			gs.Str(idstr).Color("r").Println("proxy-err")
			DelProxy(idstr)
		}
		tu := NewProxyByErrCount()
		afterID := tu.GetConfig().ID
		err = tu.Start(func() {
			DelProxy(afterID)
		})
		if err != nil {
			Reply(w, err, false)
			return
		}
		c := tu.GetConfig()
		Reply(w, c, true)

	})

	mux.HandleFunc("/proxy-new", func(w http.ResponseWriter, r *http.Request) {
		tu := NewProxy("tls")

		str := tu.GetConfig()
		Reply(w, str, true)
	})

	mux.HandleFunc("/proxy-del", func(w http.ResponseWriter, r *http.Request) {
		d, err := Recv(r.Body)
		if err != nil {
			w.WriteHeader(400)
			Reply(w, err, false)
		}
		configName := d["msg"].(string)

		str := DelProxy(configName)
		Reply(w, str, true)
	})
	return mux
}
