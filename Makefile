include .env
export

migrate-up:
	migrate -path internal/db/migrations -database "$(DATABASE_URL_MIGRATE)" up

migrate-down:
	migrate -path internal/db/migrations -database "$(DATABASE_URL_MIGRATE)" down

migrate-status:
	migrate -path internal/db/migrations -database "$(DATABASE_URL_MIGRATE)" version

migrate-create:
	migrate create -ext sql -dir internal/db/migrations -seq $(name)

install-hooks:
	git config core.hooksPath .githooks