package zonefile_test

import (
	"fmt"
	"github.com/stianwa/zonefile"
	"log"
	"os"
)

func ExampleParse() {
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
