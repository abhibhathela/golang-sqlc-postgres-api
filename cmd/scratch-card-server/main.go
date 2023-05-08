package main

import (
	"database/sql"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"os"

	"github.com/abhishheck/golang-api/pkg/helpers"
	"github.com/abhishheck/golang-api/pkg/integrations"
	"github.com/abhishheck/golang-api/pkg/rewards"
	"github.com/gin-gonic/gin"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

type unlockScratchCardRequest struct {
	UserId int64 `json:"user_id" binding:"required"`
}

type rewardCallBackRequest struct {
	ID      int64  `json:"ID"`
	OrderID string `json:"OrderID"`
	Status  string `json:"Status"`
	ScID    string `json:"ScID"`
}

func runMigrate(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})

	if err != nil {
		fmt.Println("failed to create postgres driver")
		panic(err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file:///Users/Abhishek/Developer/Go/golang-api/migrations",
		"postgres", driver)

	if err != nil {
		fmt.Println("failed to create migrate instance")
		panic(err)
	}

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		fmt.Println("failed to run migrations")
		panic(err)
	}
}

func HandleRootRouter() gin.HandlerFunc {
	return func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "hello gin, grant my wish"})
	}
}

var scUnlock = integrations.UnlockScratchCard

func HandleScratchCardUnlock(db *sql.DB, repository *rewards.Queries) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var unlockScratchCardRequest unlockScratchCardRequest
		if err := ctx.BindJSON(&unlockScratchCardRequest); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		userId := unlockScratchCardRequest.UserId

		// validate user has scratch cards to unlock
		user, err := repository.GetUser(ctx, userId)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if user.ScratchCards == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "user has no scratch cards to unlock"})
			return
		}

		rewardsRows, err := repository.GetScratchCards(ctx)

		fmt.Printf("rewardsRows: %v", rewardsRows)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		availableRewards := make([]rewards.ScratchCard, 0)
		for _, sc := range rewardsRows {
			fmt.Println("reward", sc)
			if !sc.Schedule.Valid && !sc.MaxCards.Valid && !sc.MaxCardsPerUser.Valid {
				fmt.Println("no schedule, max_cards, max_cards_per_user")
				break
			}

			if sc.Schedule.Valid {
				//! this validation can be cached in redis on daily basis to avoid iterating over the same schedule
				_, err := helpers.IsValidDateToUnlockReward(sc.Schedule.String)
				if err != nil {
					fmt.Println("error while validating the schedule", err)
					break
				}
			}

			if sc.MaxCards.Valid {
				//! find how many are unlocked
				redeemedCardsCount, err := repository.GetUnlockedScratchCardRewardCount(ctx, sc.ID)
				if err != nil {
					fmt.Println("error while getting the unlocked scratch card count", err)
					break
				}
				if redeemedCardsCount > int64(sc.MaxCards.Int32) {
					break
				}
			}

			if sc.MaxCardsPerUser.Valid {
				//! find how many are unlocked
				var args rewards.GetUnlockedScratchCardRewardCountByUserParams = rewards.GetUnlockedScratchCardRewardCountByUserParams{
					UserID:        userId,
					ScratchCardID: sc.ID,
				}

				redeemedCardsCount, err := repository.GetUnlockedScratchCardRewardCountByUser(ctx, args)
				if err != nil {
					break
				}

				if redeemedCardsCount > int64(sc.MaxCardsPerUser.Int32) {
					break
				}
			}

			availableRewards = append(availableRewards, sc)
		}

		fmt.Println("availableRewards", availableRewards)

		//! from aviailable rewards, caluculate the probability and unlock the scratch card
		if len(availableRewards) == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "no scratch cards available to unlock"})
			return
		}

		var totalWeight int32 = 0
		for _, sc := range availableRewards {
			totalWeight += sc.Weight
		}

		cumulativeWeights := make([]int32, 0)
		var cumulativeWeight int32 = 0
		for _, sc := range availableRewards {
			cumulativeWeight += sc.Weight
			cumulativeWeights = append(cumulativeWeights, cumulativeWeight)
		}

		// Generate a random number between 1 and totalWeight
		randomNum := rand.Int31n(totalWeight) + 1

		selectedItem := rewards.ScratchCard{}
		for i, item := range availableRewards {
			if randomNum <= cumulativeWeights[i] {
				selectedItem = item
				break
			}
		}

		fmt.Printf("Selected %s\n", selectedItem.RewardType)

		response, err := scUnlock(selectedItem.RewardType)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// update the user scratch cards count

		tx, err := db.Begin()

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		defer tx.Rollback()

		qtx := repository.WithTx(tx)

		err = qtx.DeductScratchCard(ctx, user.ID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		fmt.Println(rewards.RewardStatus(response.Data.Status))

		var args rewards.CreateScratchCardRewardParams = rewards.CreateScratchCardRewardParams{
			UserID:        user.ID,
			ScratchCardID: selectedItem.ID,
			Status:        rewards.RewardStatus(response.Data.Status),
			OrderID:       response.Data.OrderId.String(),
		}

		newSc, err := qtx.CreateScratchCardReward(ctx, args)
		if err != nil {
			fmt.Println("error while creating the scratch card reward", err)
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		err = tx.Commit()

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if response.Data.Status == "failed" {
			integrations.Credit(newSc.OrderID, newSc.ID)
		}

		if selectedItem.RewardType == rewards.RewardTypesR2 {
			go integrations.PollPaymentStatus(newSc.ID, response.Data.OrderId.String(), ctx, repository)
		}

		ctx.JSON(http.StatusOK, gin.H{"message": "scratch card unlocked"})
	}
}

func HandleScratchCardList(db *sql.DB, repository *rewards.Queries) gin.HandlerFunc {
	//! this can be cached in redis
	//! when a new scratch card is added, the cache can be invalidated
	return func(ctx *gin.Context) {
		rows, err := repository.GetScratchCardRewards(ctx)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusOK, rows)
	}
}

func HandleScratchCardCallback(db *sql.DB, repository *rewards.Queries) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		var rewardsRequest rewardCallBackRequest
		if err := ctx.BindJSON(&rewardsRequest); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		var arg rewards.UpdateScratchCardRewardByOrderIdParams = rewards.UpdateScratchCardRewardByOrderIdParams{
			Status:  rewards.RewardStatus(rewardsRequest.Status),
			OrderID: rewardsRequest.OrderID,
		}

		repository.UpdateScratchCardRewardByOrderId(ctx, arg)

		ctx.JSON(http.StatusOK, gin.H{"message": "success"})
	}
}

func Router(db *sql.DB, repository *rewards.Queries, httpClient *http.Client) *gin.Engine {
	router := gin.New()
	router.GET("/", HandleRootRouter())
	router.POST("/v1/scratch-card/unlock", HandleScratchCardUnlock(db, repository))
	router.POST("/v1/scratch-card/list", HandleScratchCardList(db, repository))
	router.PUT("/v1/scratch-card/callback", HandleScratchCardCallback(db, repository))
	return router
}

func main() {
	// load env
	err := godotenv.Load(".env")
	if err != nil {
		panic(err)
	}

	// connect to the database
	connStr := os.Getenv("POSTGRES_URL")
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	// check the connection
	if err := db.Ping(); err != nil {
		panic(err)
	}

	repository := rewards.New(db)

	router := Router(db, repository, &http.Client{})
	router.Run("localhost:5252")
}
