package main

import (
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/abhishheck/gamezop-task/pkg/helpers"
	"github.com/abhishheck/gamezop-task/pkg/integrations"
	"github.com/abhishheck/gamezop-task/pkg/rewards"
	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

type unlockScratchCardRequest struct {
	UserId int64 `json:"user_id" binding:"required"`
}

func main() {

	// connect to the database
	connStr := "postgres://postgres:root@localhost/test?sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	// check the connection
	if err := db.Ping(); err != nil {
		panic(err)
	}

	router := gin.Default()

	rewardsRepo := rewards.New(db)

	router.GET("/", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{"message": "hello gin, grant my wish"})
	})

	router.POST("/v1/scratch-card/unlock", func(ctx *gin.Context) {
		// take user_id from the json body

		var unlockScratchCardRequest unlockScratchCardRequest
		if err := ctx.BindJSON(&unlockScratchCardRequest); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		userId := unlockScratchCardRequest.UserId

		// validate user has scratch cards to unlock
		user, err := rewardsRepo.GetUser(ctx, userId)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		if user.ScratchCards == 0 {
			ctx.JSON(http.StatusBadRequest, gin.H{"error": "user has no scratch cards to unlock"})
			return
		}

		rewardsRows, err := rewardsRepo.GetScratchCards(ctx)

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
				_, err := helpers.IsValidDateToUnlockReward(sc.Schedule.String)
				if err != nil {
					fmt.Println("error while validating the schedule", err)
					break
				}
			}

			if sc.MaxCards.Valid {
				//! find how many are unlocked
				redeemedCardsCount, err := rewardsRepo.GetUnlockedScratchCardRewardCount(ctx, sc.ID)
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

				redeemedCardsCount, err := rewardsRepo.GetUnlockedScratchCardRewardCountByUser(ctx, args)
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

		response, err := integrations.UnlockScratchCard(selectedItem.RewardType)

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

		qtx := rewardsRepo.WithTx(tx)

		err = qtx.DeductScratchCard(ctx, user.ID)
		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		fmt.Println(rewards.RewardStatus(response.Data.Status))

		// rStatus := rewards.RewardStatus(response.Data.Status)
		// rStatus.Scan(&response.Data.Status)

		// insert into the scratch card rewards table
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
			go integrations.PoolPaymentStatus(newSc.ID, response.Data.OrderId.String(), ctx, rewardsRepo)
		}

		ctx.JSON(http.StatusOK, gin.H{"message": "scratch card unlocked"})
	})

	router.POST("/v1/scratch-card/list", func(ctx *gin.Context) {
		rows, err := rewardsRepo.GetScratchCardRewards(ctx)

		if err != nil {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		ctx.JSON(http.StatusOK, rows)
	})

	router.GET("/v1/scratch-card/callback", func(ctx *gin.Context) {

	})

	router.Run("localhost:5252")
}
