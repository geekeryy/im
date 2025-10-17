.PHONY: gen ghz pkg lint
gen:
	go generate ./...


ghz:
	ghz --insecure  --proto ./server/discovery/rpc/service/discovery.proto  --call discovery.Discovery.GetServiceIP 192.168.1.7:8085 -d '{"service_name":"test","client_key":"test"}' -c 10000 -n 100000

pkg:
	pushd client; fyne package -os darwin -icon icon.jpg ; popd


# 静态代码检查
# VSCode: "go.lintFlags": ["--config=./.golangci.yml"] 
lint:
	golangci-lint fmt
	golangci-lint run