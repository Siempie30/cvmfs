package frontend

import (
	"fmt"
	"net/http"

	be "github.com/cvmfs/gateway/internal/gateway/backend"
	"github.com/julienschmidt/httprouter"
)

// MakeTokenRingHandler creates an HTTP handler for the token ring API
func MakeTokenRingHandler(services be.ActionController) httprouter.Handle {
	return func(w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
		if h.Method == "POST" {
			handleTokenRing(services, w, h, ps)
		} else {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func handleTokenRing(services be.ActionController, w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
	fmt.Println("Received token ring")
}
