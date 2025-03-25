package frontend

import (
	"fmt"
	"net/http"

	gw "github.com/cvmfs/gateway/internal/gateway"
	be "github.com/cvmfs/gateway/internal/gateway/backend"
	"github.com/julienschmidt/httprouter"
)

// MakeTokenRingHandler creates an HTTP handler for the token ring API
func MakeTokenRingHandler(services be.ActionController) httprouter.Handle {
	return func(w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
		if h.Method == "POST" {
			handlePostTokenRing(services, w, h, ps)
		} else {
			handleGetTokenRing(services, w, h, ps)
		}
	}
}

func handlePostTokenRing(services be.ActionController, w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
	fmt.Println("Received token ring")
	gw.LogC(h.Context(), "http", gw.LogInfo).Msg("Received token ring")
}

func handleGetTokenRing(services be.ActionController, w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
	fmt.Println("Get token ring")
	gw.LogC(h.Context(), "http", gw.LogInfo).Msg("Get token ring")
}
