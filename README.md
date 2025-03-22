# api-server

go mod init api-server
go mod tidy


# Command to built multi-platform docker image and push to docker hub
docker buildx build --platform linux/amd64,linux/arm64 -t <dockerhub-username>/api-server:latest --push .
