package main

import (
	"flag"
	"io"
	"io/ioutil"
	"net"
	"time"

	"github.com/golang/glog"

	"badc0de.net/pkg/go-tibia/login"
	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/secrets"
)

func main() {
	flag.Parse()
	go logins()
	go games()
	for {
	}
}
func logins() {
	login, err := login.NewServer(&secrets.OpenTibiaPrivateKey)
	l, err := net.Listen("tcp", ":7171")
	if err != nil {
		glog.Errorln(err)
		return
	}
	glog.Infoln("gotserv loginserver now listening")
	for {
		conn, err := l.Accept()
		if err != nil {
			glog.Errorln(err)
			continue
		}
		go connection(login, conn.(*net.TCPConn))
	}
}
func connection(lgn *login.LoginServer, conn *net.TCPConn) {
	glog.Infoln("accepted connection from ", conn.RemoteAddr())
	defer conn.Close()

	conn.SetDeadline(time.Now().Add(5 * time.Second)) // TODO(ivucica): later, make this longer

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
		glog.Errorln(lgn.Serve(conn, initialMsg))
	case 0x0A:
		glog.Errorln("gameworld protocol currently unsupported")
		return
	default:
		// TODO(ivucica): send error back "wrong protocol"
		// TODO(ivucica): multiplexing on protocol should be done before this
		glog.Errorf("unknown protocol: %d", b[0])
	}

}
func games() {
	login, err := login.NewServer(&secrets.OpenTibiaPrivateKey)
	l, err := net.Listen("tcp", ":7172")
	if err != nil {
		glog.Errorln(err)
		return
	}
	glog.Infoln("gotserv gameserver now listening")
	for {
		conn, err := l.Accept()
		if err != nil {
			glog.Errorln(err)
			continue
		}
		go func() {
			msg := tnet.NewMessage()
			msg.WriteByte(0x1F)

			// timestamp
			msg.WriteByte(0x00)
			msg.WriteByte(0x00)
			msg.WriteByte(0x00)
			msg.WriteByte(0x00)

			// random byte
			msg.WriteByte(0x00)

			// we are supposed to receive the same in the initial packet
			// i.e. we should memorize the above and check later, for this connection...

			// the initial message is unencrypted. prepend size only.
			msg, err := msg.PrependSize()

			wr, err := io.Copy(conn, msg)
			if err != nil {
				glog.Errorf("error writing login message response: %s", err)
				return
			}
			glog.V(2).Infof("written %d bytes", wr)
			connection(login, conn.(*net.TCPConn))
		}()
	}
}
