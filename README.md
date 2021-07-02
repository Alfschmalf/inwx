# INWX for `libdns`

[![godoc reference](https://img.shields.io/badge/godoc-reference-blue.svg)](https://pkg.go.dev/github.com/libdns/inwx)


This package implements the libdns interfaces for the [INWX API](https://www.inwx.de/en/help/apidoc)

## Authenticating

To authenticate you need to supply your INWX Username and Password.

## Example

Here's a minimal example of how to get all DNS records for zone. See also: [provider_test.go](https://github.com/libdns/inwx/blob/master/provider_test.go)

```go
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/libdns/libdns/inwx"
)

func main() {
	user := os.Getenv("LIBDNS_INWX_USER")
	if user == "" {
		fmt.Printf("LIBDNS_INWX_USER not set\n")
		return
	}

	pass := os.Getenv("LIBDNS_INWX_USER")
	if pass == "" {
		fmt.Printf("LIBDNS_INWX_USER not set\n")
		return
	}

	zone := os.Getenv("LIBDNS_INWX_ZONE")
	if zone == "" {
		fmt.Printf("LIBDNS_INWX_ZONE not set\n")
		return
	}

	p := &inwx.Provider{
		AuthUsername: user,
		AuthPassword: pass,
	}

	records, err := p.GetRecords(context.WithTimeout(context.Background(), time.Duration(15*time.Second)), zone)
	if err != nil {
		fmt.Printf("Error: %s", err.Error())
		return
	}

	fmt.Println(records)
}

```

