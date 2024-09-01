compile:
	protoc api/v1/*.proto \
	--go_out=. \
	--go_opt=paths=source_relatie \
	--proto_path=.
test:
	go test -race ./...