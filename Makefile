.PHONY: dev-api dev-web build build-web build-api build-admin admin run test fmt vet

dev-api:
	cd server && go run . -addr :8080

dev-web:
	cd web && npm run dev

build: build-web build-api build-admin

build-web:
	cd web && npm ci && npm run build

build-api:
	mkdir -p bin
	cd server && go build -o ../bin/memebank .

build-admin:
	mkdir -p bin
	cd server && go build -o ../bin/memebank-admin ./cmd/admin

admin:
	cd server && go run ./cmd/admin $(ARGS)

run:
	./bin/memebank -addr :8080 -data ./data -static ./web/dist

test:
	cd server && go test ./...
	cd web && npm run typecheck

fmt:
	cd server && gofmt -w .

vet:
	cd server && go vet ./...
