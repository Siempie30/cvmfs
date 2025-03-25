package backend

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
)

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
	_, err = http.DefaultClient.Do(req)
	if err != nil {
		fmt.Println("Error posting token:", err)
		return err
	}

	return nil
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
