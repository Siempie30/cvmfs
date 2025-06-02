package backend

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	gw "github.com/cvmfs/gateway/internal/gateway"
)

// Map to store whether this gateway possesses the token per repository
var hasToken map[string]bool = make(map[string]bool)
var tokenReceptionTime map[string]time.Time = make(map[string]time.Time)
var tokenMutex sync.Mutex
var ringData struct {
	Repos []struct {
		RepoName string     `json:"repoName"`
		Gateways []gwStatus `json:"gateways"`
	} `json:"repos"`
}

type gwStatus struct {
	Address string `json:"address"`
	Status  int    `json:"status"`
}

// Initialize the token ring service by populating the token ring table in the database
func InitTokenRing(s *Services) error {
	ctx := context.Background()
	t0 := time.Now()
	outcome := "success"
	defer logAction(ctx, "init_token_ring", &outcome, t0)

	// Initialize the token state
	repos, err := s.GetRepos(ctx)
	if err != nil {
		outcome = err.Error()
		return fmt.Errorf("Error getting repositories: %w", err)
	}
	tokenMutex.Lock()
	for repo, _ := range repos {
		hasToken[repo] = false
	}
	tokenMutex.Unlock()

	// Populate the database's token ring table
	err = populateDbTokenring(ctx, s)
	if err != nil {
		outcome = err.Error()
		return fmt.Errorf("Error populating token ring table: %w", err)
	}

	return nil
}

// Populate the token ring table in the database with the gateways and their statuses for each repository.
// It attempts to retrieve the token ring configuration from the gateways specified in the repository configuration.
// If it fails to retrieve the configuration from all of the gateways, it uses the addresses in the configuration, with status 0 (up).
func populateDbTokenring(ctx context.Context, s *Services) error {
	t0 := time.Now()
	outcome := "success"
	defer logAction(ctx, "populate_db_tokenring", &outcome, t0)

	tx, err := s.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		outcome = err.Error()
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get all the repositories
	repos, err := s.GetRepos(ctx)
	if err != nil {
		outcome = err.Error()
		return fmt.Errorf("could not get repositories: %w", err)
	}

	address, err := getAddress(strconv.Itoa(s.Config.Port))
	if err != nil {
		outcome = "could not get own address: " + err.Error()
		return fmt.Errorf("could not get own address: %w", err)
	}

	// Loop through the repos
	for repoName, cfg := range repos {
		foundSelf := false

		// Attempt to retrieve token ring configuration from one of the specified gateways, until successful
		retrievedRing := false
		var gwAddresses []gwStatus
		for _, gateway := range cfg.TokenRing {
			gwAddresses, err = retrieveRingFromGw(gateway, repoName)
			if err != nil {
				errStr := fmt.Sprintf("Error retrieving ring data from gateway %s for repo %s: %v", gateway, repoName, err)
				gw.LogC(ctx, errStr, gw.LogDebug)
				continue
			}
			retrievedRing = true
			break
		}
		if !retrievedRing {
			errStr := fmt.Sprintf("Could not retrieve token ring from any gateway for repo %s, using local address instead", repoName)
			gw.LogC(ctx, errStr, gw.LogDebug)
			for _, gw := range cfg.TokenRing {
				gwAddresses = append(gwAddresses, gwStatus{
					Address: gw,
					Status:  0, // Assuming up status
				})
			}
		}

		for _, gw := range gwAddresses {
			// Insert the repo and gateway into the token ring table
			res, err := tx.ExecContext(ctx,
				"INSERT INTO TokenRing (Address, Repository, Status) VALUES (?, ?, ?);",
				gw.Address, repoName, gw.Status)
			if err != nil {
				outcome = err.Error()
				return fmt.Errorf("could not insert token ring entry: %w", err)
			}
			numInserts, err := res.RowsAffected()
			if err != nil {
				outcome = "new token ring entry" + gw.Address + "for repo" + repoName + "not inserted"
				return fmt.Errorf("new token ring entry %s for repo %s not inserted. error: %s", gw.Address, repoName, err.Error())
			}
			if err == nil && numInserts == 0 {
				outcome = "new token ring entry" + gw.Address + "for repo" + repoName + "not inserted"
				return fmt.Errorf("new token ring entry %s for repo %s not inserted", gw.Address, repoName)
			}
			if gw.Address == address {
				foundSelf = true
			}
		}
		if !foundSelf {
			// If this gateway is not in the ring, request other gateways to add it, and add it locally
			err = s.RequestAddition(ctx, tx, repoName, address)
			if err != nil {
				outcome = err.Error()
				return fmt.Errorf("could not request addition of %s to token ring for repo %s: %w", address, repoName, err)
			}
			err = s.AddToRing(ctx, tx, repoName, address)
			if err != nil {
				outcome = err.Error()
				return fmt.Errorf("could not add %s to token ring for repo %s: %w", address, repoName, err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		outcome = "could not commit transaction: " + err.Error()
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	return nil
}

// Retrieves array of gateways and status from the specified gateway for a specific repository, using the API
func retrieveRingFromGw(gwAddress string, repo string) ([]gwStatus, error) {
	url := fmt.Sprintf("%s/hagroup", gwAddress)
	payload := map[string]string{
		"repo": repo,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create GET request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve ring from gateway: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Gateways []gwStatus `json:"gateways"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.Gateways, nil
}

func (s *Services) AcceptRingToken(ctx context.Context, repository string) error {
	outcome := "success"
	defer logAction(ctx, "accept_ring_token", &outcome, time.Now())

	tokenMutex.Lock()
	reposMap, err := s.GetRepos(ctx)
	if err != nil {
		tokenMutex.Unlock()
		outcome = "failed to get repositories: " + err.Error()
		return fmt.Errorf("Error getting repositories: %w", err)
	}
	// Check if the repository is managed by this gateway
	repoFound := false
	for k := range reposMap {
		if k == repository {
			repoFound = true
			break
		}
	}
	if !repoFound {
		tokenMutex.Unlock()
		outcome = "repository not found: " + repository
		return fmt.Errorf("repository %s not found in repos", repository)
	}

	if hasToken[repository] {
		tokenMutex.Unlock()
		outcome = "token already accepted for " + repository
		return fmt.Errorf("token already accepted for %s, are there multiple tokens at play?", repository)
	}
	defer tokenMutex.Unlock()
	hasToken[repository] = true
	gw.LogC(ctx, "token_ring", gw.LogInfo).Msgf("Token accepted for repository: %s", repository)

	// Gateway has received the token, so it can start accepting leases for duration of LeaseAcquisitionTime
	tokenReceptionTime[repository] = time.Now()
	time.AfterFunc(s.Config.LeaseAcquisitionTime, func() {
		var err error
		ctx := context.Background()
		result, err := s.GetLeases(ctx)
		if err != nil {
			gw.LogC(ctx, "token_ring", gw.LogError).Msgf("Error getting leases: %v", err)
			return
		}
		if len(result) == 0 {
			// If there's no leases active, post the token to the next gateway right away
			gw.LogC(ctx, "token_ring", gw.LogInfo).Msgf("No active leases, posting token immediately")
			err = s.PostRingToken(repository)
		} else {
			// Wait for either the lease notification to signal 0 leases, or for the max lease time
			select {
			case <-s.LeaseNotificationChan:
				result, _ := s.GetLeases(ctx)
				if len(result) == 0 { // No more active leases, so token can be posted early
					gw.LogC(ctx, "token_ring", gw.LogInfo).Msgf("Last lease cancelled or committed, posting token")
					err = s.PostRingToken(repository)
					break
				}
			case <-time.After(s.Config.MaxLeaseTime):
				// Max lease time reached, so post the token
				gw.LogC(ctx, "token_ring", gw.LogInfo).Msgf("Max lease time reached, posting token")
				err = s.PostRingToken(repository)
				break
			}
		}

		if err != nil {
			gw.LogC(ctx, "token_ring", gw.LogError).Msgf("Error posting token: %v", err)
		} else {
			gw.LogC(ctx, "token_ring", gw.LogDebug).Msgf("Token posted successfully")
		}
	})

	return nil
}

// PostRingToken posts the token to the next gateway in the ring.
func (s *Services) PostRingToken(repository string) error {
	ctx := context.Background()
	outcome := "success"
	defer logAction(ctx, "post_ring_token", &outcome, time.Now())

	address, err := getAddress(strconv.Itoa(s.Config.Port))
	if err != nil {
		outcome = "could not get own address: " + err.Error()
		return fmt.Errorf("could not get own address: %w", err)
	}
	nextGw, err := s.GetNextRingGateway(ctx, repository, address, 0)
	if err != nil {
		outcome = "error getting next gateway: " + err.Error()
		return fmt.Errorf("error getting next gateway: %w", err)
	}
	err = retryPostToken(ctx, s, repository, nextGw)
	if err != nil {
		outcome = "error posting token to next gateway: " + err.Error()
		return fmt.Errorf("error posting token to next gateway: %w", err)
	}

	return nil
}

// retryPostToken posts the token to the specified gateway. targetGw should be the gateway's address
func retryPostToken(ctx context.Context, s *Services, repository string, targetGw string) error {
	// If the target is the same as the current address, skip posting
	address, err := getAddress(strconv.Itoa(s.Config.Port))
	if err != nil {
		gw.LogC(ctx, "token_ring", gw.LogError).Msgf("Error getting address: %v", err)
		return fmt.Errorf("error getting address: %w", err)
	}
	if targetGw == address {
		gw.LogC(ctx, "token_ring", gw.LogDebug).Msgf("Target gateway is the same as current address, skipping posting")
		// Update token state to prevent issue when re-accepting the token
		tokenMutex.Lock()
		hasToken[repository] = false
		tokenMutex.Unlock()
		s.AcceptRingToken(ctx, repository)
		return nil
	}
	url := fmt.Sprintf("%s/hagroup", targetGw)
	gw.LogC(ctx, "token_ring", gw.LogInfo).Msgf("Posting token for: %s to: %s", repository, url)

	// Get the gateway next to the target. This will be used if the token is not successfully posted to the target
	nextGw, err := s.GetNextRingGateway(ctx, repository, targetGw, 0)
	if err != nil {
		gw.LogC(ctx, "token_ring", gw.LogError).Msgf("Error getting next gateway: %v", err)
		return fmt.Errorf("error getting next gateway: %w", err)
	}

	// Create the payload with the repository information
	payload := map[string]string{
		"repo": repository,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		gw.LogC(ctx, "token_ring", gw.LogError).Msgf("Error marshaling payload: %v", err)
		return fmt.Errorf("error marshaling payload: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		gw.LogC(ctx, "token_ring", gw.LogInfo).Msgf("Error creating request: %v, attempting next gateway in ring", err)
		s.RequestStatusUpdate(ctx, repository, targetGw, 3)
		s.SetGwStatus(ctx, repository, targetGw, 3)
		err = retryPostToken(ctx, s, repository, nextGw)
		return err
	}
	req.Close = true
	req.Header.Set("Content-Type", "application/json")

	// Create an HTTP client with a timeout
	client := &http.Client{
		Timeout: 10 * time.Second, // Set a 10-second timeout
	}

	resp, err := client.Do(req)
	if err != nil {
		gw.LogC(ctx, "token_ring", gw.LogInfo).Msgf("Error posting token: %v, attempting next gateway in ring", err)
		s.RequestStatusUpdate(ctx, repository, targetGw, 3)
		s.SetGwStatus(ctx, repository, targetGw, 3)
		err = retryPostToken(ctx, s, repository, nextGw)
		return err
	}
	defer resp.Body.Close()

	// Parse the response
	var responseMessage map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&responseMessage); err != nil {
		gw.LogC(ctx, "token_ring", gw.LogError).Msgf("Error decoding response: %v", err)
		return fmt.Errorf("error decoding response: %w", err)
	}

	// Check the acknowledgment status
	if ack, ok := responseMessage["acknowledgement"]; ok {
		if ack == "ok" {
			gw.LogC(ctx, "token_ring", gw.LogDebug).Msgf("Acknowledgment received: ok")
		} else {
			if errMsg, exists := responseMessage["error"]; exists {
				gw.LogC(ctx, "token_ring", gw.LogInfo).Msgf("Error message: %s", errMsg)
			}
			gw.LogC(ctx, "token_ring", gw.LogInfo).Msgf("received error acknowledgment: %s, attempting next gateway in ring", ack)
			err = retryPostToken(ctx, s, repository, nextGw)
			return err
		}
	} else {
		gw.LogC(ctx, "token_ring", gw.LogInfo).Msgf("Acknowledgment not found in response. Attempting next gateway in ring")
		err = retryPostToken(ctx, s, repository, nextGw)
		return err
	}

	// Update token state
	if targetGw != address { // Only set hasToken to false if the token is not posted to self
		tokenMutex.Lock()
		hasToken[repository] = false
		tokenMutex.Unlock()
		go func() {
			// Calculate cycle time
			gateways, err := s.GetRingGatewaysStatus(ctx, repository)
			if err != nil {
				gw.LogC(ctx, "token_ring", gw.LogError).Msgf("Error getting gateways: %v", err)
				return
			}
			var functionalGateways []gwStatus
			for _, gateway := range gateways {
				if gateway.Status <= 1 { // Gateways with status 0 (up) or 1 (high load) are considered functional
					functionalGateways = append(functionalGateways, gateway)
				}
			}
			n_gateways := len(functionalGateways)
			t_node := s.Config.LeaseAcquisitionTime + s.Config.MaxLeaseTime + (10 * time.Second) // The 10 seconds are the acknowledgement time.
			t_cycle := time.Duration(n_gateways-1) * t_node
			// Wait for the duration of the cycle time
			time.Sleep(t_cycle)
			// Check if the reception time has been updated to a more recent value (meaning that the token has completed a full cycle)
			if !tokenReceptionTime[repository].After(time.Now().Add(-t_cycle)) {
				gw.LogC(ctx, "token_ring", gw.LogInfo).Msgf("Cycle time reached for repository %s, invalidating and regenerating token", repository)
				s.SendInvalidationRequest(ctx, repository)
				s.InvalidateToken(ctx, repository)
				// Take ownership of the new token
				s.AcceptRingToken(ctx, repository)
			}
		}()
	}

	return nil
}

func (s *Services) HasRingToken(ctx context.Context, repository string) bool {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	return hasToken[repository]
}

// Determines if a lease may be started based on
// 1. If the multiple gateway feature is enabled
// 2. If the gateway has the token for the specified repository
// 3. If the gateway is still within the lease acquisition time
func (s *Services) CanStartLease(ctx context.Context, repository string) bool {
	// If the multi-gateway feature is not enabled, leasee can be started regardlessly
	if !s.Config.EnableMultiGateway {
		return true
	}

	// Check if the token is received
	if !s.HasRingToken(ctx, repository) {
		return false
	}
	// Check if the token reception time is within the allowed time
	tokenMutex.Lock()
	receptionTime, exists := tokenReceptionTime[repository]
	if !exists {
		return false
	}
	tokenMutex.Unlock()
	return time.Since(receptionTime) < s.Config.LeaseAcquisitionTime
}

// getAddress constructs the address of this gateway using the hostname and port.
func getAddress(port string) (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("could not get hostname: %w", err)
	}
	address := "http://" + hostname + ":" + port + "/api/v1"
	return address, nil
}

// GetNextRingGateway returns the next gateway in the ring for the specified repository
// according to the current address. The maxStatus argument is used to ignore gateways with a status
// greater than the specified value.
// If the current address is not found in the ring, an error is returned.
func (s *Services) GetNextRingGateway(ctx context.Context, repository string, currentAddress string, maxStatus int) (string, error) {
	// Start a transaction
	tx, err := s.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return "", fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	gateways, err := s.GetRingGateways(ctx, tx, repository)
	if err != nil {
		return "", fmt.Errorf("could not get gateways: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("could not commit transaction: %w", err)
	}

	if len(gateways) == 0 {
		return "", fmt.Errorf("no gateways found for repo '%s'", repository)
	}

	for i, p := range gateways {
		if p == currentAddress {
			return gateways[(i+1)%len(gateways)], nil
		}
	}
	return "", fmt.Errorf("current address not found in gateways")
}

// RequestStatusUpdate sends a request to update the status of a gateway in the ring, to all gateways in the specified repo's ring
// TODO(siemv) I don't think this needs to be in the services interface? It can just be local?
func (s *Services) RequestStatusUpdate(ctx context.Context, repository string, address string, status int) error {
	// Start a transaction
	tx, err := s.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get all the addresses from the specified repository
	lines, err := s.GetRingGateways(ctx, tx, repository)
	if err != nil {
		return fmt.Errorf("could not get addresses: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	// Create a payload containing the gateway address, repository name, and status
	payload := map[string]interface{}{
		"address": address,
		"repo":    repository,
		"status":  status,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %w", err)
	}

	selfAddress, err := getAddress(strconv.Itoa(s.Config.Port))
	if err != nil {
		gw.LogC(ctx, "token_ring", gw.LogError).Msgf("could not get own address: %v", err)
		return fmt.Errorf("could not get own address: %w", err)
	}
	// Send HTTP addition request to each address except the current one
	for _, line := range lines {
		if line == selfAddress {
			gw.LogC(ctx, "token_ring", gw.LogDebug).Msgf("Skipping update request to self: %s", selfAddress)
			continue
		}
		url := fmt.Sprintf("%s/hagroup/status", line)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
		if err != nil {
			gw.LogC(ctx, "token_ring", gw.LogError).Msgf("could not create status update request: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			gw.LogC(ctx, "token_ring", gw.LogError).Msgf("could not send status update request: %v", err)
			continue
		}
		defer resp.Body.Close()
	}

	return nil
}

// SetGwStatus updates the status of a gateway in the token ring table of the gw db for the specified repository
func (s *Services) SetGwStatus(ctx context.Context, repository string, address string, status int) error {
	// Start a transaction
	tx, err := s.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Update the status of the gateway in the database
	_, err = tx.ExecContext(ctx,
		"UPDATE TokenRing SET Status = ? WHERE Repository = ? AND Address = ?",
		status, repository, address)
	if err != nil {
		return fmt.Errorf("could not update gateway status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	return nil
}

func (s *Services) RequestAddition(ctx context.Context, tx *sql.Tx, repository string, address string) error {
	// Get all the addresses from the specified repository
	lines, err := s.GetRingGateways(ctx, tx, repository)
	if err != nil {
		return fmt.Errorf("could not get addresses: %w", err)
	}

	// Create a payload containing the address and ring file name
	payload := map[string]string{
		"address": address,
		"repo":    repository,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %w", err)
	}

	selfAddress, err := getAddress(strconv.Itoa(s.Config.Port))
	if err != nil {
		gw.LogC(ctx, "token_ring", gw.LogError).Msgf("could not get own address: %v", err)
		return fmt.Errorf("could not get own address: %w", err)
	}

	// Send HTTP addition request to each address except the current one
	for _, line := range lines {
		if line == selfAddress {
			gw.LogC(ctx, "token_ring", gw.LogDebug).Msgf("Skipping addition request to self: %s", selfAddress)
			continue
		}
		url := fmt.Sprintf("%s/hagroup/addition", line)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
		if err != nil {
			gw.LogC(ctx, "token_ring", gw.LogInfo).Msgf("could not create gateway addition request: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			gw.LogC(ctx, "token_ring", gw.LogWarn).Msgf("could not send gateway addition request: %v", err)
			continue
		}
		defer resp.Body.Close()
	}

	return nil
}

func (s *Services) RequestRemoval(ctx context.Context, repository string, address string) error {
	// Start a transaction
	tx, err := s.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get all the addresses for the specified repository
	lines, err := s.GetRingGateways(ctx, tx, repository)
	if err != nil {
		return fmt.Errorf("could not get addresses: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	// Create a payload containing the address and repo name
	payload := map[string]string{
		"address": address,
		"repo":    repository,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %w", err)
	}

	selfAddress, err := getAddress(strconv.Itoa(s.Config.Port))
	if err != nil {
		gw.LogC(ctx, "token_ring", gw.LogError).Msgf("could not get own address: %v", err)
		return fmt.Errorf("could not get own address: %w", err)
	}

	// Send HTTP removal request to each address except the current one
	for _, line := range lines {
		if line == selfAddress {
			gw.LogC(ctx, "token_ring", gw.LogDebug).Msgf("Skipping removal request to self: %s", selfAddress)
			continue
		}
		url := fmt.Sprintf("%s/hagroup/removal", line)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
		if err != nil {
			gw.LogC(ctx, "token_ring", gw.LogWarn).Msgf("could not create gateway removal request: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			gw.LogC(ctx, "token_ring", gw.LogWarn).Msgf("could not send gateway removal request: %v", err)
			continue
		}
		defer resp.Body.Close()
	}
	gw.LogC(ctx, "token_ring", gw.LogDebug).Msgf("Removal request of %s sent to all gateways", selfAddress)
	return nil
}

// Add a gateway to the token ring for the specified repository
// If the repository does not exist or the address is already in the ring, an error is returned
// If the transaction is nil, a new transaction is started and committed at the end
func (s *Services) AddToRing(ctx context.Context, tx *sql.Tx, repository string, address string) error {
	t0 := time.Now()

	outcome := "success"
	defer logAction(ctx, "add_to_ring", &outcome, t0)

	commitTx := false
	if tx == nil {
		// Start a transaction
		var err error
		tx, err = s.DB.SQL.BeginTx(ctx, nil)
		if err != nil {
			outcome = err.Error()
			return fmt.Errorf("could not begin transaction: %w", err)
		}
		commitTx = true
		defer tx.Rollback()
	}

	_, err := tx.ExecContext(ctx,
		"INSERT INTO TokenRing (Address, Repository, Status) VALUES (?, ?, 0);",
		address, repository)
	if err != nil {
		outcome = err.Error()
		return fmt.Errorf("could not insert gateway %s into token ring for repo %s: %w", address, repository, err)
	}

	if commitTx {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("could not commit transaction: %w", err)
		}
	}

	return nil
}

func removeFromRing(ctx context.Context, repository string, address string, s *Services) error {
	err := s.RequestRemoval(ctx, repository, address)
	if err != nil {
		return fmt.Errorf("could not request removal of: %w", err)
	}
	err = s.RemoveLocally(ctx, repository, address)
	if err != nil {
		return fmt.Errorf("could not remove locally: %w", err)
	}
	return nil
}

func (s *Services) SendInvalidationRequest(ctx context.Context, repository string) error {
	// Start a transaction
	tx, err := s.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Get all the addresses from the specified repository
	lines, err := s.GetRingGateways(ctx, tx, repository)
	if err != nil {
		return fmt.Errorf("could not get addresses: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	// Create a payload containing the repository information
	payload := map[string]string{
		"repo": repository,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %w", err)
	}

	// Send HTTP invalidation request to each address, except itself
	selfAddress, err := getAddress(strconv.Itoa(s.Config.Port))
	if err != nil {
		gw.LogC(ctx, "token_ring", gw.LogError).Msgf("could not get own address: %v", err)
		return fmt.Errorf("could not get own address: %w", err)
	}
	for _, line := range lines {
		if line == selfAddress {
			gw.LogC(ctx, "token_ring", gw.LogDebug).Msgf("Skipping invalidation request to self: %s", selfAddress)
			continue
		}
		url := fmt.Sprintf("%s/hagroup/invalidation", line)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
		if err != nil {
			gw.LogC(ctx, "token_ring", gw.LogDebug).Msgf("could not create token invalidation request: %v", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			gw.LogC(ctx, "token_ring", gw.LogDebug).Msgf("could not send gateway invalidation request: %v", err)
			continue
		}
		defer resp.Body.Close()
	}

	return nil
}

func (s *Services) InvalidateToken(ctx context.Context, repository string) {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	hasToken[repository] = false
	s.CancelLeases(ctx, repository+"/")
}

func (s *Services) RemoveLocally(ctx context.Context, repository string, address string) error {
	tx, err := s.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx,
		"DELETE FROM TokenRing WHERE Repository = ? AND Address = ?",
		repository, address)
	if err != nil {
		return fmt.Errorf("could not remove gateway %s from token ring for repo %s: %w", address, repository, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("could not commit transaction: %w", err)
	}

	return nil
}

// GetRingGateways returns the addresses of all gateways in the token ring of the
// specified repository. It retrieves this information from the gateway db.
func (s *Services) GetRingGateways(ctx context.Context, tx *sql.Tx, repository string) ([]string, error) {
	// Query the database
	rows, err := tx.QueryContext(ctx, "SELECT Address FROM TokenRing WHERE Repository = ? ORDER BY Address;", repository)
	if err != nil {
		return nil, fmt.Errorf("could not query token ring: %w", err)
	}
	defer rows.Close()

	// Collect the addresses
	var addresses []string
	for rows.Next() {
		var address string
		if err := rows.Scan(&address); err != nil {
			return nil, fmt.Errorf("could not scan address: %w", err)
		}
		addresses = append(addresses, address)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return addresses, nil
}

func (s *Services) GetRingGatewaysStatus(ctx context.Context, repository string) ([]gwStatus, error) {
	tx, err := s.DB.SQL.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("could not begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Query the database
	rows, err := tx.QueryContext(ctx, "SELECT Address, Status FROM TokenRing WHERE Repository = ? ORDER BY Address;", repository)
	if err != nil {
		return nil, fmt.Errorf("could not query token ring: %w", err)
	}
	defer rows.Close()

	// Collect all gateways from the specified repo
	var gateways []gwStatus
	for rows.Next() {
		var address string
		var status int
		if err := rows.Scan(&address, &status); err != nil {
			return nil, fmt.Errorf("could not scan address: %w", err)
		}
		gateways = append(gateways, gwStatus{
			Address: address,
			Status:  status,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("could not commit transaction: %w", err)
	}

	if len(gateways) == 0 {
		return nil, fmt.Errorf("no gateways found for repo '%s'", repository)
	}

	return gateways, nil
}
