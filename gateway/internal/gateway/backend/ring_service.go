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

// retryPostToken posts the token to the specified gateway. targetGw should be the gateway hostname
func retryPostToken(targetGw string, ringFile string) error {
	// Post the token to the next gateway
	fmt.Println("Target gateway is: ", targetGw)
	url := fmt.Sprintf("http://%s:4929/api/v1/token-ring", targetGw)
	fmt.Println("Posting to: ", url)

	// Get the gateway next to the target. This will be used if the token is not succesfully posted to the target
	nextGw, err := getNextRingGateway(ringFile, targetGw)
	if err != nil {
		fmt.Println("Error getting next gateway:", err)
		return err
	}

	req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer([]byte("repoName")))
	if err != nil {
		fmt.Println("Error creating request: ", err, "attempting next gateway in ring")
		requestRemoval(targetGw, ringFile)
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
		requestRemoval(targetGw, ringFile)
		err = retryPostToken(nextGw, ringFile)
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
			err = retryPostToken(nextGw, ringFile)
			return err
		}
	} else {
		fmt.Println("Acknowledgment not found in response. Attempting next gateway in ring")
		err = retryPostToken(nextGw, ringFile)
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

func requestRemoval(hostName string, ringFile string) error {
	// Get all the hostnames from the ring file
	lines, err := getHostnames(ringFile)
	if err != nil {
		return fmt.Errorf("could not get hostnames: %w", err)
	}

	// Create a payload containing the hostname and ring file content
	payload := map[string]string{
		"hostName": hostName,
		"ringFile": ringFile,
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("could not marshal payload: %w", err)
	}

	// Send HTTP removal request to each hostname
	for _, line := range lines {
		url := fmt.Sprintf("http://%s:4929/api/v1/removal", line)
		req, err := http.NewRequest(http.MethodPost, url, bytes.NewBuffer(payloadBytes))
		if err != nil {
			fmt.Println("could not create gw removal request:", err)
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			fmt.Println("could not send gw removal request:", err)
		}
		defer resp.Body.Close()
	}
	fmt.Println("Removal request of", hostName, "sent to all gateways")
	return nil
}

func (s *Services) RemoveFromRing(hostName string, ringFile string) error {
	// Get all the hostnames from the ringfile
	lines, err := getHostnames(ringFile)
	if err != nil {
		return fmt.Errorf("could not get hostnames: %w", err)
	}

	// Copy all lines except the one scheduled for removal
	var newLines []string
	for _, line := range lines {
		if line != hostName {
			newLines = append(newLines, line)
		}
	}

	// Write the new lines back to the ring file
	file, err := os.Create(ringFile)
	if err != nil {
		return fmt.Errorf("could not create ring file: %w", err)
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	for _, line := range newLines {
		if _, err := writer.WriteString(line + "\n"); err != nil {
			return fmt.Errorf("could not write to ring file: %w", err)
		}
	}
	if err := writer.Flush(); err != nil {
		return fmt.Errorf("could not flush ring file: %w", err)
	}
	fmt.Println("Removed from ring file:", hostName)
	return nil
}

func getHostnames(ringFile string) ([]string, error) {
	file, err := os.Open(ringFile)
	if err != nil {
		return nil, fmt.Errorf("could not open ring file: %w", err)
	}
	defer file.Close()
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("could not read ring file: %w", err)
	}
	return lines, nil
}
