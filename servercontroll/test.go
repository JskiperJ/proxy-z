package servercontroll

import (
	"time"

	"gitee.com/dark.H/gs"
)

func TestServer(server string) (t time.Duration, IDS gs.List[string]) {
	st := time.Now()
	ok := true
	f := ""
	if !gs.Str(server).In(":55443") {
		server += ":55443"
	}
	if gs.Str(server).In("://") {
		f = "https://" + gs.Str(server).Split("://")[1].Str()
	} else {
		f = "https://" + gs.Str(server).Str()
	}

	HTTPSGet(f + "/proxy-info").Json().Every(func(k string, v any) {
		if k == "status" {
			// gs.S(v).Color("g").Println(server)
			if v != "ok" {
				gs.Str("server is not alive !").Color("r").Println()
				ok = false
			}
		} else if k == "ids" {
			idsS := v.([]any)
			for _, i := range idsS {
				IDS = IDS.Add(i.(string))
			}
		}
	})
	if !ok {
		return time.Duration(30000) * time.Hour, IDS
	}
	return time.Since(st), IDS
}

func SendUpdate(server string) {
	f := ""
	if !gs.Str(server).In(":55443") {
		server += ":55443"
	}
	if gs.Str(server).In("://") {
		f = "https://" + gs.Str(server).Split("://")[1].Str()
	} else {
		f = "https://" + gs.Str(server).Str()
	}
	HTTPSPost(f+"/z11-update", nil).Json().Every(func(k string, v any) {
		gs.S(v).Color("g").Println(server + " > " + k)
	})
}
