# cacheAside

[![Go](https://github.com/me-cs/cacheAside/workflows/Go/badge.svg?branch=main)](https://github.com/me-cs/cacheAside/actions)
[![codecov](https://codecov.io/gh/me-cs/cacheAside/branch/main/graph/badge.svg)](https://codecov.io/gh/me-cs/cacheAside)
[![Release](https://img.shields.io/github/v/release/me-cs/cacheAside.svg?style=flat-square)](https://github.com/me-cs/cacheAside)
[![Go Report Card](https://goreportcard.com/badge/github.com/me-cs/cacheAside)](https://goreportcard.com/report/github.com/me-cs/cacheAside)
[![Go Reference](https://pkg.go.dev/badge/github.com/me-cs/cacheAside.svg)](https://pkg.go.dev/github.com/me-cs/cacheAside)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
## Description
cacheAside is a generic cache-aside implementation

### Example use:

```go
package main

import (
	"fmt"
	
	"github.com/me-cs/cacheAside"
)

type UserInfo struct {
	Id   string
	Name string
}

func init() {
	cacheAside.Init(nil)
}

// you db fetch method

func DbUserInfo(id string) (*UserInfo, bool, error) {
	return &UserInfo{
		Id:   "1",
		Name: "tom",
	}, false, nil
}

// warp you dao fetch method

func DaoUserInfo(id string) (*UserInfo, error) {
	return cacheAside.Get(id, DbUserInfo)
}

// user in business code
func main() {
	u, err := DaoUserInfo("1")
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%+v\n", u)
}

```