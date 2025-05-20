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
			if strings.HasSuffix(h.URL.Path, "status") {
				handleUpdateStatus(services, w, h, ps)
			} else if strings.HasSuffix(h.URL.Path, "removal") {
				handleRemoveFromRing(services, w, h, ps)
			} else if strings.HasSuffix(h.URL.Path, "addition") {
				handleAddToRing(services, w, h, ps)
			} else if strings.HasSuffix(h.URL.Path, "invalidation") {
				handleInvalidateToken(services, w, h, ps)
			} else {
				handlePostTokenRing(services, w, h, ps)
			}
		} else {
			handleGetTokenRing(services, w, h, ps)
		}
	}
}

// POST method to append a gateway to token ring of specified repo
func handleUpdateStatus(services be.ActionController, w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
	fmt.Println("Received status update request")

	ctx := h.Context()
	var reqMsg struct {
		Address string `json:"address"`
		Repo    string `json:"repo"`
		Status  int    `json:"status"`
	}
	if err := json.NewDecoder(h.Body).Decode(&reqMsg); err != nil {
		httpWrapError(ctx, err, "invalid request body", w, http.StatusBadRequest)
		return
	}

	err := services.SetGwStatus(ctx, reqMsg.Repo, reqMsg.Address, reqMsg.Status)
	if err != nil {
		fmt.Println("failed to add:", reqMsg.Address, "to token ring for repo", reqMsg.Repo, ": ", err)
		replyJSON(ctx, w, message{"acknowledgement": "error", "error": err.Error()})
	} else {
		replyJSON(ctx, w, message{"acknowledgement": "ok"})
	}
}

// POST method to append a gateway to token ring of specified repo
func handleAddToRing(services be.ActionController, w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
	fmt.Println("Received request to add gw to token ring")

	ctx := h.Context()
	var reqMsg struct {
		Address string `json:"address"`
		Repo    string `json:"repo"`
	}
	if err := json.NewDecoder(h.Body).Decode(&reqMsg); err != nil {
		httpWrapError(ctx, err, "invalid request body", w, http.StatusBadRequest)
		return
	}

	err := services.AddToRing(ctx, nil /* No transaction */, reqMsg.Repo, reqMsg.Address)
	if err != nil {
		fmt.Println("failed to add:", reqMsg.Address, "to token ring for repo", reqMsg.Repo, ": ", err)
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
		Address string `json:"address"`
		Repo    string `json:"repo"`
	}
	if err := json.NewDecoder(h.Body).Decode(&reqMsg); err != nil {
		httpWrapError(ctx, err, "invalid request body", w, http.StatusBadRequest)
		return
	}

	err := services.RemoveLocally(ctx, reqMsg.Repo, reqMsg.Address)
	if err != nil {
		fmt.Println("Failed to remove", reqMsg.Address, "from token ring for repository", reqMsg.Repo, ":", err)
		replyJSON(ctx, w, message{"acknowledgement": "error", "error": err.Error()})
	} else {
		replyJSON(ctx, w, message{"acknowledgement": "ok"})
	}
}

// POST method to post token for specified rpeo to this gateway
func handlePostTokenRing(services be.ActionController, w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
	ctx := h.Context()
	var reqMsg struct {
		Repo string `json:"repo"`
	}
	if err := json.NewDecoder(h.Body).Decode(&reqMsg); err != nil {
		httpWrapError(ctx, err, "invalid request body", w, http.StatusBadRequest)
		return
	}
	err := services.AcceptRingToken(ctx, reqMsg.Repo)
	if err != nil {
		fmt.Println("Error accepting token: ", err)
		replyJSON(ctx, w, message{"acknowledgement": "error", "error": err.Error()})
	} else {
		replyJSON(ctx, w, message{"acknowledgement": "ok"})
	}
}

// POST method to invalidate token for specified repo
func handleInvalidateToken(services be.ActionController, w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
	ctx := h.Context()
	var reqMsg struct {
		Repo string `json:"repo"`
	}
	if err := json.NewDecoder(h.Body).Decode(&reqMsg); err != nil {
		httpWrapError(ctx, err, "invalid request body", w, http.StatusBadRequest)
		return
	}
	services.InvalidateToken(ctx, reqMsg.Repo)
	replyJSON(ctx, w, message{"acknowledgement": "ok"})
}

// GET method to see if this gateway has token
func handleGetTokenRing(services be.ActionController, w http.ResponseWriter, h *http.Request, ps httprouter.Params) {
	ctx := h.Context()
	var reqMsg struct {
		Repo string `json:"repo"`
	}
	if err := json.NewDecoder(h.Body).Decode(&reqMsg); err != nil {
		httpWrapError(ctx, err, "invalid request body", w, http.StatusBadRequest)
		return
	}
	msg := make(map[string]interface{})
	gateways, err := services.GetRingGatewaysStatus(ctx, reqMsg.Repo)
	if err != nil {
		fmt.Println("Error getting token ring gateways: ", err)
	}
	if len(gateways) == 0 {
		msg["status"] = "error: no gateways"
		msg["gateways"] = []string{}
	} else {
		msg["status"] = "ok"
		msg["gateways"] = gateways
	}
	if !services.HasRingToken(ctx, reqMsg.Repo) {
		msg["has_token"] = false
	} else {
		msg["has_token"] = true
	}
	replyJSON(ctx, w, msg)
}
