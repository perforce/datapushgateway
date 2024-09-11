BINARY=datapushgateway

# These are the values we want to pass for VERSION and BUILD
VERSION=`git describe --tags`
BUILD_DATE=`date +%FT%T%z`
USER=`git config user.email`
BRANCH=`git rev-parse --abbrev-ref HEAD`
REVISION=`git rev-parse --short HEAD`

# Module for versioning
MODULE="github.com/perforce/p4prometheus"
LDFLAGS=-ldflags "-w -s -X ${MODULE}/version.Version=${VERSION} -X ${MODULE}/version.BuildDate=${BUILD_DATE} -X ${MODULE}/version.Branch=${BRANCH} -X ${MODULE}/version.Revision=${REVISION} -X ${MODULE}/version.BuildUser=${USER}"

# Builds the project
build:
	go build ${LDFLAGS}

# Builds distribution and setup directory
dist: clean
	GOOS=linux GOARCH=amd64 go build -o bin/${BINARY}-linux-amd64 ${LDFLAGS}
	GOOS=darwin GOARCH=amd64 go build -o bin/${BINARY}-darwin-amd64 ${LDFLAGS}
	GOOS=darwin GOARCH=arm64 go build -o bin/${BINARY}-darwin-arm64 ${LDFLAGS}
	GOOS=windows GOARCH=amd64 go build -o bin/${BINARY}-windows-amd64.exe ${LDFLAGS}
	rm -f bin/${BINARY}*amd64*.gz bin/${BINARY}*arm64*.gz
	-chmod +x bin/${BINARY}*amd64* bin/${BINARY}*arm64*
# gzip bin/${BINARY}*amd64* bin/${BINARY}*arm64*

	# Create setup directory
	mkdir -p setup/datapushgateway

	# Copy necessary files into setup/datapushgateway
	cp auth.yaml config.yaml setup.sh p4stuff/.p4config setup/datapushgateway/
	cp bin/${BINARY}-linux-amd64 setup/datapushgateway/${BINARY}

	# Create the packages directory
	mkdir -p packages

	# Create the tarball in the packages directory
	tar -czvf packages/datapushgateway.tar.gz -C setup datapushgateway

# Cleans the project: deletes binaries, setup directory, and tarball
clean:
	rm -rf bin setup packages
