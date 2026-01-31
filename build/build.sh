#!/usr/bin/env bash
go clean
export GOEXPERIMENT=greenteagc
Version=0.1.0
BuildHash=$(git rev-parse HEAD)
GC_FLAGS="-d=loopvar=2"
LD_FLAGS="-s -w -T 0x10000000 -X main.Version=$Version -X main.BuildHash=$BuildHash"
go build -mod=mod --trimpath -gcflags="$GC_FLAGS" -ldflags="$LD_FLAGS" -tags=netgo -o hamburger .