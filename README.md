controlmaster
-------------
[![GoDoc](http://godoc.org/github.com/bored-engineer/controlmaster?status.svg)](http://godoc.org/github.com/bored-engineer/controlmaster)

An implementation of the OpenSSH ControlMaster [protocol](https://github.com/openssh/openssh-portable/blob/master/PROTOCOL.mux) for Golang

NOTE: this relies on a _fork_ of golang.org/x/crypto/ssh with support for custom transports ([github.com/bored-engineer/ssh](https://github.com/bored-engineer/ssh))

usage
-----
```go
package main

import (
	"log"

	"github.com/bored-engineer/controlmaster"
)

func main() {
	client, err := controlmaster.Dial("unix", "PATH_TO_CONTROLMASTER")
	if err != nil {
		log.Fatal(err)
	}
	session, err := client.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	defer session.Close()
	out, err := session.CombinedOutput("id")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(out))
}
```
