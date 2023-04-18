CREATE TYPE reward_types AS ENUM ('r1', 'r2', 'r3');
CREATE TYPE reward_status AS ENUM ('pending', 'success', 'failed');

-- users -- this table keep the users
CREATE TABLE users (
  id                BIGSERIAL   PRIMARY KEY,
  name              text        NOT NULL,
  scratch_cards     integer     NOT NULL,
  created_at        timestamp   NOT NULL DEFAULT now(),
  updated_at        timestamp   NOT NULL DEFAULT now()
);

-- scratch_cards -- this table keep the scratch cards
CREATE TABLE scratch_cards (
  id                 BIGSERIAL       PRIMARY KEY,
  schedule           varchar(255)    NULL,
  max_cards          integer         NULL,
  max_cards_per_user integer         NULL,
  weight             integer         NOT NULL,
  reward_type        reward_types    NOT NULL,
  created_at         timestamp       NOT NULL DEFAULT now(),
  updated_at         timestamp       NOT NULL DEFAULT now()
);

-- scratch_cards_rewards -- this keep the history of rewards assigned to users with status
CREATE TABLE scratch_cards_rewards (
  id                 BIGSERIAL          PRIMARY KEY,
  scratch_card_id    BIGINT             NOT NULL,
  user_id            BIGINT             NOT NULL,
  order_id           varchar(255)       NOT NULL,
  status             reward_status      NOT NULL,
  created_at         timestamp          NOT NULL DEFAULT now(),
  updated_at         timestamp          NOT NULL DEFAULT now()
);