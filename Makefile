BINARY_NAME=wfts
.DEFAULT_GOAL=index

build: test
	go build -o ./.bin/${BINARY_NAME} ./cmd/app/main.go

test:
	@go test ./... -v -count=1 -parallel=1 2>&1 | grep -v "no test files" || true

index: build
	./.bin/${BINARY_NAME}

panic-test: build
	./.bin/${BINARY_NAME} > ./logs/panic.txt 2>&1

index-gui: build
	./.bin/${BINARY_NAME} --gui

search: build
	./.bin/${BINARY_NAME} -i

python-run:
	./internal/utils/semantic_embeddings/.venv/bin/python ./internal/utils/semantic_embeddings/app.py