package main

import (
	"github.com/Uitware/locreg/pkg/cmd"
	"log"
)

func main() {
	log.SetFlags(0) // remove timestamp from all log entries across packages
	cmd.Execute()
}
