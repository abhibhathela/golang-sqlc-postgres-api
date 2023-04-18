package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/abhishheck/gamezop-task/pkg/rewards"
	"github.com/google/uuid"
)

type RewardsResponse struct {
	Code string `json:"code"`
	Data struct {
		Status  string    `json:"status"`
		OrderId uuid.UUID `json:"orderId"`
	} `json:"data"`
	Success bool   `json:"success"`
	Version string `json:"version"`
}

type OrderStatusResponse struct {
	Code string `json:"code"`
	Data struct {
		Status string `json:"status"`
	} `json:"data"`
	Success bool   `json:"success"`
	Version string `json:"version"`
}

func UnlockScratchCardV1() (RewardsResponse, error) {
	url := "http://localhost:3010/r1/payout"
	method := "POST"

	uuid, _ := uuid.NewRandom()

	payload := strings.NewReader(`{
		"scId": "` + uuid.String() + `"
	}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return RewardsResponse{}, err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error while sending request:", err)
		return RewardsResponse{}, err
	}
	defer res.Body.Close()

	var data RewardsResponse

	fmt.Println("data", data)

	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		fmt.Println("Error decoding JSON response: ????", err)
		return RewardsResponse{}, err
	}

	return data, nil
}

func UnlockScratchCardV2() (RewardsResponse, error) {
	url := "http://localhost:3010/r2/payout"
	method := "POST"

	uuid, _ := uuid.NewRandom()

	payload := strings.NewReader(`{
		"scId": "` + uuid.String() + `"
	}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return RewardsResponse{}, err
	}
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error while sending request:", err)
		return RewardsResponse{}, err
	}
	defer res.Body.Close()

	var data RewardsResponse

	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		fmt.Println("Error decoding JSON response:", err)
		return RewardsResponse{}, err
	}

	//! need to create go routine that will check the status of the order every 5 seconds
	go PoolPaymentStatus()

	return data, nil
}

func UnlockScratchCardV3() (RewardsResponse, error) {
	url := "http://localhost:3010/r3/payout"
	method := "POST"

	uuid, _ := uuid.NewRandom()

	payload := strings.NewReader(`{
		"scId": "` + uuid.String() + `"
	}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return RewardsResponse{}, err
	}

	req.Header.Add("x-callback-url", "http://localhost:5252/callback")
	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		fmt.Println("Error while sending request:", err)
		return RewardsResponse{}, err
	}
	defer res.Body.Close()

	var data RewardsResponse

	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		fmt.Println("Error decoding JSON response:", err)
		return RewardsResponse{}, err
	}

	return data, nil
}

func PoolPaymentStatus() {
	startTime := time.Now()
	for time.Since(startTime) < 60*time.Second {
		// make the API call to get the order status
		status, err := CheckPayoutStatus()
		if err != nil {
			// handle the error
			fmt.Println("Error: ", err)
		} else {
			// print the order status
			fmt.Println("Order status: ", status)
			if status != "pending" {
				//! update the database with given status and do remaining things
			}
		}
		// wait for 5 seconds before making the next API call
		time.Sleep(5 * time.Second)
	}
	fmt.Println("Reached max time limit of 60 seconds")
}

func CheckPayoutStatus() (string, error) {
	// localhost:3010/r2/payout/status?order-id=92227ca9-23eb-4945-981d-ba7a20a7fc40
	url := "localhost:3010/r2/payout/status?order-id=92227ca9-23eb-4945-981d-ba7a20a7fc40"
	method := "GET"

	payload := &bytes.Buffer{}
	writer := multipart.NewWriter(payload)
	err := writer.Close()
	if err != nil {
		fmt.Println(err)
		return "", err
	}

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", err
	}
	defer res.Body.Close()

	var data OrderStatusResponse

	err = json.NewDecoder(res.Body).Decode(&data)
	if err != nil {
		fmt.Println("Error decoding JSON response:", err)
	}
	return data.Data.Status, nil
}

func UnlockScratchCard(rewardsType rewards.RewardTypes) (RewardsResponse, error) {
	switch rewardsType {
	case rewards.RewardTypesR1:
		return UnlockScratchCardV1()
	case rewards.RewardTypesR2:
		return UnlockScratchCardV2()
	case rewards.RewardTypesR3:
		return UnlockScratchCardV3()
	}
	return RewardsResponse{}, nil
}
