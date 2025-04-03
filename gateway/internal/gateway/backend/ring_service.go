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
		err := s.PostRingToken()
		if err != nil {
			fmt.Println("Error posting token:", err)
		} else {
			fmt.Println("Token posted to next gateway")
		}
	}()
	return nil

}

// PostRingToken posts the token to the next gateway in the ring.
func (s *Services) PostRingToken() error {
	currGw, err := getHostname()
	if err != nil {
		fmt.Println("Error getting hostname:", err)
		return err
	}
	nextGw, err := getNextRingGateway(s.Ringfile, currGw)
	if err != nil {
		fmt.Println("Error getting next gateway:", err)
		return err
	}
	err = retryPostToken(nextGw, s.Ringfile)
	if err != nil {
		fmt.Println("Error posting token to next gateway:", err)
		return err
	}
	fmt.Println("Token posted to next gateway")
	return nil
}

// RetryPostToken posts the token to the specified gateway. targetGw should be the gateway hostname
func retryPostToken(targetGw string, ringFile string) error {
	fmt.Println("PostRingToken in backend called")

	// Post the token to the next gateway
	fmt.Println("Target gateway is: ", targetGw)
	url := fmt.Sprintf("http://%s:4929/api/v1/token-ring", targetGw)
	fmt.Println("Posting to: ", url)

	// Get the gateway next to the target. This will be used if the token is not succesfully posted to the target
	nextGw, err := getNextRingGateway(ringFile, targetGw)
	if err != nil {
		fmt.Println("Error getting next gateway:", err)
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte("repoName")))
	if err != nil {
		fmt.Println("Error creating request: ", err, "attempting next gateawy in ring")
		err = nil
		err = retryPostToken(nextGw, ringFile)
		return err
	}

	// Create an HTTP client with a timeout
	client := &http.Client{
		Timeout: 10 * time.Second, // Set a 10-second timeout
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error posting token:", err, "attempting next gateway in ring")
		err = nil
		err = retryPostToken(nextGw, ringFile)
		if err != nil {
			fmt.Println("Error posting token to next gateway:", err)
		}
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
			err = nil
			err = retryPostToken(nextGw, ringFile)
			if err != nil {
				fmt.Println("Error posting token to next gateway:", err)
			}
			return err
		}
	} else {
		fmt.Println("Acknowledgment not found in response. Attempting next gateway in ring")
		err = nil
		err = retryPostToken(nextGw, ringFile)
		if err != nil {
			fmt.Println("Error posting token to next gateway:", err)
		}
		return err
	}

	// Update token state
	tokenMutex.Lock()
	hasToken = false
	tokenMutex.Unlock()

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
