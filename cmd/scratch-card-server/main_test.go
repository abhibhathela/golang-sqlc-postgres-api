package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/abhishheck/gamezop-task/pkg/integrations"
	"github.com/abhishheck/gamezop-task/pkg/rewards"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func SetupApp(testClient *http.Client) (*gin.Engine, *rewards.Queries, func()) {

	connStr := "postgres://postgres:root@localhost/rewards_test?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		fmt.Println("failed to open db connection")
		panic(err)
	}

	// check the connection
	if err := db.Ping(); err != nil {
		fmt.Println("failed to ping db")
		panic(err)
	}

	runMigrate(db)
	repository := rewards.New(db)

	//! insert test data

	//! insert user
	ctxBack := context.Background()
	_, err = repository.CreateUser(ctxBack, rewards.CreateUserParams{
		Name:         "test",
		ScratchCards: 1000,
	})
	if err != nil {
		fmt.Println("failed to create user")
		panic(err)
	}

	//! insert scratch card
	_, err = repository.CreateScratchCard(ctxBack, rewards.CreateScratchCardParams{
		Schedule:        sql.NullString{String: "* * * * 1-5", Valid: true},
		MaxCards:        sql.NullInt32{Int32: 100, Valid: true},
		RewardType:      rewards.RewardTypesR1,
		MaxCardsPerUser: sql.NullInt32{},
		Weight:          6,
	})

	if err != nil {
		fmt.Println("failed to create scratch card")
		panic(err)
	}

	return Router(db, repository, testClient), repository, func() {
		db.Close()
	}

}

// scratch-card/list
func TestListScratchCardsAPICall(t *testing.T) {

	router, _, close := SetupApp(&http.Client{})
	defer close()

	req, err := http.NewRequest("POST", "/v1/scratch-card/list", nil)
	require.Nil(t, err, "failed to create request")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	responseData, err := ioutil.ReadAll(w.Body)
	require.Nil(t, err, "failed to read response body")

	println(string(responseData))

	require.Equal(t, http.StatusOK, w.Code, "response code is not 200")
}

func TestUnlockScratchCard(t *testing.T) {

	old := scUnlock
	defer func() { scUnlock = old }()

	scUnlock = func(rewardsType rewards.RewardTypes) (integrations.RewardsResponse, error) {

		fmt.Println(">>>>>>>>>>>>>>>>>>>. what the fuck")

		res := integrations.RewardsResponse{
			Success: true,
			Version: "v1",
		}

		res.Data.Status = "pending"
		res.Data.OrderId = uuid.New()

		return res, nil
	}

	router, _, close := SetupApp(&http.Client{})
	defer close()

	body := unlockScratchCardRequest{
		UserId: 1,
	}

	bodyStr, err := json.Marshal(body)
	require.Nil(t, err, "failed to marshal body")

	req, err := http.NewRequest("POST", "/v1/scratch-card/unlock", strings.NewReader(string(bodyStr)))
	require.Nil(t, err, "failed to create request")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	responseData, err := ioutil.ReadAll(w.Body)
	require.Nil(t, err, "failed to read response body")

	fmt.Println(string(responseData))

	require.Equal(t, http.StatusOK, w.Code, "response code is not 200")
}
