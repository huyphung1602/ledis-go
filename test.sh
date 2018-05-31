#!/bin/bash
# @Author: zealotnt
# @Date:   2018-05-31 23:58:51

go test ./... -coverprofile ledis.coverprofile
go tool cover -func=ledis.coverprofile
go tool cover -html=ledis.coverprofile -o ledis.coverprofile.html
