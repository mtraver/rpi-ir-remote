BUILD_ARMV6 := GOOS=linux GOARCH=arm GOARM=6 go build
BUILD_ARMV7 := GOOS=linux GOARCH=arm GOARM=7 go build

OUT_DIR := out

all: server cli

.PHONY: server
server:
	go build -o $(OUT_DIR)/$@ ./cmd/$@/main.go
	$(BUILD_ARMV7) -o $(OUT_DIR)/armv7/$@ ./cmd/$@/main.go
	$(BUILD_ARMV6) -o $(OUT_DIR)/armv6/$@ ./cmd/$@/main.go

.PHONY: cli
cli:
	go build -o $(OUT_DIR)/$@ ./cmd/$@/main.go
	$(BUILD_ARMV7) -o $(OUT_DIR)/armv7/$@ ./cmd/$@/main.go
	$(BUILD_ARMV6) -o $(OUT_DIR)/armv6/$@ ./cmd/$@/main.go

clean:
	rm -rf $(OUT_DIR)
