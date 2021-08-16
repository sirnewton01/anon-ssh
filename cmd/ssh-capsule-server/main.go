package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/gliderlabs/ssh"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

var CLI struct {
	ListenAddress string        `name:"listen-address" default:":1966"`
	IdleTimeout   time.Duration `name:"idle-timeout" default:"10s"`
	HostKey       string        `arg name:"hostkey" help:"Host PEM key to use for this server. If the file doesn't exist then one will be generated." type:"path" required:""`

	DefaultCapsule string `arg name:"default-capsule" help:"Location of the configuration of the default capsule. If the directory doesn't exist a default capsule will be generated there." type:"path" required:""`

	Capsule []string `name:"capsule" help:"The location of an extra capsule that will be virtually hosted with this server." type:"path"`
}

const COMMAND_LIST_TEMPLATE = `# The following is a list of commands templates that will be permitted on this server
# It is important to choose a minimal set since anyone with access to your
# server can run these without any authentication using this server.
# Note the special <path> tokens represent paths relative to the capsule content
# directory.
#
# Read-only commands:
#cat <path>
#gemini <path>
#scp -f <path>
#git-upload-pack <path>
`

const GROUP_TEMPLATE = `# This is a list of public keys and additional groups
# for the key. Additional groups can give access to additional commands.
#
# <key type> <key> group1 group2 ...
# ssh-rsa ADKFJSKLDFJ... admin site-admin
# 
# Additional commands for a group are listed in a file commands-groupname
# (eg. commands-admin and commands-site-admin from above example) with the
# same format as the commands file.
`

// TODO make this much more comprehensive while being safe
var PATH_REGEX = regexp.MustCompile("^[a-zA-Z0-9\\-\\./_]+$")

func pathMatch(path string, capsuleContentPath string) string {
	path = filepath.Clean(path)
	if !filepath.IsAbs(path) {
		path = "/" + path
	}
	for strings.HasPrefix(path, "/..") {
		path = path[3:]
	}

	return filepath.Join(capsuleContentPath, path)
}

func commandMatch(cmdTemplate []string, cmd []string, capsuleContentPath string) []string {
	for i := range cmdTemplate {
		if cmdTemplate[i] == "<path>" {
			// Special handling for paths
			if !PATH_REGEX.MatchString(cmd[i]) {
				return nil
			}

			matchedPath := pathMatch(cmd[i], capsuleContentPath)

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

func validateCommand(cmd []string, capsulePath string, publicKey string) []string {
	cmdFiles := []string{"commands"}

	// Consult the group file if available to see if there are any
	//  additional commands that this user can run.
	groupFile, err := os.Open(filepath.Join(capsulePath, "group"))
	if err == nil {
		defer groupFile.Close()
		s := bufio.NewScanner(groupFile)
		for s.Scan() {
			l := s.Text()
			if len(l) > len(publicKey) && strings.HasPrefix(l, publicKey) {
				groups := strings.Split(l[len(publicKey):], " ")
				for _, g := range groups {
					if g != "" {
						cmdFiles = append(cmdFiles, fmt.Sprintf("commands-%s", g))
					}
				}
				break
			}
		}
	}

	for _, cf := range cmdFiles {
		cmdFile, err := os.Open(filepath.Join(capsulePath, cf))
		if err != nil {
			log.Printf("ERROR %s\n", err)
			continue
		}
		defer cmdFile.Close()

		capsuleContentPath := filepath.Join(capsulePath, "content")

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

			cmdMatch := commandMatch(cmdTemplate, cmd, capsuleContentPath)

			if cmdMatch != nil && len(cmdMatch) > 0 {
				return cmdMatch
			}
		}
	}

	return nil
}

func isCapsuleForHost(capsulePath string, host string) bool {
	hostFile, err := os.Open(filepath.Join(capsulePath, "host"))
	if err != nil {
		log.Printf("ERROR: %s\n", err)
		return false
	}

	scanner := bufio.NewScanner(hostFile)
	for scanner.Scan() {
		if host == scanner.Text() {
			return true
		}
	}

	return false
}

func main() {
	kong.Parse(&CLI)

	// As a convenience, let's generate the files if they don't exist
	if _, err := os.Stat(CLI.HostKey); os.IsNotExist(err) {
		log.Printf("Generating host-key: %s\n", CLI.HostKey)
		kgc := exec.Command("ssh-keygen", "-m", "PEM", "-f", CLI.HostKey, "-N", "")
		err := kgc.Run()
		if err != nil {
			log.Printf("ERROR: %s\n", err)
			os.Exit(1)
		}
	}

	if _, err := os.Stat(CLI.DefaultCapsule); os.IsNotExist(err) {
		log.Printf("Generating default capsule: %s\n", CLI.DefaultCapsule)
		err := os.Mkdir(CLI.DefaultCapsule, 0700)
		if err != nil {
			log.Printf("ERROR: %s\n", err)
			os.Exit(1)
		}

		// We can't guess what the hostname is supposed to be
		//  so we'll generate it in case they want to fill it in
		//  later.
		hf, err := os.Create(filepath.Join(CLI.DefaultCapsule, "host"))
		if err != nil {
			log.Printf("ERROR: %s\n", err)
			os.Exit(1)
		}
		hf.WriteString("")
		hf.Close()

		cf, err := os.Create(filepath.Join(CLI.DefaultCapsule, "commands"))
		if err != nil {
			log.Printf("ERROR: %s\n", err)
			os.Exit(1)
		}
		cf.WriteString(COMMAND_LIST_TEMPLATE)
		cf.Close()

		err = os.Mkdir(filepath.Join(CLI.DefaultCapsule, "content"), 0700)
		if err != nil {
			log.Printf("ERROR: %s\n", err)
			os.Exit(1)
		}

		idxf, err := os.Create(filepath.Join(CLI.DefaultCapsule, "content", "index.gmi"))
		if err != nil {
			log.Printf("ERROR: %s\n", err)
			os.Exit(1)
		}
		idxf.WriteString("Welcome to my capsule!")
		idxf.Close()

		gf, err := os.Create(filepath.Join(CLI.DefaultCapsule, "group"))
		if err != nil {
			log.Printf("ERROR: %s\n", err)
			os.Exit(1)
		}
		gf.WriteString(GROUP_TEMPLATE)
		gf.Close()
	}

	server := &ssh.Server{
		Addr:        CLI.ListenAddress,
		IdleTimeout: CLI.IdleTimeout,
	}

	server.Handle(func(s ssh.Session) {
		host := "default"
		for _, e := range s.Environ() {
			if strings.HasPrefix(e, "HOST=") && len(e) > 5 {
				host = e[5:]
				if host == "" {
					host = "default"
				}
				break
			}
		}
		pubkey := s.PublicKey().Type() + " " + base64.StdEncoding.EncodeToString(s.PublicKey().Marshal())

		if len(s.Command()) == 0 {
			io.WriteString(s, fmt.Sprintf("Welcome to %s, user %s\n", host, s.User()))
			io.WriteString(s, fmt.Sprintf("Your public key is %s\n", pubkey))
			io.WriteString(s, fmt.Sprintf("Your environment: %v\n", s.Environ()))
			// Some friendly warnings in case the client seems to be set up incorrectly
			if s.User() != "capsule" {
				io.WriteString(s.Stderr(), fmt.Sprintf("WARNING: your username is not configured as 'capsule' for this host. You might want to change your SSH settings to protect against tracking\n"))
			}
			if host == "default" {
				io.WriteString(s.Stderr(), fmt.Sprintf("WARNING: you HOST environment variable is not set to the hostname that you are connecting. You might want to change your SSH settings to send the hostname so that you can take advantage of virtual hosting in the future.\n"))

			}
			s.Exit(0)
			log.Printf("Greeting sent\n")
			return
		}

		log.Printf("Command requested: %v\n", s.Command())

		capsule := CLI.DefaultCapsule
		if host != "default" && !isCapsuleForHost(capsule, host) {
			// Let's try one of the alternate capsules
			//  for a match.
			for _, c := range CLI.Capsule {
				if isCapsuleForHost(c, host) {
					capsule = c
					break
				}
			}
		}

		cmd := validateCommand(s.Command(), capsule, pubkey)

		if len(cmd) == 0 {
			log.Printf("Command blocked: %v\n", s.Command())
			io.WriteString(s, "Command not found\n")
			s.Exit(127)
			return
		}

		log.Printf("Executing command: %v\n", cmd)
		c := exec.Command(cmd[0], cmd[1:]...)
		c.Dir = capsule
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
	server.SetOption(ssh.HostKeyFile(CLI.HostKey))
	log.Printf("Server started on addresss %s", CLI.ListenAddress)
	log.Fatal(server.ListenAndServe())
}
