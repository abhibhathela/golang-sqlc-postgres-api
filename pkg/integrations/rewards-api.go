package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/abhishheck/gamezop-task/pkg/rewards"
	"github.com/gin-gonic/gin"
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
	url := os.Getenv("REWARDS_ENDPOINT") + "/r1/payout"
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
	url := os.Getenv("REWARDS_ENDPOINT") + "/r2/payout"
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
	return data, nil
}

func UnlockScratchCardV3() (RewardsResponse, error) {
	url := os.Getenv("REWARDS_ENDPOINT") + "/r3/payout"
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

	req.Header.Add("x-callback-url", "https://6b6e-123-201-2-130.in.ngrok.io/v1/scratch-card/callback")
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

func PollPaymentStatus(id int64, orderId string, ctx *gin.Context, queries *rewards.Queries) {
	startTime := time.Now()
	for time.Since(startTime) < 60*time.Second {
		// make the API call to get the order status
		status, err := CheckPayoutStatus(orderId)
		if err != nil {
			// handle the error
			fmt.Println("Error: ", err)
		} else {
			// print the order status
			fmt.Println("Order status: ", status)
			if status == "failed" {
				err := queries.UpdateScratchCardReward(ctx, rewards.UpdateScratchCardRewardParams{
					ID:     id,
					Status: rewards.RewardStatus(status),
				})
				if err != nil {
					fmt.Println("Error: ", err)
				}
				//! credit on fail
				Credit(orderId, id)
				return
			}
			if status == "success" {
				err := queries.UpdateScratchCardReward(ctx, rewards.UpdateScratchCardRewardParams{
					ID:     id,
					Status: rewards.RewardStatus(status),
				})
				if err != nil {
					fmt.Println("Error: ", err)
				}
				return
			}
		}
		// wait for 5 seconds before making the next API call
		time.Sleep(5 * time.Second)
	}
	fmt.Println("Reached max time limit of 60 seconds")
}

func CheckPayoutStatus(orderId string) (string, error) {
	url := os.Getenv("REWARDS_ENDPOINT") + "/r2/payout/status?order-id=" + orderId
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

func Credit(orderId string, scratchCardId int64) {
	url := os.Getenv("REWARDS_ENDPOINT") + "/credit"
	method := "PUT"

	payload := strings.NewReader(`{
		"orderId": "` + orderId + `",
		"scratchCardId": 1
	}`)

	client := &http.Client{}
	req, err := http.NewRequest(method, url, payload)

	if err != nil {
		fmt.Println(err)
		return
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()
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
