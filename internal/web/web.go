package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"time"

	"code.dogecoin.org/gossip/dnet"
	"code.dogecoin.org/gossip/iden"
	"code.dogecoin.org/governor"
)

func New(bind string, port int, newIden chan iden.IdentityMsg) governor.Service {
	mux := http.NewServeMux()
	a := &WebAPI{
		srv: http.Server{
			Addr:    net.JoinHostPort(bind, strconv.Itoa(port)),
			Handler: mux,
		},
		newIden: newIden,
	}

	mux.HandleFunc("/ident", a.signIdent)

	return a
}

type WebAPI struct {
	governor.ServiceCtx
	srv     http.Server
	newIden chan iden.IdentityMsg
}

func (a *WebAPI) Stop() {
	// new goroutine because Shutdown() blocks
	go func() {
		// cannot use ServiceCtx here because it's already cancelled
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		a.srv.Shutdown(ctx) // blocking call
		cancel()
	}()
}

func (a *WebAPI) Run() {
	log.Printf("HTTP server listening on: %v\n", a.srv.Addr)
	if err := a.srv.ListenAndServe(); err != http.ErrServerClosed { // blocking call
		log.Printf("HTTP server: %v\n", err)
	}
}

type NewIdent struct {
	Name    string `json:"name"`
	Country string `json:"country"`
	City    string `json:"city"`
	Lat     int    `json:"lat"`
	Long    int    `json:"long"`
}

func (a *WebAPI) signIdent(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		// request
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("bad request: %v", err), http.StatusBadRequest)
			return
		}
		var to NewIdent
		err = json.Unmarshal(body, &to)
		if err != nil {
			http.Error(w, fmt.Sprintf("error decoding JSON: %s", err.Error()), http.StatusBadRequest)
			return
		}

		var icon [1584]byte

		a.newIden <- iden.IdentityMsg{
			Time:    dnet.DogeNow(),
			Name:    to.Name,
			Bio:     "",
			Lat:     int16(to.Lat),
			Long:    int16(to.Long),
			Country: to.Country,
			City:    to.City,
			Icon:    icon[:],
		}

		bytes, err := json.Marshal("OK")
		if err != nil {
			http.Error(w, fmt.Sprintf("error encoding JSON: %s", err.Error()), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
		w.Header().Set("Allow", "GET, OPTIONS")
		w.Write(bytes)
	} else {
		options(w, r, "GET, OPTIONS")
	}
}

func options(w http.ResponseWriter, r *http.Request, options string) {
	switch r.Method {
	case http.MethodOptions:
		w.Header().Set("Allow", options)
		w.WriteHeader(http.StatusNoContent)

	default:
		w.Header().Set("Allow", options)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}
