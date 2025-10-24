.PHONY: gen ghz pkg lint model deploy client
gen:
	go generate ./...

# Generate model
# Example: make model
model:
	goctl model mysql ddl -s docs/sql/mysql.sql -d model


ghz:
	ghz --insecure  --proto ./server/discovery/rpc/service/discovery.proto  --call discovery.Discovery.GetServiceIP 192.168.1.7:8085 -d '{"service_name":"test","client_key":"test"}' -c 10000 -n 100000

pkg:
	pushd client; fyne package -os darwin -icon assets/icon.jpg ; popd

# 构建linux arm64 二进制文件
build:
	if [ "$(shell uname -m)" = "aarch64" ] || [ "$(shell uname -m)" = "arm64" ]; then \
		GOOS=linux GOARCH=arm64 go build -o .deploy/bin/im cmd/main.go; \
	else \
		GOOS=linux GOARCH=amd64 go build -o .deploy/bin/im cmd/main.go; \
	fi

# 部署本地服务端
deploy:
	docker compose -f .deploy/base.yaml up -d
	docker compose -f .deploy/service.yaml up -d

# 启动客户端
client:
	pushd client; go run main.go; popd

# 推送多架构镜像
push:
	GOOS=linux GOARCH=amd64 go build -o .deploy/bin/im cmd/main.go;
	docker buildx build --platform linux/amd64 -t docker.io/comeonjy/im:latest-amd64 -f .deploy/Dockerfile .
	GOOS=linux GOARCH=arm64 go build -o .deploy/bin/im cmd/main.go;
	docker buildx build --platform linux/arm64 -t docker.io/comeonjy/im:latest-arm64 -f .deploy/Dockerfile .
	docker manifest create comeonjy/im:latest comeonjy/im:latest-amd64 comeonjy/im:latest-arm64
	docker manifest push comeonjy/im:latest

# 静态代码检查
# VSCode: "go.lintFlags": ["--config=./.golangci.yml"] 
lint:
	golangci-lint fmt
	golangci-lint run