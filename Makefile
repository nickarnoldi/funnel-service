BUILD_DIR := ./build

targets := server client
recreate_build_env := go mod tidy && rm -rf $(BUILD_DIR) && mkdir -p $(BUILD_DIR)
local_build_target = `go build -o $(BUILD_DIR)/$(target) cmd/$(target)/main.go`
local_build := $(foreach target,$(targets),$(local_build_target))
prod_build_target = `GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(target) cmd/$(target)/main.go`
prod_build := $(foreach target,$(targets),$(prod_build_target))

local:
	@echo "Building for local..."
	@$(recreate_build_env)
	@[ -f ./.env ] && cp .env $(BUILD_DIR)/ || true
	@$(local_build)

prod:
	@echo "Building for production..."
	@$(recreate_build_env)
	@[ -f ./.env.prod ] && cp .env.prod $(BUILD_DIR)/.env || true
	@$(prod_build)
	@mv $(BUILD_DIR)/server $(BUILD_DIR)/application
	@cd $(BUILD_DIR) && zip -FSr ../application.zip .