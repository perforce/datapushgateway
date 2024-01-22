# Makefile for datapushgateway

BINARY=datapushgateway

# These are the values we want to pass for VERSION and BUILD
# git tag 1.0.1
# git commit -am "One more change after the tags"
VERSION=`git describe --tags`
BUILD_DATE=`date +%FT%T%z`
USER=`git config user.email`
BRANCH=`git rev-parse --abbrev-ref HEAD`
REVISION=`git rev-parse --short HEAD`

# Setup the -ldflags option for go build here, interpolate the variable values
# Note the Version module is in a different git repo.
MODULE="github.com/perforce/p4prometheus"
LDFLAGS=-ldflags "-w -s -X ${MODULE}/version.Version=${VERSION} -X ${MODULE}/version.BuildDate=${BUILD_DATE} -X ${MODULE}/version.Branch=${BRANCH} -X ${MODULE}/version.Revision=${REVISION} -X ${MODULE}/version.BuildUser=${USER}"
LDFLAGS=-ldflags "-w -s -X ${MODULE}/version.Version=${VERSION} -X ${MODULE}/version.BuildDate=${BUILD_DATE} -X ${MODULE}/version.Branch=${BRANCH} -X ${MODULE}/version.Revision=${REVISION} -X ${MODULE}/version.BuildUser=${USER}"

# Builds the project
build:
	go build ${LDFLAGS}

# Builds distribution
dist:
	GOOS=darwin GOARCH=amd64 go build -o bin/${BINARY}-darwin-amd64 ${LDFLAGS}
	GOOS=darwin GOARCH=arm64 go build -o bin/${BINARY}-darwin-arm64 ${LDFLAGS}
	GOOS=linux GOARCH=amd64 go build -o bin/${BINARY}-linux-amd64 ${LDFLAGS}
	GOOS=windows GOARCH=amd64 go build -o bin/${BINARY}-windows-amd64.exe ${LDFLAGS}
	rm -f bin/${BINARY}*amd64*.gz bin/${BINARY}*arm64*.gz
	-chmod +x bin/${BINARY}*amd64* bin/${BINARY}*arm64* 
	gzip bin/${BINARY}*amd64* bin/${BINARY}*arm64*

# Cleans our project: deletes binaries
clean:
	if [ -f ${BINARY} ] ; then rm ${BINARY} ; fi

.PHONY: clean install