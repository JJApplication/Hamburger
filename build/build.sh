#!/usr/bin/env bash
go clean
export GOEXPERIMENT=greenteagc
GC_FLAGS="-d=loopvar=2"
LD_FLAGS="-s -w -T 0x10000000"
go build -mod=mod --trimpath -gcflags="$GC_FLAGS" -ldflags="$LD_FLAGS" -tags=netgo -o hamburger .