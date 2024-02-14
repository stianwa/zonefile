# zonefile
[![Go Reference](https://pkg.go.dev/badge/github.com/stianwa/zonefile.svg)](https://pkg.go.dev/github.com/stianwa/zonefile) [![Go Report Card](https://goreportcard.com/badge/github.com/stianwa/zonefile)](https://goreportcard.com/report/github.com/stianwa/zonefile)

Package zonefile implements a DNS zonefile parser according to RFC1035
chapter 5.

Installation
------------

The recommended way to install zonefile

```
go get github.com/stianwa/zonefile
```

Example
-------

```go

package main

import (
	"fmt"
	"github.com/stianwa/zonefile"
	"log"
	"os"
)

func main() {
	content, err := os.ReadFile("somedir/example.com")
	if err != nil {
		log.Fatal(err)
	}

	ents, err := zonefile.Parse(content)
	if err != nil {
		log.Fatal(err)
	}

	for _, e := range ents {
		switch v := e.(type) {
		case *zonefile.RR:
			// Delete all record TTLs
			v.TTL = ""
		}
		fmt.Print(e)
	}
}
```

State
-----
The zonefile module is currently under development. Do not use for production.

License
-------

GPLv3, see [LICENSE.md](LICENSE.md)
