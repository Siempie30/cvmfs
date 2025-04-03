package frontend

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	be "github.com/cvmfs/gateway/internal/gateway/backend"
	"github.com/julienschmidt/httprouter"
)

// MakeTokenRingHandler creates an HTTP handler for the token ring API
func MakeTokenRingHandler(services be.ActionController) httprouter.Handle {
	return func(w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
		if h.Method == "POST" {
			if strings.HasSuffix(h.URL.Path, "removal") {
				fmt.Println("Received request to remove gw from token ring")
				handleRemoveFromRing(services, w, h, ps)
			} else if strings.HasSuffix(h.URL.Path, "addition") {
				handleAddToRing(services, w, h, ps)
			} else {
				handlePostTokenRing(services, w, h, ps)
			}
		} else {
			handleGetTokenRing(services, w, h, ps)
		}
	}
}

// POST method to append a gateway to token ring
func handleAddToRing(services be.ActionController, w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
	fmt.Println("Received request to add gw to token ring")

	ctx := h.Context()
	var reqMsg struct {
		HostName string `json:"hostName"`
		RingFile string `json:"ringFile"`
	}
	if err := json.NewDecoder(h.Body).Decode(&reqMsg); err != nil {
		httpWrapError(ctx, err, "invalid request body", w, http.StatusBadRequest)
		return
	}

	err := services.AddToRing(reqMsg.HostName, reqMsg.RingFile)
	if err != nil {
		fmt.Println("failed to add:", reqMsg.HostName, "to token ring: ", err)
		replyJSON(ctx, w, message{"acknowledgement": "error", "error": err.Error()})
	} else {
		replyJSON(ctx, w, message{"acknowledgement": "ok"})
	}
}

// POST method to remove a gateway from token ring
func handleRemoveFromRing(services be.ActionController, w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
	fmt.Println("Received request to remove gw from token ring")

	ctx := h.Context()
	var reqMsg struct {
		HostName string `json:"hostName"`
		RingFile string `json:"ringFile"`
	}
	if err := json.NewDecoder(h.Body).Decode(&reqMsg); err != nil {
		httpWrapError(ctx, err, "invalid request body", w, http.StatusBadRequest)
		return
	}

	err := services.RemoveFromRing(reqMsg.HostName, reqMsg.RingFile)
	if err != nil {
		fmt.Println("Failed to remove", reqMsg.HostName, "from token ring:", err)
		replyJSON(ctx, w, message{"acknowledgement": "error", "error": err.Error()})
	} else {
		replyJSON(ctx, w, message{"acknowledgement": "ok"})
	}
}

// POST method to post token to this gateway
func handlePostTokenRing(services be.ActionController, w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
	fmt.Println("Received token ring")

	ctx := h.Context()
	err := services.AcceptRingToken(ctx)
	if err != nil {
		fmt.Println("Error posting token: ", err)
		replyJSON(ctx, w, message{"acknowledgement": "error", "error": err.Error()})
	} else {
		replyJSON(ctx, w, message{"acknowledgement": "ok"})
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
