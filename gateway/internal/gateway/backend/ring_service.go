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
		err := s.PostRingToken(ctx)
		if err != nil {
			fmt.Println("Error posting token:", err)
		} else {
			fmt.Println("Token posted to next gateway")
		}
	}()
	return nil

}

func (s *Services) PostRingToken(ctx context.Context) error {
	fmt.Println("PostRingToken in backend called")
	fileName := s.Ringfile

	// Get the address of the next gateway
	hostName, err := os.Hostname()
	if err != nil {
		fmt.Println("Error getting hostname")
		return err
	}
	fmt.Println("Hostname is: ", hostName)
	nextAddress, err := getNextRingGateway(fileName, hostName)
	if err != nil {
		fmt.Println("Error getting next gateway: ", err)
		return err
	}

	// Post the token to the next gateway
	fmt.Println("Next gateway is: ", nextAddress)
	url := fmt.Sprintf("http://%s:4929/api/v1/token-ring", nextAddress)
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
