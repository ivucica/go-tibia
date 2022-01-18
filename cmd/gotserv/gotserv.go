package main

import (
	"flag"
	"io"
	"io/ioutil"
	"net"
	"os"
	"time"

	"fmt"
	"net/http"
	"runtime"

	"github.com/golang/glog"
	"github.com/gorilla/mux"

	_ "golang.org/x/net/trace"

	"badc0de.net/pkg/flagutil"

	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/login"
	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/otb/map"
	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/secrets"
	"badc0de.net/pkg/go-tibia/things/full"
	"badc0de.net/pkg/go-tibia/web"
)

var (
	quitChan = make(chan int)

	tibiaPicPath string
	mapPath      string

	debugWebServer = flag.String("debug_web_server_listen_address", "", "where the debug server will listen")
	muxRouter      *mux.Router
)

func setupFilePathFlags() {
	full.SetupFilePathFlags()
	paths.SetupFilePathFlag("map.otbm", "map_path", &mapPath)
	paths.SetupFilePathFlag("Tibia.pic", "tibia_pic_path", &tibiaPicPath)
}

func main() {
	setupFilePathFlags()

	flagutil.Parse()
	glog.Infoln("starting gotserv services")
	go logins()
	go games()

	if *debugWebServer != "" {
		http.HandleFunc("/debug/minimetrics", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "runtime.NumGoroutine(): %d\n", runtime.NumGoroutine())
		})

		muxRouter = mux.NewRouter().PathPrefix("/debug/").Subrouter()
		http.Handle("/debug/", muxRouter)

		go http.ListenAndServe(*debugWebServer, nil)
	}

	for {
		select {
		case <-quitChan:
			return
		}
	}
}
func logins() {
	login, err := login.NewServer(&secrets.OpenTibiaPrivateKey)
	if err != nil {
		glog.Errorln(err)
		return
	}

	gameworld, err := gameworld.NewServer(&secrets.OpenTibiaPrivateKey)
	if err != nil {
		glog.Errorln(err)
		return
	}

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
		go connection(login, gameworld, conn.(*net.TCPConn))
	}
}
func connection(lgn *login.LoginServer, gw *gameworld.GameworldServer, conn *net.TCPConn) {
	glog.Infoln("accepted connection from ", conn.RemoteAddr())
	defer conn.Close()

	// This deadline is extended later after login.
	conn.SetDeadline(time.Now().Add(5 * time.Second))

	initialMsg, err := tnet.ReadMessage(conn)
	if err != nil {
		glog.Errorln(err)
		return
	}

	// Skip checksum.
	checksums := [4]byte{}
	checksumsSlice := checksums[:]
	initialMsg.Read(checksumsSlice)

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
		glog.Errorln(gw.Serve(conn, initialMsg))
		return
	default:
		// TODO(ivucica): send error back "wrong protocol"
		// TODO(ivucica): multiplexing on protocol should be done before this
		glog.Errorf("unknown protocol: %d", b[0])
	}

}

func serveLameDuck(l *net.TCPListener, stop chan bool, lgn *login.LoginServer, gw *gameworld.GameworldServer) {
	// This is a dirty hack to temporarily accept connection even while setting up the
	// actual server, but abort ~approximately when told to stop.
	//
	// It would be much nicer to properly integrate the cancellation of attempts to
	// Accept() as soon as stop happens. But this hack will do for now.

	gw.LameDuckText = "Server still starting up. Try again soon."
	defer func() {
		gw.LameDuckText = ""
	}()

	for {
		select {
		case <-stop:
			l.SetDeadline(time.Time{}) // zero value = no deadline = default
			return
		default:
			// do nothing, just go on even if stop is not received
		}

		l.SetDeadline(time.Now().Add(1 * time.Second))
		conn, err := l.Accept()
		if err != nil {
			glog.Errorln(err)
			continue
		}

		go func() {
			localAddr := conn.LocalAddr()
			if localAddr == nil {
				glog.Errorln("could not get local addr")
				return
			}
			glog.Infof("connection accepted via %v", localAddr)

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
			connection(lgn, gw, conn.(*net.TCPConn))
		}()
	}
}

func games() {
	l, err := net.Listen("tcp", ":7172")
	if err != nil {
		glog.Errorln(err)
		return
	}

	login, err := login.NewServer(&secrets.OpenTibiaPrivateKey)
	if err != nil {
		glog.Errorln(err)
		return
	}
	gw, err := gameworld.NewServer(&secrets.OpenTibiaPrivateKey)
	if err != nil {
		glog.Errorln(err)
		return
	}

	lameDuckStop := make(chan bool)
	go serveLameDuck(l.(*net.TCPListener), lameDuckStop, login, gw)

	///
	t, err := full.FromFilePathFlags()
	if err != nil {
		glog.Errorln("creating thing registry", err)
		return
	}

	webh := web.NewHandler(t, full.PathFlagValue(full.FlagTibiaSprPath), tibiaPicPath)
	webh.RegisterRoutes(muxRouter)
	///

	gw.SetThings(t)

	var m gameworld.MapDataSource
	if mapPath == ":test:" {
		m = gameworld.NewMapDataSource()
	} else {
		f, err := os.Open(mapPath)
		if err != nil {
			glog.Errorln("opening map file", err)
			return
		}
		m, err = otbm.New(f, t)
		if err != nil {
			glog.Errorln("reading map file", err)
			return
		}
	}
	webh.RegisterMapRoute(muxRouter, m)
	gw.SetMapDataSource(m)

	///

	lameDuckStop <- true
	glog.Infoln("gotserv gameserver now listening")
	for {
		conn, err := l.Accept()
		if err != nil {
			glog.Errorln(err)
			continue
		}
		go func() {
			localAddr := conn.LocalAddr()
			if localAddr == nil {
				glog.Errorln("could not get local addr")
				return
			}
			glog.Infof("connection accepted via %v", localAddr)

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
			connection(login, gw, conn.(*net.TCPConn))
		}()
	}
}
