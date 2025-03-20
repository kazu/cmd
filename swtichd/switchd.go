package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/nasa9084/go-switchbot/v3"
)

const (
	URL    = "https://api.switch-bot.com/v1.1/devices/D3353334214C/commands"
	TOKEN  = "e36cf51b86640dc97ab6e6fe74e20e6b4d3fd6862b9bd9d6f3b6d477cb5855cac6ec06e3477080dc3698417471800370"
	Secret = "2b1fe508b6a9490059e3016140ffcca5"
	devid  = "D3353334214C"
)

func openGate(w http.ResponseWriter, r *http.Request) {
	//fmt.Fprint(w, "hoge")

	c := switchbot.New(TOKEN, Secret)

	e := c.Device().Command(context.Background(), devid, switchbot.PressCommand())

	estr := "OK"
	if e != nil {
		estr = e.Error()
	}

	fmt.Fprintf(w, "press err %s\n", estr)

}

func fuga(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "fuga")
}

func main() {
	server := http.Server{
		Addr:    ":9999",
		Handler: nil,
	}

	http.HandleFunc("/on", openGate)
	http.HandleFunc("/fuga", fuga)

	server.ListenAndServe()
}
