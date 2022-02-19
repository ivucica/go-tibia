package main

import (
	"fmt"
	"os"

	"github.com/SherClockHolmes/webpush-go"
)

func main() {
	priv, pub, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		panic(err)
	}

	vapidIncSh, err := os.Open(os.Getenv("GOPATH") + "/src/badc0de.net/pkg/go-tibia/vapid.inc.sh")
	if err == nil {
		vapidIncSh.Close()
		panic("aborting: vapid.inc.sh already exists")
	}

	vapidIncSh, err = os.Create(os.Getenv("GOPATH") + "/src/badc0de.net/pkg/go-tibia/vapid.inc.sh")
	if err != nil {
		panic("aborting: vapid.inc.sh could not be created: " + err.Error())
	}
	fmt.Fprintf(vapidIncSh, `GOTIBIA_VAPID_PRIVATE=%s
GOTIBIA_VAPID_PUBLIC=%s
`, priv, pub)
	vapidIncSh.Close()
	fmt.Println("done")
}
