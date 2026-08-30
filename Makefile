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

# Scaffold de fichiers avec le bon package.
#   make scaffold name=toto
#     -> internal/handler/toto.handler.go       (package handler)
#        internal/repository/toto.repository.go (package repository)
#        internal/models/toto.go                (package models)
#        internal/routes/toto.routes.go         (package routes)
#   make scaffold name=toto layers=handler,model   (restreindre aux couches voulues)
layers ?= handler,repository,model,routes
BASH ?= C:/Program Files/Git/bin/bash.exe

scaffold:
	@"$(BASH)" scripts/scaffold.sh "$(name)" "$(layers)"

.PHONY: migrate-up migrate-down migrate-status migrate-create install-hooks scaffold