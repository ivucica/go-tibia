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

	_ "golang.org/x/net/trace"

	"badc0de.net/pkg/flagutil"

	tdat "badc0de.net/pkg/go-tibia/dat"
	"badc0de.net/pkg/go-tibia/gameworld"
	"badc0de.net/pkg/go-tibia/login"
	tnet "badc0de.net/pkg/go-tibia/net"
	"badc0de.net/pkg/go-tibia/otb/items"
	"badc0de.net/pkg/go-tibia/otb/map"
	"badc0de.net/pkg/go-tibia/paths"
	"badc0de.net/pkg/go-tibia/secrets"
	"badc0de.net/pkg/go-tibia/things"

	"badc0de.net/pkg/go-tibia/spr"
	"image/png"
	"strconv"
)

var (
	quitChan = make(chan int)

	itemsOTBPath string
	itemsXMLPath string
	tibiaDatPath string
	tibiaSprPath string
	mapPath      string

	debugWebServer = flag.String("debug_web_server_listen_address", "", "where the debug server will listen")
)

func setupFilePathFlags() {
	paths.SetupFilePathFlag("items.otb", "items_otb_path", &itemsOTBPath)
	paths.SetupFilePathFlag("items.xml", "items_xml_path", &itemsXMLPath)
	paths.SetupFilePathFlag("Tibia.dat", "tibia_dat_path", &tibiaDatPath)
	paths.SetupFilePathFlag("Tibia.spr", "tibia_spr_path", &tibiaSprPath)
	paths.SetupFilePathFlag("map.otbm", "map_path", &mapPath)
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
	t, err := things.New()
	if err != nil {
		glog.Errorln("creating thing registry", err)
		return
	}

	f, err := os.Open(itemsOTBPath)
	if err != nil {
		glog.Errorln("opening items otb file for add", err)
		return
	}
	itemsOTB, err := itemsotb.New(f)
	f.Close()

	f, err = os.Open(itemsXMLPath)
	if err != nil {
		glog.Errorln("opening items xml file for add", err)
		return
	}
	itemsOTB.AddXMLInfo(f)
	f.Close()

	if err != nil {
		glog.Errorln("parsing items otb for add", err)
		return
	}
	t.AddItemsOTB(itemsOTB)

	f, err = os.Open(tibiaDatPath)
	if err != nil {
		glog.Errorln("opening tibia dat file for add", err)
		return
	}
	dataset, err := tdat.NewDataset(f)
	f.Close()
	if err != nil {
		glog.Errorln("parsing tibia dat for add", err)
		return
	}
	t.AddTibiaDataset(dataset)

	hasSpr := false
	f, err = os.Open(tibiaSprPath)
	if err == nil { // sprites are optional
		s, err := spr.DecodeAll(f)
		f.Close()
		if err != nil {
			glog.Errorln("parsing tibia spr for add", err)
			return
		}
		t.AddSpriteSet(s)
		hasSpr = true
	}

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
	gw.SetMapDataSource(m)

	if hasSpr {
		http.HandleFunc("/debug/map", func(w http.ResponseWriter, r *http.Request) {
			var tx, ty uint16
			var tbot, ttop uint8
			var tw, th int

			tx = 84
			ty = 84
			tbot = 7
			ttop = 0
			tw = 18
			th = 14

			if x := r.URL.Query().Get("x"); x != "" {
				txI, _ := strconv.Atoi(x)
				tx = uint16(txI)
			}
			if y := r.URL.Query().Get("y"); y != "" {
				tyI, _ := strconv.Atoi(y)
				ty = uint16(tyI)
			}
			if w := r.URL.Query().Get("w"); w != "" {
				tw, _ = strconv.Atoi(w)
			}
			if h := r.URL.Query().Get("h"); h != "" {
				th, _ = strconv.Atoi(h)
			}
			if bot := r.URL.Query().Get("bot"); bot != "" {
				tbotI, _ := strconv.Atoi(bot)
				tbot = uint8(tbotI)
			}
			if top := r.URL.Query().Get("top"); top != "" {
				ttopI, _ := strconv.Atoi(top)
				ttop = uint8(ttopI)
			}

			if tw > 70 {
				tw = 70
			}
			if th > 70 {
				th = 70
			}

			// TODO: more input validation! never allow for number inside CompositeMap to go negative, e.g.
			img := gameworld.CompositeMap(m, t, tx, ty, ttop, tbot, tw, th, 32, 32)
			w.Header().Set("Content-Type", "image/png")
			w.WriteHeader(http.StatusOK)
			png.Encode(w, img)
		})
	}

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
