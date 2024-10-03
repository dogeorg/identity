package web

import (
	"context"
	"encoding/base64"
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
	"code.dogecoin.org/identity/internal/spec"
)

const DogeIconSize = dnet.DogeIconSize + 1 // +1 for style byte (XXX fix in gossip pkg)

func New(bind net.IP, port uint16, webdir string, announceChanges chan any, store spec.Store) governor.Service {
	mux := http.NewServeMux()
	a := &WebAPI{
		srv: http.Server{
			Addr:    net.JoinHostPort(bind.String(), strconv.Itoa(int(port))),
			Handler: mux,
		},
		announceChanges: announceChanges,
		_store:          store,
	}

	mux.HandleFunc("/profile", a.postIdent)

	fs := http.FileServer(http.Dir(webdir))
	mux.Handle("/", fs)

	return a
}

type WebAPI struct {
	governor.ServiceCtx
	srv             http.Server
	announceChanges chan any
	_store          spec.Store
	store           spec.StoreCtx
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
	a.store = a._store.WithCtx(a.Context)
	log.Printf("HTTP server listening on: %v\n", a.srv.Addr)
	if err := a.srv.ListenAndServe(); err != http.ErrServerClosed { // blocking call
		log.Printf("HTTP server: %v\n", err)
	}
}

type NewIdent struct {
	Name    string  `json:"name"`    // [30] display name
	Bio     string  `json:"bio"`     // [120] short biography
	Lat     float64 `json:"lat"`     // WGS84 +/- 90 degrees, 60 seconds (accurate to 1850m)
	Long    float64 `json:"long"`    // WGS84 +/- 180 degrees, 60 seconds (accurate to 1850m)
	Country string  `json:"country"` // [2] ISO 3166-1 alpha-2 code (optional)
	City    string  `json:"city"`    // [30] city name (optional)
	Icon    string  `json:"icon"`    // base64-encoded 48x48 compressed (1584 bytes)
}

const (
	minLat  = -90.0
	maxLat  = 90.0
	minLong = -180.0
	maxLong = 180.0
)

func (a *WebAPI) postIdent(w http.ResponseWriter, r *http.Request) {
	opts := "GET, POST, OPTIONS"
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
		lat := int(to.Lat * 10) // quantize to nearest 0.1 degree
		if to.Long < minLong || to.Long > maxLong {
			http.Error(w, fmt.Sprintf("invalid longitude: out of range [%v, %v] (got %v)", minLong, maxLong, to.Long), http.StatusBadRequest)
			return
		}
		long := int(to.Long * 10) // quantize to nearest 0.1 degree
		if len(to.Country) != 2 && len(to.Country) != 0 {
			http.Error(w, fmt.Sprintf("invalid country: expecting ISO 3166-1 alpha-2 code (got %v)", len(to.Country)), http.StatusBadRequest)
			return
		}
		if len(to.City) > 30 {
			http.Error(w, fmt.Sprintf("invalid city: more than 30 characters (got %v)", len(to.City)), http.StatusBadRequest)
			return
		}
		icon, err := base64.StdEncoding.DecodeString(to.Icon)
		if err != nil {
			http.Error(w, fmt.Sprintf("invalid icon: %v", err.Error()), http.StatusBadRequest)
			return
		}
		if len(icon) != dnet.DogeIconSize && len(icon) != 0 {
			http.Error(w, fmt.Sprintf("invalid icon: expecting %v bytes (got %v)", DogeIconSize, len(icon)), http.StatusBadRequest)
			return
		}

		pro := spec.Profile{
			Name:    to.Name,
			Bio:     to.Bio,
			Lat:     lat,
			Long:    long,
			Country: to.Country,
			City:    to.City,
			Icon:    icon,
		}
		err = a.store.SetProfile(pro)
		if err != nil {
			http.Error(w, fmt.Sprintf("cannot store profile: %v", err), http.StatusInternalServerError)
			return
		}

		a.announceChanges <- iden.IdentityMsg{
			Time:    dnet.DogeNow(),
			Name:    to.Name,
			Bio:     to.Bio,
			Lat:     int16(lat),
			Long:    int16(long),
			Country: to.Country,
			City:    to.City,
			Icon:    icon,
		}

		sendProfile(w, &pro, opts)
	} else if r.Method == http.MethodGet {
		w.Header().Add("Cache-Control", "private; max-age=0")

		pro, err := a.store.GetProfile()
		if err != nil {
			if !spec.IsNotFoundError(err) {
				http.Error(w, fmt.Sprintf("cannot load profile: %v", err), http.StatusInternalServerError)
				return
			}
		}

		sendProfile(w, &pro, opts)
	} else {
		options(w, r, opts)
	}
}

func sendProfile(w http.ResponseWriter, pro *spec.Profile, opts string) {
	res := NewIdent{
		Name:    pro.Name,
		Bio:     pro.Bio,
		Lat:     float64(pro.Lat) / 10.0,  // undo quantization
		Long:    float64(pro.Long) / 10.0, // undo quantization
		Country: pro.Country,
		City:    pro.City,
		Icon:    base64.StdEncoding.EncodeToString(pro.Icon),
	}
	bytes, err := json.Marshal(res)
	if err != nil {
		http.Error(w, fmt.Sprintf("error encoding JSON: %s", err.Error()), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Length", strconv.Itoa(len(bytes)))
	w.Header().Set("Allow", opts)
	w.Write(bytes)
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
