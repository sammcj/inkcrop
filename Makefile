.DEFAULT_GOAL := build

build:
	@echo "Building inkcrop"
	@mkdir -p dist
	@go build -o dist/inkcrop
	@chmod +x dist/inkcrop
	@echo "Build complete"
	@ls -la dist/*

install:
	@echo "Installing inkcrop"
	@cp dist/inkcrop /usr/local/bin/inkcrop
	@echo "Install complete"

build-container:
	@echo "Building inkcrop container image"
	@docker build -t sammcj/inkcrop:dev .
	@echo "Container image build complete"

clean:
	rm -rf dist/*
	@echo "Clean complete"

serve:
	@echo "Starting server"
	./dist/inkcrop -daemon