package web

import (
	"context"
	"encoding/hex"
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

func New(bind string, port int, announceChanges chan any) governor.Service {
	mux := http.NewServeMux()
	a := &WebAPI{
		srv: http.Server{
			Addr:    net.JoinHostPort(bind, strconv.Itoa(port)),
			Handler: mux,
		},
		announceChanges: announceChanges,
	}

	mux.HandleFunc("/ident", a.postIdent)

	return a
}

type WebAPI struct {
	governor.ServiceCtx
	srv             http.Server
	announceChanges chan any
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
	Name    string `json:"name"`    // [30] display name
	Bio     string `json:"bio"`     // [120] short biography
	Lat     int    `json:"lat"`     // WGS84 +/- 90 degrees, 60 seconds (accurate to 1850m)
	Long    int    `json:"long"`    // WGS84 +/- 180 degrees, 60 seconds (accurate to 1850m)
	Country string `json:"country"` // [2] ISO 3166-1 alpha-2 code (optional)
	City    string `json:"city"`    // [30] city name (optional)
	Icon    string `json:"icon"`    // hex-encoded 48x48 compressed (1584 bytes)
}

const (
	minLat  = -90 * 60
	maxLat  = 90 * 60
	minLong = -180 * 60
	maxLong = 180 * 60
)

func (a *WebAPI) postIdent(w http.ResponseWriter, r *http.Request) {
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

		// validate profile fields
		if len(to.Name) > 30 {
			http.Error(w, fmt.Sprintf("invalid name: more than 30 characters (got %v)", len(to.Name)), http.StatusBadRequest)
			return
		}
		if len(to.Bio) > 120 {
			http.Error(w, fmt.Sprintf("invalid bio: more than 120 characters (got %v)", len(to.Bio)), http.StatusBadRequest)
			return
		}
		if to.Lat < minLat || to.Lat > maxLat {
			http.Error(w, fmt.Sprintf("invalid latitude: out of range [%v, %v] (got %v)", minLat, maxLat, to.Lat), http.StatusBadRequest)
			return
		}
		if to.Long < minLong || to.Long > maxLong {
			http.Error(w, fmt.Sprintf("invalid longitude: out of range [%v, %v] (got %v)", minLong, maxLong, to.Long), http.StatusBadRequest)
			return
		}
		if len(to.Country) != 2 && len(to.Country) != 0 {
			http.Error(w, fmt.Sprintf("invalid country: expecting ISO 3166-1 alpha-2 code (got %v)", len(to.Country)), http.StatusBadRequest)
			return
		}
		if len(to.City) > 30 {
			http.Error(w, fmt.Sprintf("invalid city: more than 30 characters (got %v)", len(to.City)), http.StatusBadRequest)
			return
		}
		icon, err := hex.DecodeString(to.Icon)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid icon: %v", err.Error()), http.StatusBadRequest)
			return
		}
		if len(icon) != dnet.DogeIconSize {
			http.Error(w, fmt.Sprintf("invalid icon: expecting %v bytes (got %v)", dnet.DogeIconSize, len(icon)), http.StatusBadRequest)
			return
		}

		a.announceChanges <- iden.IdentityMsg{
			Time:    dnet.DogeNow(),
			Name:    to.Name,
			Bio:     to.Bio,
			Lat:     int16(to.Lat),
			Long:    int16(to.Long),
			Country: to.Country,
			City:    to.City,
			Icon:    icon,
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
