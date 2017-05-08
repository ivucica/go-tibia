package main

import (
	"flag"
	"io"
	"io/ioutil"
	"net"

	"github.com/golang/glog"

	"badc0de.net/pkg/go-tibia/login"
	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/secrets"
)

func main() {
	flag.Parse()
	login, err := login.NewServer(&secrets.OpenTibiaPrivateKey)

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
			defer conn.Close()
			initialMsg, err := tnet.ReadMessage(conn)
			if err != nil {
				glog.Errorln(err)
				return
			}

			r := io.LimitReader(initialMsg, 1)
			b, err := ioutil.ReadAll(r)
			if err != nil {
				glog.Errorf("failed reading proto id, dropping conn: %s", err)
				return
			}
			if len(b) != 1 {
				glog.Errorf("no proto id. dropping conn.")
				return
			}

			switch b[0] {
			case 0x01:
				glog.Errorln(login.Serve(conn, initialMsg))
			default:
				// TODO(ivucica): send error back "wrong protocol"
				// TODO(ivucica): multiplexing on protocol should be done before this
				glog.Errorf("unknown protocol: %d", b[0])
			}
		}()
	}
}
