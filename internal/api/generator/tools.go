//go:build tools

// this file is here so that `go mod download` will download the modules needed to build the project
package main

import (
	_ "github.com/golang/protobuf/protoc-gen-go"
)
