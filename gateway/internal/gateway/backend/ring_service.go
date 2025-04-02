package backend

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var hasToken bool
var tokenMutex sync.Mutex

func (s *Services) AcceptRingToken(ctx context.Context) error {
	fmt.Println("AcceptRingToken in backend called")
	tokenMutex.Lock()
	hasToken = true
	tokenMutex.Unlock()
	fmt.Println("Token accepted")

	// Post the token to the next gateway after 30 seconds
	go func() {
		fmt.Println("Waiting 30 seconds to post token to next gateway")
		<-time.After(30 * time.Second)
		currGw, err := getHostname()
		if err != nil {
			fmt.Println("Error getting hostname:", err)
			return
		}
		nextGw, err := getNextRingGateway(s.Ringfile, currGw)
		if err != nil {
			fmt.Println("Error getting next gateway:", err)
			return
		}
		err = s.PostRingToken(nextGw)
		if err != nil {
			fmt.Println("Error posting token:", err)
		} else {
			fmt.Println("Token posted to next gateway")
		}
	}()
	return nil

}

// PostRingToken posts the token to the next gateway in the ring.
// targetGw is the hostname of the gateway to which the token should be posted.
func (s *Services) PostRingToken(targetGw string) error {
	fmt.Println("PostRingToken in backend called")

	// Post the token to the next gateway
	fmt.Println("Target gateway is: ", targetGw)
	url := fmt.Sprintf("http://%s:4929/api/v1/token-ring", targetGw)
	fmt.Println("Posting to: ", url)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte("repoName")))
	if err != nil {
		fmt.Println("Error creating request")
		return err
	}

	// Create an HTTP client with a timeout
	client := &http.Client{
		Timeout: 10 * time.Second, // Set a 10-second timeout
	}

	resp, err := client.Do(req)
	if err != nil {
		// If the error is a timeout, try again on the next gateway in the ring
		if os.IsTimeout(err) {
			fmt.Println("Timeout error:", err)
			nextGw, err := getNextRingGateway(s.Ringfile, targetGw)
			if err != nil {
				fmt.Println("Error getting next gateway:", err)
				return err
			}
			err = s.PostRingToken(nextGw)
			if err != nil {
				fmt.Println("Error posting token to next gateway:", err)
				return err
			}
		}
		fmt.Println("Error posting token:", err)
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
			return fmt.Errorf("received error acknowledgment: %s", ack)
		}
	} else {
		fmt.Println("Acknowledgment not found in response")
		return fmt.Errorf("invalid response from next gateway")
	}

	// Update token state
	tokenMutex.Lock()
	hasToken = false
	tokenMutex.Unlock()
	fmt.Println("Token posted to next gateway")

	return nil
}

func (s *Services) HasRingToken(ctx context.Context) bool {
	tokenMutex.Lock()
	defer tokenMutex.Unlock()
	return hasToken
}

func getHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("could not get hostname: %w", err)
	}
	return hostname, nil
}

func getNextRingGateway(ringFile string, currentAddress string) (string, error) {
	file, err := os.Open(ringFile)
	if err != nil {
		fmt.Println("Error opening ring file:", err)
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var addresses []string
	for scanner.Scan() {
		addresses = append(addresses, strings.TrimSpace(scanner.Text()))
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error scanning file:", err)
		return "", err
	}

	for i, p := range addresses {
		if p == currentAddress {
			return addresses[(i+1)%len(addresses)], nil
		}
	}
	return "", fmt.Errorf("current address not found in ring")
}
