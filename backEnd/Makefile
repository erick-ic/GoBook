docker:
	@rm gobook || true
	@GOOS=linux GOARCH=arm64 go build -tags=k8s -o gobook .
	@docker rmi -f flycash/gobook:v0.0.2
	@docker build -t flycash/gobook:v0.0.2 .