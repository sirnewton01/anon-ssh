package main

import (
	"bufio"
	"fmt"
	"github.com/gliderlabs/ssh"
	xcssh "golang.org/x/crypto/ssh"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// TODO make this much more comprehensive while being safe
var PATH_REGEX = regexp.MustCompile("^[a-zA-Z0-9\\-\\./_]+$")

func pathMatch(path string) string {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		path = "/" + path
	}

	pbFile, err := os.Open("path-bindings")
	if err != nil {
		log.Printf("ERROR opening path-bindings: %s\n", err)
		return ""
	}
	defer pbFile.Close()

	scanner := bufio.NewScanner(pbFile)

	matchKey := ""
	matchVal := ""

	for scanner.Scan() {
		l := scanner.Text()
		// Skip comments
		if strings.HasPrefix(l, "#") {
			continue
		}

		mp := strings.Split(l, ":")
		if len(mp) != 2 {
			log.Printf("Invalid entry in path-bindings: %s\n", l)
			continue
		}

		k, v := mp[0], mp[1]

		// This is a perfect match, return
		//  with the value
		if path == k {
			return v
		}

		// This must match as a full path segment
		if !strings.HasSuffix(k, "/") {
			k = k + "/"
		}

		if strings.HasPrefix(path, k) {
			if len(k) > len(matchKey) {
				matchKey = k
				matchVal = v
			}
		}
	}

	if matchKey != "" {
		return filepath.Join(matchVal, path[len(matchKey):])
	}

	log.Printf("Unrecognized path: %s\n", path)
	return ""
}

func commandMatch(cmdTemplate []string, cmd []string) []string {
	for i := range cmdTemplate {
		if cmdTemplate[i] == "<path>" {
			// Special handling for paths
			if !PATH_REGEX.MatchString(cmd[i]) {
				return nil
			}

			matchedPath := pathMatch(cmd[i])

			if matchedPath == "" {
				return nil
			}

			cmdTemplate[i] = matchedPath
		} else if cmdTemplate[i] != cmd[i] {
			return nil
		}
	}

	return cmdTemplate
}

func validateCommand(cmd []string) []string {
	cmdFile, err := os.Open("commands")
	if err != nil {
		log.Printf("ERROR %s\n", err)
		return nil
	}
	defer cmdFile.Close()

	scanner := bufio.NewScanner(cmdFile)
	for scanner.Scan() {
		l := scanner.Text()

		if len(l) == 0 || strings.HasPrefix(l, "#") {
			continue
		}

		cmdTemplate := strings.Split(l, " ")

		if len(cmdTemplate) != len(cmd) {
			continue
		}

		cmdMatch := commandMatch(cmdTemplate, cmd)

		if cmdMatch != nil && len(cmdMatch) > 0 {
			return cmdMatch
		}
	}

	return nil
}

func main() {
	// ssh-keygen -m PEM -f hostkey.pem
	server := &ssh.Server{
		Addr:        ":1966",
		IdleTimeout: 10 * time.Second,
	}

	server.Handle(func(s ssh.Session) {
		pubkey := xcssh.FingerprintSHA256(s.PublicKey())

		if len(s.Command()) == 0 {
			log.Printf("Greeting sent\n")
			io.WriteString(s, fmt.Sprintf("Welcome %s\n", s.User()))
			io.WriteString(s, fmt.Sprintf("Your public key is %s\n", pubkey))
			io.WriteString(s, fmt.Sprintf("Your environment: %v\n", s.Environ()))
			s.Exit(0)
			return
		}

		log.Printf("Command requested: %v\n", s.Command())

		cmd := validateCommand(s.Command())

		if len(cmd) == 0 {
			log.Printf("Command blocked: %v\n", s.Command())
			io.WriteString(s, "Command not found\n")
			s.Exit(127)
			return
		}

		log.Printf("Executing command: %v\n", cmd)
		c := exec.Command(cmd[0], cmd[1:]...)
		stdout, err := c.StdoutPipe()
		if err != nil {
			log.Printf("ERROR: %s\n", err)
		}
		stdin, err := c.StdinPipe()
		if err != nil {
			log.Printf("ERROR: %s\n", err)
		}
		stderr, err := c.StderrPipe()
		if err != nil {
			log.Printf("ERROR: %s\n", err)
		}

		if err := c.Start(); err != nil {
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

		if err := c.Wait(); err != nil {
			log.Printf("ERROR: %s\n", err)
		}

		s.Exit(c.ProcessState.ExitCode())
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
