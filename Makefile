
PROJECTNAME=$(shell basename "$(PWD)")

BINARY_NAME=pg_gen

# Go переменные.
GOBASE=$(shell pwd)
GOPATH=$(GOBASE)/vendor:$(GOBASE):/home/azer/code/golang  #Вы можете удалить или изменить путь после двоеточия.
GOBIN=$(GOBASE)/bin
GOFILES=$(wildcard *.go)

# Make пишет работу в консоль Linux. Сделаем его silent.
MAKEFLAGS += --silent

exec:
	@GOPATH=$(GOPATH) GOBIN=$(GOBIN) $(run)

build:
 GOARCH=arm64 GOOS=darwin go build -o bin/${BINARY_NAME}-darwin main.go