-- name: GetUser :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: DeductScratchCard :exec
UPDATE users SET scratch_cards = scratch_cards - 1
WHERE id = $1;

-- name: GetScratchCards :many
SELECT * FROM scratch_cards;

-- name: GetScratchCardRewards :many
SELECT scr.id, scr.user_id, scr.scratch_card_id, scr.status, sc.reward_type, u.name, u.scratch_cards
FROM scratch_cards_rewards AS scr
JOIN scratch_cards AS sc ON sc.id = scr.scratch_card_id
JOIN users AS u ON u.id = scr.user_id;

-- name: CreateScratchCardReward :one
INSERT INTO scratch_cards_rewards (scratch_card_id, user_id, order_id ,status)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: GetScratchCardReward :one
SELECT * FROM scratch_cards_rewards
WHERE id = $1 LIMIT 1;

-- name: UpdateScratchCardReward :exec
UPDATE scratch_cards_rewards SET status = $2
WHERE id = $1;

-- name: UpdateScratchCardRewardByOrderId :exec
UPDATE scratch_cards_rewards SET status = $2
WHERE order_id = $1;

-- name: GetUnlockedScratchCardRewardCount :one
SELECT COUNT(*) FROM scratch_cards_rewards
WHERE scratch_card_id = $1 AND status = 'success';

-- name: GetUnlockedScratchCardRewardCountByUser :one
SELECT COUNT(*) FROM scratch_cards_rewards
WHERE scratch_card_id = $1 AND user_id = $2 AND status = 'success';