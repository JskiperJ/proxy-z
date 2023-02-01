package main

import (
	"flag"
	"os"
	"time"

	"gitee.com/dark.H/ProxyZ/clientcontroll"
	"gitee.com/dark.H/ProxyZ/deploy"
	"gitee.com/dark.H/ProxyZ/servercontroll"
	"gitee.com/dark.H/gs"
)

func main() {
	server := ""
	dev := false
	update := false
	vultrmode := false
	gitmode := false
	daemon := false
	httpmode := false
	noopenbrowser := false
	// cli := false
	// configbuild := false
	l := 1080
	flag.StringVar(&server, "H", "https://localhost:55443", "set server addr/set ssh name / set some other ")
	flag.IntVar(&l, "l", 3080, "set local socks5 listen port")

	flag.BoolVar(&dev, "dev", false, "use ssh to devploy proxy server ; example -H 'user@host:port/pwd' -dev ")
	flag.BoolVar(&update, "update", false, "set this server update by git")
	flag.BoolVar(&vultrmode, "vultr", false, "true to use vultr api to search host")
	flag.BoolVar(&gitmode, "git", false, "true to use git to login group proxy")
	flag.BoolVar(&httpmode, "http", false, "true to use http mode")
	flag.BoolVar(&noopenbrowser, "no-open", false, "true not open browser")
	flag.BoolVar(&daemon, "d", false, "true to run deamon")
	// flag.BoolVar(&cli, "cli", false, "true to use cli-client")
	// flag.BoolVar(&configbuild, "", false, "true to use vultr api to build host group")

	flag.Parse()

	if dev {
		deploy.DepBySSH(server)
		os.Exit(0)
	}
	if vultrmode {
		deploy.VultrMode(server)
		os.Exit(0)
	}
	if update {
		servercontroll.SendUpdate(server)
		os.Exit(0)
	}

	if daemon {
		logFile := gs.TMP.PathJoin("z.log").Str()
		args := []string{}
		for _, a := range os.Args {
			if a == "-d" {
				continue
			}
			args = append(args, a)
		}
		deploy.Daemon(args, logFile)
		time.Sleep(2 * time.Second)
		gs.Str("%s run background | log: %s").F(os.Args[0], logFile).Println("Daemon")
		os.Exit(0)
	}

	if gitmode {
		if !gs.Str(server).StartsWith("https://git") {
			server = "https://" + string(gs.Str("55594657571e515d5f1f5653405b1c7a1d53541c555946").Derypt("2022"))
		}
		server = deploy.GitMode(server)
		if server == "" {
			os.Exit(0)
		}
		clientcontroll.RunLocal(server, l)
	}
	if httpmode {
		deploy.LocalAPI(noopenbrowser)
	}

}
