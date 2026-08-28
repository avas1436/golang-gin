run:
	go run ./cmd/api

build:
	go build ./cmd/api

test:
	go test ./...

lint:
	golangci-lint run

fmt:
	goimports -w .

air:
	air

# دریافت یک فایل خلاصه از کل کد های محتویات پروژه
summary:
	npx repomix