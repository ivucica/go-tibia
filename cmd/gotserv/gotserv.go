package main

import (
	"badc0de.net/pkg/go-tibia/login"
	"flag"
	"net"

	"github.com/golang/glog"
)

func main() {
	flag.Parse()
	p := "14299623962416399520070177382898895550795403345466153217470516082934737582776038882967213386204600674145392845853859217990626450972452084065728686565928113"
	q := "7630979195970404721891201847792002125535401292779123937207447574596692788513647179235335529307251350570728407373705564708871762033017096809910315212884101"
	login, err := login.NewServer(p, q)

	l, err := net.Listen("tcp", ":7171")
	if err != nil {
		glog.Errorln(err)
		return
	}
	glog.Infoln("gotserv now listening")
	for {
		conn, err := l.Accept()
		if err != nil {
			glog.Errorln(err)
			continue
		}
		go func() {
			glog.Errorln(login.Serve(conn))
		}()
	}
}
