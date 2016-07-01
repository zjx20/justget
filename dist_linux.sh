#!/bin/sh

if [ "$1" = "" ]; then
	echo "  Please specify the version parameter, usage:"
	echo "      ./$(basename $0) version"
	exit 1
fi

mkdir -p build/linux-amd64
GOOS=linux GOARCH=amd64 go build -o build/linux-amd64/justget
tar -C build/linux-amd64 -cjvf "justget-$1-linux-amd64.tar.bz2" justget
echo "Done"
