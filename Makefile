APP_NAME=weaveset

api:
	@go build -o bin/$(APP_NAME)-apiserver ./cmd/apiserver
	@hugo -s site --minify
	@bin/$(APP_NAME)-apiserver

worker:
	@go build -o bin/$(APP_NAME)-worker ./cmd/worker
	@bin/$(APP_NAME)-worker

simple:
	@go build -o bin/$(APP_NAME)-simple ./cmd/simple
	@bin/$(APP_NAME)-simple

test:
	@go test -v -count=1 ./...

docker-buildsimple:
	@docker build -t weaveset-simple -f infra/docker/simple.Dockerfile .

docker-buildall:
	@docker build -t weaveset-apiserver -f infra/docker/apiserver.Dockerfile .
	@docker build -t weaveset-worker -f infra/docker/worker.Dockerfile .
	@docker build -t weaveset-rebuild -f infra/docker/rebuild.Dockerfile .

reset-site:
	@rm -rf site/content/list
	@rm -rf site/public
	@rm -rf site/resources
