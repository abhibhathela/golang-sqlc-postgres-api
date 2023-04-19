PID_FILE := './.pid'
POSTGRESQL_URL='postgres://postgres:root@localhost:5432/rewards?sslmode=disable'

migrate-up:
	@migrate -database ${POSTGRESQL_URL} -path migrations -verbose up

migrate-down:
	@migrate -database ${POSTGRESQL_URL} -path migrations -verbose down

run-restart: ## restart the API server
	@pkill -P `cat $(PID_FILE)` || true
	@printf '%*s\n' "80" '' | tr ' ' -
	@echo "Source file changed. Restarting server..."
	@go run cmd/scratch-card-server/main.go & echo $$! > $(PID_FILE)
	@printf '%*s\n' "80" '' | tr ' ' -

run-live: ## run the API server with live reload support (requires fswatch)
	@go run cmd/scratch-card-server/main.go & echo $$! > $(PID_FILE)
	@fswatch -x -o --event Created --event Updated --event Renamed -r internal pkg cmd config | xargs -n1 -I {} make run-restart
