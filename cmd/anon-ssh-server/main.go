package main

import (
	"fmt"
	"github.com/gliderlabs/ssh"
	xcssh "golang.org/x/crypto/ssh"
	"io"
	"log"
	"os/exec"
	"time"
)

func main() {
	// ssh-keygen -m PEM -f hostkey.pem
	server := &ssh.Server{
		Addr:        ":1966",
		IdleTimeout: 1 * time.Second,
	}

	server.Handle(func(s ssh.Session) {
		pubkey := xcssh.FingerprintSHA256(s.PublicKey())

		if len(s.Command()) == 0 {
			io.WriteString(s, fmt.Sprintf("Welcome %s\n", s.User()))
			io.WriteString(s, fmt.Sprintf("Your public key is %s\n", pubkey))
			io.WriteString(s, fmt.Sprintf("Your environment: %v\n", s.Environ()))
			io.WriteString(s, fmt.Sprintf("%+v\n", s))
			s.Exit(0)
			return
		}

		// FIXME huge security issue running arbitrary commands!
		fmt.Printf("Executing command: %v\n", s.Command())
		cmd := exec.Command(s.Command()[0], s.Command()[1:]...)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			log.Printf("ERROR: %s\n", err)
		}
		stdin, err := cmd.StdinPipe()
		if err != nil {
			log.Printf("ERROR: %s\n", err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			log.Printf("ERROR: %s\n", err)
		}

		if err := cmd.Start(); err != nil {
			log.Printf("ERROR: %s\n", err)
		}

		go func() {
			defer stdout.Close()
			io.Copy(s, stdout)
		}()
		go func() {
			defer stdin.Close()
			io.Copy(stdin, s)
		}()
		go func() {
			defer stderr.Close()
			io.Copy(s.Stderr(), stderr)
		}()

		if err := cmd.Wait(); err != nil {
			log.Printf("ERROR: %s\n", err)
		}

		s.Exit(cmd.ProcessState.ExitCode())
	})
	server.SetOption(ssh.PublicKeyAuth(func(ctx ssh.Context, key ssh.PublicKey) bool {
		// All public keys are allowed
		return true
	}))
	server.SetOption(ssh.PasswordAuth(func(ctx ssh.Context, pass string) bool {
		// Passwords are never correct
		return false
	}))
	server.SetOption(ssh.HostKeyFile("hostkey.pem"))
	log.Fatal(server.ListenAndServe())
}
