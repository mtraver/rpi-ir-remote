BUILD_ARMV6 := GOOS=linux GOARCH=arm GOARM=6 go build
BUILD_ARMV7 := GOOS=linux GOARCH=arm GOARM=7 go build

OUT_DIR := out

all: proto server cli makereq web

.PHONY: server
server:
	go build -o $(OUT_DIR)/$@ ./cmd/$@
	$(BUILD_ARMV7) -o $(OUT_DIR)/armv7/$@ ./cmd/$@
	$(BUILD_ARMV6) -o $(OUT_DIR)/armv6/$@ ./cmd/$@

.PHONY: cli
cli:
	go build -o $(OUT_DIR)/$@ ./cmd/$@
	$(BUILD_ARMV7) -o $(OUT_DIR)/armv7/$@ ./cmd/$@
	$(BUILD_ARMV6) -o $(OUT_DIR)/armv6/$@ ./cmd/$@

.PHONY: makereq
makereq:
	go build -o $(OUT_DIR)/$@ ./cmd/$@
	$(BUILD_ARMV7) -o $(OUT_DIR)/armv7/$@ ./cmd/$@
	$(BUILD_ARMV6) -o $(OUT_DIR)/armv6/$@ ./cmd/$@

.PHONY: web
web:
	go build -o $(OUT_DIR)/$@ ./$@

.PHONY: proto
proto:
	protoc --go_out=. irremotepb/irremote.proto
	protoc --go_out=. cmd/server/configpb/config.proto

clean:
	rm -rf $(OUT_DIR)
