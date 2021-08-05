package main

import (
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/sirnewton01/ssh-capsules/pkg/setup"
	"os"
)

var CLI struct {
	Host    string `arg name:"host" help:"The name of the capsule host to get set up with SSH and a cryptographic key." required:""`
}

func main() {
	kong.Parse(&CLI)
	host := CLI.Host

	err := setup.AssertCapsuleConfig(host)
	if err != nil {
		fmt.Fprintf(os.Stderr, "An error occurred: %s\n", err)
		os.Exit(1)
	}
	fmt.Printf("capsule@%s\n", host)
}
