.PHONY: gen ghz pkg lint model deploy
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


deploy:
	docker compose -f .deploy/base.yaml up -d

# 静态代码检查
# VSCode: "go.lintFlags": ["--config=./.golangci.yml"] 
lint:
	golangci-lint fmt
	golangci-lint run