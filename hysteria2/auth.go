package hysteria2

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/sentinel-official/sentinel-go-sdk/v2/libs/safe"
)

// authRequest represents the JSON body sent by Hysteria2 on each connection attempt.
type authRequest struct {
	Addr string `json:"addr"` // Addr is the client's remote address.
	Auth string `json:"auth"` // Auth is the credential presented by the client (UUID).
	Tx   uint64 `json:"tx"`   // Tx is the client's requested bandwidth in bytes per second.
}

// authResponse represents the JSON response returned to Hysteria2.
type authResponse struct {
	OK bool   `json:"ok"` // OK indicates whether authentication succeeded.
	ID string `json:"id"` // ID is the user identifier assigned to this session.
}

// newAuthHandler returns an http.Handler that validates incoming Hysteria2 auth requests
// against the provided peers map. A connection is accepted iff its auth field matches
// a UUID present in the map.
//
// The handler carries no shared secret of its own: it is bound to the loopback interface
// (127.0.0.1) so only the local Hysteria2 process can reach it, and the only acceptable
// credential is a valid peer UUID.
func newAuthHandler(peers *safe.Map[string, Peer]) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() { _ = r.Body.Close() }()

		// Read the request body.
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, fmt.Sprintf("reading body: %v", err), http.StatusBadRequest)

			return
		}

		// Parse the auth request.
		var req authRequest
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, fmt.Sprintf("parsing body: %v", err), http.StatusBadRequest)

			return
		}

		// Look up the UUID in the peers map.
		resp := authResponse{OK: false}
		if peers.Exists(req.Auth) {
			resp.OK = true
			resp.ID = req.Auth
		}

		// Encode and write the response.
		w.Header().Set("Content-Type", "application/json")

		data, err := json.Marshal(resp)
		if err != nil {
			http.Error(w, fmt.Sprintf("encoding response: %v", err), http.StatusInternalServerError)

			return
		}

		if _, err := w.Write(data); err != nil {
			return
		}
	})
}
