package connect

// The command is intentionally kept in the source tree so generated protobuf
// and Connect bindings can be reproduced with the repository's toolchain.
// It requires protoc, protoc-gen-go, and protoc-gen-connect-go on PATH.
//go:generate protoc --proto_path=. --go_out=. --go_opt=paths=source_relative --connect-go_out=. --connect-go_opt=paths=source_relative,package_suffix= hamburger.proto
