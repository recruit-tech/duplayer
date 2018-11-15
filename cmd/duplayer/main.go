package main

import (
	"log"
	_ "net/http/pprof"

	"github.com/recruit-tech/duplayer"
)

func main() {
	log.SetPrefix("duplayer: ")
	log.SetFlags(0)
	if err := duplayer.Duplayer(); err != nil {
		log.Fatal(err)
	}
}
