package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
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

func (s *Services) InitTokenRing() error {
	// Initialize the token state
	repos, err := s.GetRepositories()
	if err != nil {
		return fmt.Errorf("Error getting repositories: %w", err)
	}
	tokenMutex.Lock()
	for _, repo := range repos {
		hasToken[repo] = false
	}
	tokenMutex.Unlock()

	// Get the address of the current gateway
	address, err := getAddress(strconv.Itoa(s.Config.Port))
	if err != nil {
		return fmt.Errorf("Error getting address: %w", err)
	}

	// Check if the current gateway is already in the ring for each repository
	for _, repo := range repos {
		lines, err := s.GetRingGateways(repo)
		if err != nil {
			return fmt.Errorf("Error getting addresses for repo %s: %w", repo, err)
		}
		var found bool
		for _, line := range lines {
			if line == address {
				found = true
				break
			}
		}
		if !found {
			// Gateway is not yet in the ring, so add it
			err = s.RequestAddition(repo, address)
			if err != nil {
				return fmt.Errorf("Error adding gateway to ring:, %w", err)
			}
			// Manually add to own ring, as the frontend is not loaded and therefore cannot receive the addition request
			s.AddToRing(repo, address)
		}
	}
	return nil
}

func (s *Services) AcceptRingToken(ctx context.Context, repository string) error {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	hasToken[repository] = true
	fmt.Println("Token accepted for", repository)

	// Gateway has received the token, so it can start accepting leases for duration of LeaseAcquisitionTime
	tokenReceptionTime[repository] = time.Now()
	time.AfterFunc(s.Config.LeaseAcquisitionTime, func() {
		var err error
		result, err := s.GetLeases(context.Background()) // Using background context is not really intented and slightly hacky
		if err != nil {
			fmt.Println("Error getting leases:", err)
			return
		}
		if len(result) == 0 {
			// If there's no leases active, post the token to the next gateway right away
			fmt.Println("No active leases, posting token immediately")
			err = s.PostRingToken(repository)
		} else {
			// Wait for either the lease notification to signal 0 leases, or for the max lease time

			select {
			case <-s.LeaseNotificationChan:
				// TODO(siemv): I believe this only works if there is only one active lease after the lease acquisition period has passed. Check this!
				result, _ := s.GetLeases(context.Background())
				if len(result) == 0 { // No more active leases, so token can be posted early
					fmt.Println("Last lease cancelled or committed, posting token")
					err = s.PostRingToken(repository)
					break
				}
			case <-time.After(s.Config.MaxLeaseTime):
				// Max lease time reached, so post the token
				fmt.Println("Max lease time reached, posting token")
				err = s.PostRingToken(repository)
				break
			}
		}

		if err != nil {
			fmt.Println("Error posting token:", err)
		} else {
			fmt.Println("Token posted successfully")
		}
	})
	return nil

}

// PostRingToken posts the token to the next gateway in the ring.
func (s *Services) PostRingToken(repository string) error {
	address, err := getAddress(strconv.Itoa(s.Config.Port))
	if err != nil {
		fmt.Println("Error getting address:", err)
		return err
	}
	nextGw, err := s.GetNextRingGateway(repository, address, 0)
	if err != nil {
		fmt.Println("Error getting next gateway:", err)
		return err
	}
	err = s.RetryPostToken(repository, nextGw)
	if err != nil {
		fmt.Println("Error posting token to next gateway:", err)
		return err
	}
	fmt.Println("Token posted to next gateway")
	return nil
}

// retryPostToken posts the token to the specified gateway. targetGw should be the gateway's address
func (s *Services) RetryPostToken(repository string, targetGw string) error {
	// Post the token to the next gateway
	fmt.Println("Target gateway is: ", targetGw)
	// If the target is the same as the current address, skip posting
	address, err := getAddress(strconv.Itoa(s.Config.Port))
	if err != nil {
		fmt.Println("Error getting address:", err)
	}
	if targetGw == address {
		fmt.Println("Target gateway is the same as current address, skipping posting")
		tokenMutex.Lock()
		hasToken[repository] = false
		tokenMutex.Unlock()
		s.AcceptRingToken(context.Background(), repository)
		return nil
	}
	url := fmt.Sprintf("%s/token-ring", targetGw)
	fmt.Println("Posting token for:", repository, "to:", url)

	// Get the gateway next to the target. This will be used if the token is not successfully posted to the target
	nextGw, err := s.GetNextRingGateway(repository, targetGw, 0)
	if err != nil {
		fmt.Println("Error getting next gateway:", err)
		return err
	}

	// Create the payload with the repository information
	payload := map[string]string{
		"repo": repository,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshaling payload:", err)
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("Error creating request: ", err, "attempting next gateway in ring")
		s.RequestStatusUpdate(repository, targetGw, 3)
		s.SetGwStatus(repository, targetGw, 3)
		err = s.RetryPostToken(repository, nextGw)
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
		fmt.Println("Error posting token:", err, "attempting next gateway in ring")
		s.RequestStatusUpdate(repository, targetGw, 3)
		s.SetGwStatus(repository, targetGw, 3)
		err = s.RetryPostToken(repository, nextGw)
		return err
	}
	defer resp.Body.Close()

	// Parse the response
	var responseMessage map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&responseMessage); err != nil {
		fmt.Println("Error decoding response:", err)
		return err
	}

	// Check the acknowledgment status
	if ack, ok := responseMessage["acknowledgement"]; ok {
		if ack == "ok" {
			fmt.Println("Acknowledgment received: ok")
		} else {
			fmt.Printf("Acknowledgment received: %s\n", ack)
			if errMsg, exists := responseMessage["error"]; exists {
				fmt.Printf("Error message: %s\n", errMsg)
			}
			fmt.Println("received error acknowledgment:", ack, "attempting next gateway in ring")
			err = s.RetryPostToken(repository, nextGw)
			return err
		}
	} else {
		fmt.Println("Acknowledgment not found in response. Attempting next gateway in ring")
		err = s.RetryPostToken(repository, nextGw)
		return err
	}

	// Update token state
	address, err = getAddress(strconv.Itoa(s.Config.Port))
	if err != nil {
		fmt.Println("Error getting address:", err)
	}
	if targetGw != address { // Only set hasToken to false if the token is not posted to self
		tokenMutex.Lock()
		hasToken[repository] = false
		tokenMutex.Unlock()
		go func() {
			// Calculate cycle time
			gateways, err := s.GetRingGateways(repository)
			if err != nil {
				fmt.Println("Error getting gateways:", err)
				return
			}
			n_gateways := len(gateways)
			t_node := s.Config.LeaseAcquisitionTime + s.Config.MaxLeaseTime + (10 * time.Second) // The 10 seconds are the acknowledgement time.
			t_cycle := time.Duration(n_gateways-1) * t_node
			// Wait for the duration of the cycle time
			time.Sleep(t_cycle)
			// Check if the reception time has been updated to a more recent value (meaning that the token has completed a full cycle)
			if !tokenReceptionTime[repository].After(time.Now().Add(-t_cycle)) {
				// If not, generate a new token
				fmt.Println("Cycle time reached, invalidating and regenerating token")
				s.SendInvalidationRequest(repository)
				s.InvalidateToken(context.Background(), repository)
				// Take ownership of the new token
				s.AcceptRingToken(context.Background(), repository)
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

func (s *Services) CanStartLease(ctx context.Context, repository string) bool {
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
// greateer than the specified value.
// If the current address is not found in the ring, an error is returned.
func (s *Services) GetNextRingGateway(repository string, currentAddress string, maxStatus int) (string, error) {
	gateways, err := s.GetRingGateways(repository)

	if err != nil {
		return "", fmt.Errorf("could not get gateways: %w", err)
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
func (s *Services) RequestStatusUpdate(repository string, address string, status int) error {
	// Get all the addresses from the specified repository
	lines, err := s.GetRingGateways(repository)
	if err != nil {
		return fmt.Errorf("could not get addresses: %w", err)
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
		fmt.Println("could not get address:", err)
	}
	// Send HTTP addition request to each address except the current one
	for _, line := range lines {
		if line == selfAddress {
			fmt.Println("Skipping update request to self:", selfAddress)
			continue
		}
		url := fmt.Sprintf("%s/token-ring/status", line)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
		if err != nil {
			fmt.Println("could not create status update request:", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("could not send status update request:", err)
			continue
		}
		defer resp.Body.Close()
	}

	return nil
}

func (s *Services) SetGwStatus(repository string, address string, status int) error {
	file, err := os.OpenFile(s.Ringfile, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("could not open ring file: %w", err)
	}
	defer file.Close()

	// Decode the existing JSON structure
	if err := json.NewDecoder(file).Decode(&ringData); err != nil {
		return fmt.Errorf("could not decode ring file: %w", err)
	}

	// Update the status of the specified gateway in the specified repository
	var gatewayFound bool
	for i, repo := range ringData.Repos {
		if repo.RepoName == repository {
			for j, gateway := range repo.Gateways {
				if gateway.Address == address {
					ringData.Repos[i].Gateways[j].Status = status
					gatewayFound = true
					break
				}
			}
			break
		}
	}

	if !gatewayFound {
		return fmt.Errorf("gateway %s not found in repository %s", address, repository)
	}

	// Write the updated JSON structure back to the file
	file.Seek(0, 0)  // Reset file pointer to the beginning
	file.Truncate(0) // Clear the file content
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty-print JSON
	if err := encoder.Encode(&ringData); err != nil {
		return fmt.Errorf("could not encode ring file: %w", err)
	}

	fmt.Println("Updated status of gateway:", address, "in repository:", repository)
	return nil
}

func (s *Services) RequestAddition(repository string, address string) error {
	// Get all the addresses from the specified repository
	lines, err := s.GetRingGateways(repository)
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

	// Send HTTP addition request to each address except the current one
	for _, line := range lines {
		address, err := getAddress(strconv.Itoa(s.Config.Port))
		if err != nil {
			fmt.Println("could not get address:", err)
			continue
		}
		if line == address {
			fmt.Println("Skipping addition request to self:", address)
			continue
		}
		url := fmt.Sprintf("%s/token-ring/addition", line)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
		if err != nil {
			fmt.Println("could not create gw addition request:", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("could not send gw addition request:", err)
			continue
		}
		defer resp.Body.Close()
	}

	return nil
}

func (s *Services) RequestRemoval(repository string, address string) error {
	// Get all the addresses for the specified repository
	lines, err := s.GetRingGateways(repository)
	if err != nil {
		return fmt.Errorf("could not get addresses: %w", err)
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

	// Send HTTP removal request to each address except the current one
	for _, line := range lines {
		if line == address {
			fmt.Println("Skipping removal request to self:", address)
			continue
		}
		url := fmt.Sprintf("%s/token-ring/removal", line)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
		if err != nil {
			fmt.Println("could not create gw removal request:", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("could not send gw removal request:", err)
			continue
		}
		defer resp.Body.Close()
	}
	fmt.Println("Removal request of", address, "sent to all gateways")
	return nil
}

func (s *Services) AddToRing(repository string, address string) error {
	// Load the ring file contents
	file, err := os.OpenFile(s.Ringfile, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("could not open ring file: %w", err)
	}
	defer file.Close()

	// Decode the existing JSON structure
	if err := json.NewDecoder(file).Decode(&ringData); err != nil {
		return fmt.Errorf("could not decode ring file: %w", err)
	}

	// Check if the specified repo exists
	var repoFound bool
	for i, repo := range ringData.Repos {
		if repo.RepoName == repository {
			repoFound = true
			// Check if the address is already in the ring
			for _, gateway := range repo.Gateways {
				if gateway.Address == address {
					fmt.Println("Address already in gateways")
					return nil
				}
			}
			// Append the address to the ring
			ringData.Repos[i].Gateways = append(ringData.Repos[i].Gateways, struct {
				Address string `json:"address"`
				Status  int    `json:"status"`
			}{
				Address: address,
				Status:  0,
			})
			break
		}
	}

	// Repo does not exist, so ignore the addition
	if !repoFound {
		fmt.Println("Repo", repository, "not found in ring file, ignoring addition")
		return nil
	}

	// Write the updated JSON structure back to the file
	file.Seek(0, 0)  // Reset file pointer to the beginning
	file.Truncate(0) // Clear the file content
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty-print JSON
	if err := encoder.Encode(&ringData); err != nil {
		return fmt.Errorf("could not encode ring file: %w", err)
	}

	fmt.Println("Added to ring file:", address)
	return nil
}

func removeFromRing(repository string, address string, s *Services) error {
	err := s.RequestRemoval(repository, address)
	if err != nil {
		return fmt.Errorf("could not request removal of: %w", err)
	}
	err = s.RemoveLocally(repository, address)
	if err != nil {
		return fmt.Errorf("could not remove locally: %w", err)
	}
	return nil
}

func (s *Services) SendInvalidationRequest(repository string) error {
	// Get all the addresses from the specified repository
	lines, err := s.GetRingGateways(repository)
	if err != nil {
		return fmt.Errorf("could not get addresses: %w", err)
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
	address, err := getAddress(strconv.Itoa(s.Config.Port))
	for _, line := range lines {
		if line == address {
			fmt.Println("Skipping invalidation request to self:", address)
			continue
		}
		url := fmt.Sprintf("%s/token-ring/invalidation", line)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
		if err != nil {
			fmt.Println("could not create token invalidation request:", err)
			continue
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("could not send gw invalidation request:", err)
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

func (s *Services) RemoveLocally(repository string, address string) error {
	// Load the ring file contents
	file, err := os.OpenFile(s.Ringfile, os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("could not open ring file: %w", err)
	}
	defer file.Close()

	// Decode the existing JSON structure
	if err := json.NewDecoder(file).Decode(&ringData); err != nil {
		return fmt.Errorf("could not decode ring file: %w", err)
	}

	// Check if the specified repo exists
	var repoFound bool
	for i, repo := range ringData.Repos {
		if repo.RepoName == repository {
			repoFound = true
			// Remove the address from the gateways
			var updatedGateways []gwStatus
			for _, gateway := range repo.Gateways {
				if gateway.Address != address {
					updatedGateways = append(updatedGateways, gateway)
				}
			}
			ringData.Repos[i].Gateways = updatedGateways
			break
		}
	}

	// If the repo does not exist, ignore the removal
	if !repoFound {
		fmt.Println("repo", repository, "not found in ring file")
		return nil
	}

	// Write the updated JSON structure back to the file
	file.Seek(0, 0)  // Reset file pointer to the beginning
	file.Truncate(0) // Clear the file content
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ") // Pretty-print JSON
	if err := encoder.Encode(&ringData); err != nil {
		return fmt.Errorf("could not encode ring file: %w", err)
	}

	fmt.Println("Removed from ring file:", address)
	return nil
}

func (s *Services) GetRepositories() ([]string, error) {
	// Load the ring file contents
	file, err := os.Open(s.Ringfile)
	if err != nil {
		return nil, fmt.Errorf("could not open ring file: %w", err)
	}
	defer file.Close()

	// Decode the existing JSON structure
	if err := json.NewDecoder(file).Decode(&ringData); err != nil {
		return nil, fmt.Errorf("could not decode ring file: %w", err)
	}

	var repositories []string
	for _, repo := range ringData.Repos {
		repositories = append(repositories, repo.RepoName)
	}

	return repositories, nil
}

func (s *Services) GetRingGateways(repository string) ([]string, error) {
	file, err := os.Open(s.Ringfile)
	if err != nil {
		return nil, fmt.Errorf("could not open ring file: %w", err)
	}
	defer file.Close()

	// Decode the JSON structure
	if err := json.NewDecoder(file).Decode(&ringData); err != nil {
		return nil, fmt.Errorf("could not decode ring file: %w", err)
	}

	// Collect all gateways from the specified repo
	var addresses []string
	for _, repo := range ringData.Repos {
		if repo.RepoName == repository {
			for _, gateway := range repo.Gateways {
				addresses = append(addresses, gateway.Address)
			}
			break
		}
	}

	if len(addresses) == 0 {
		return nil, fmt.Errorf("no gateways found for repo '%s'", repository)
	}

	return addresses, nil
}

func (s *Services) GetRingGatewaysStatus(repository string) ([]gwStatus, error) {
	file, err := os.Open(s.Ringfile)
	if err != nil {
		return nil, fmt.Errorf("could not open ring file: %w", err)
	}
	defer file.Close()

	// Decode the JSON structure
	if err := json.NewDecoder(file).Decode(&ringData); err != nil {
		return nil, fmt.Errorf("could not decode ring file: %w", err)
	}

	// Collect all gateways from the specified repo
	var gateways []gwStatus
	for _, repo := range ringData.Repos {
		if repo.RepoName == repository {
			for _, gateway := range repo.Gateways {
				gateways = append(gateways, gateway)
			}
			break
		}
	}

	if len(gateways) == 0 {
		return nil, fmt.Errorf("no gateways found for repo '%s'", repository)
	}

	return gateways, nil
}
