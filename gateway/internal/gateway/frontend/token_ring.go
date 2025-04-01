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
			handlePostTokenRing(services, w, h, ps)
		} else {
			handleGetTokenRing(services, w, h, ps)
		}
	}
}

// POST method to post token to this gateway
func handlePostTokenRing(services be.ActionController, w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
	fmt.Println("Received token ring")

	ctx := h.Context()
	err := services.AcceptRingToken(ctx)
	if err != nil {
		fmt.Println("Error posting token: ", err)
		replyJSON(ctx, w, message{"status": "error", "error": err.Error()})
	} else {
		replyJSON(ctx, w, message{"status": "ok"})
	}
}

// GET method to see if this gateway has token
func handleGetTokenRing(services be.ActionController, w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
	if !services.HasRingToken(h.Context()) {
		fmt.Println("No token")
		replyJSON(h.Context(), w, message{"status": "no token"})
	} else {
		fmt.Println("Has token")
		replyJSON(h.Context(), w, message{"status": "has token"})
	}
}
