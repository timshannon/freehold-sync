all:
	$(shell go-bindata web/... && go build)
