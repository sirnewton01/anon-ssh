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
	"text/template"
	"time"
)

var CLI struct {
	ListenAddress string        `name:"listen-address" default:":1966"`
	IdleTimeout   time.Duration `name:"idle-timeout" default:"10s"`
	HostKey       string        `arg name:"hostkey" help:"Host PEM key to use for this server. If the file doesn't exist then one will be generated." type:"path" required:"" env:"HOST_KEY_LOC"`

	DefaultCapsule string `arg name:"default-capsule" help:"Location of the configuration of the default capsule. If the directory doesn't exist a default capsule will be generated there." type:"path" required:"" env:"CAPSULE_LOC"`

	Capsule []string `name:"capsule" help:"The location of an extra capsule that will be virtually hosted with this server." type:"path"`
}

const COMMAND_LIST_TEMPLATE = `# The following is a list of commands templates that will be permitted on this server
# It is important to choose a minimal set since anyone with access to your
# server can run these without any authentication using this server.
# Note the special <path> tokens represent paths relative to the capsule content
# directory.
#
# Default command when the user doesn't provide one.
tpl
#
# Read-only commands:
#ls <path>
#tpl <path>
#cat <path>
#wc -c <path>
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
# same format as the commands file. In these files you can put the commands
# available only to only those groups.
`

const MAIN_GMI_TEMPLATE = `# {{ .env.HOST }} (This Capsule)
Welcome capsule user!

Some information we can see about yourself:

## IDENT (Public Encryption Key)
{{ .env.IDENT }}

{{ if index .env "LANG" }}## LANG (Preferred Language)
{{ .env.LANG }}{{ end }}
{{ if index .env "TZ" }}## TZ (Preferred Timezone)
{{ .env.TZ }}{{ end }}
If you are the owner of this capsule you can change this welcome page to guide visitors to the capabilities and areas of your capsule. This page is in contents/main.gmi of your capsule directory and can be evaluated using the tpl built-in command.

You can add new files to your capsule to the contents directory, but first you will need to provide a command that will allow them to view your files. If you uncomment the "cat <path>" command in the commands file then users can run that command in your capsule with the file path and see them easily if they are text. This welcome document can be a place where you put a list of files to download.

=> ssh capsule@{{ .env.HOST }} cat about.gmi
=> ssh capsule@{{ .env.HOST }} cat contacts.gmi

Files that are meant to be downloaded to a visitor's computer can be done using scp. Enable that in the capsule commands file. You can provide examples for them to change and suit their needs.

` + "```\nscp capsule@{{ .env.HOST }}:/site_backup.tar.gz .\n```" + `

Instead of curating a complete list here of all the files that are in your capsule you can provide the ls command. It depends on the nature of your capsule. It's best to start with only the commands that are needed. Once the "ls <path>" command is enabled then visitors can look around the capsule like this:

` + "```\nssh capsule@{{ .env.HOST }} ls /\n```" + `
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

			// No command is provided and this is the default
			if len(cmdTemplate) == 1 && len(cmd) == 0 {
				return cmdTemplate
			}

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

		idxf, err := os.Create(filepath.Join(CLI.DefaultCapsule, "content", "main.gmi"))
		if err != nil {
			log.Printf("ERROR: %s\n", err)
			os.Exit(1)
		}
		idxf.WriteString(MAIN_GMI_TEMPLATE)
		idxf.Close()

		gf, err := os.Create(filepath.Join(CLI.DefaultCapsule, "group"))
		if err != nil {
			log.Printf("ERROR: %s\n", err)
			os.Exit(1)
		}
		gf.WriteString(GROUP_TEMPLATE)
		gf.Close()

		err = os.Mkdir(filepath.Join(CLI.DefaultCapsule, "bin"), 0700)
		if err != nil {
			log.Printf("ERROR: %s\n", err)
			os.Exit(1)
		}
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

		// See if the command exists in the capsule's bin directory first
		if _, err := os.Stat(filepath.Join(capsule, "bin", cmd[0])); !os.IsNotExist(err) {
			cmd[0] = filepath.Join(capsule, "bin", cmd[0])
		}

		// This command is usingo the built-in template processor
		//
		// Usage:
		// tpl [<path>]
		//
		if cmd[0] == "tpl" {
			fp := "main.gmi"

			if len(cmd) == 2 {
				fp = cmd[1]
			}

			tmpl, err := template.ParseFiles(filepath.Join(capsule, "content", fp))
			if err != nil {
				log.Printf("Error parsing template %s: %s\n", fp, err)
				io.WriteString(s, "Command not found\n")
				s.Exit(127)
				return
			}

			envdata := map[string]interface{}{
				"HOST":  host,
				"IDENT": pubkey,
			}
			data := map[string]interface{}{"env": envdata}

			// Copy only certain environment variables from client
			for _, env := range s.Environ() {
				if strings.HasPrefix(env, "TZ=") {
					envdata["TZ"] = env[3:]
				} else if strings.HasPrefix(env, "LANG=") {
					envdata["LANG"] = env[5:]
				}
			}

			err = tmpl.Execute(s, data)
			if err != nil {
				log.Printf("Error executing template %s: %s\n", fp, err)
				io.WriteString(s, "Command not found\n")
				s.Exit(127)
				return
			}
			s.Exit(0)
			return
		}

		c := exec.Command(cmd[0], cmd[1:]...)

		c.Dir = filepath.Join(capsule, "content") // Current working directory is the capsule content
		c.Env = os.Environ()                      // Copy the environment of the server

		// Copy only certain environment variables from client
		for _, env := range s.Environ() {
			if strings.HasPrefix(env, "TZ=") ||
				strings.HasPrefix(env, "LANG=") {
				c.Env = append(c.Env, env)
			}
		}

		// Strip out any terminal variables
		for ie, e := range c.Env {
			if strings.HasPrefix(e, "TERM=") {
				c.Env = append(c.Env[:ie], c.Env[ie+1:]...)
				break
			}
		}

		// Depending on the command these may be used by that process
		c.Env = append(c.Env, "HOST="+host)
		c.Env = append(c.Env, "IDENT="+pubkey)

		// Add the capsule's path to the PATH environment
		for ipe, pe := range c.Env {
			if strings.HasPrefix(pe, "PATH=") {
				pe = "PATH=" + filepath.Join(capsule, "bin") + ":" + pe[5:]
				c.Env[ipe] = pe
				break
			}
		}

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
