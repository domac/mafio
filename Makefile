.PHONY: all clean
# 被编译的文件
BUILDFILE=main.go
# 编译后的静态链接库文件名称
TARGETNAME=mafio
# GOOS为目标主机系统 
# mac os : "darwin"
# linux  : "linux"
# windows: "windows"
GOOS=linux
# GOARCH为目标主机CPU架构, 默认为amd64 
GOARCH=amd64

all: format test build clean

test:
	go test -v . 

format:
	gofmt -w .

build:
	mkdir -p releases
	cp agent.sh releases/agent.sh
	cp config/agent.conf releases/agent.conf
	GOOS=$(GOOS) GOARCH=$(GOARCH) go build -v -o releases/$(TARGETNAME) $(BUILDFILE)
clean:
	go clean -i