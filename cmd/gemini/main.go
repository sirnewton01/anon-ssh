package main

import (
	"fmt"
	gemini "git.sr.ht/~yotam/go-gemini"
	"github.com/alecthomas/kong"
	"github.com/sirnewton01/ssh-capsules/pkg/setup"
	"io"
	"mime"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var CLI struct {
	Path          string `arg name:"path" help:"The path/URL/SSH address of the gemini resource to start a transaction or the root of a capsule to start a server." required""`
	ListenAddress string `flag name:"listen-address" help:"Start a gemini server and listen on this address and the provided path as the root of the capsule. Example: --listen-address=:1965"`
	HostCertPEM   string `flag name:"host-cert" help:"The path to the host cert in PEM format."`
	HostKeyPEM    string `flag name:"host-key" help:"The path to the host private key in PEM format."`
}

type handler struct {
	path string
}

func (h handler) Handle(req gemini.Request) gemini.Response {
	u, _ := url.Parse(req.URL)
	resp := gemini.Response{}
	// TODO sanitize
	p := filepath.Join(h.path, u.Path)

	file, err := os.Open(p)
	if err != nil {
		resp.Status = 51
		resp.Meta = "Not Found"
		return resp
	}

	if info, err := file.Stat(); err != nil {
		file.Close()
		resp.Status = 51
		resp.Meta = "Not Found"
		return resp
	} else if info.IsDir() {
		file.Close()

		p = filepath.Join(p, "index.gmi")
		file, err = os.Open(p)
		if err != nil {
			resp.Status = 51
			resp.Meta = "Not Found"
			return resp
		}
	}

	fe := filepath.Ext(p)
	mt := ""
	if fe != "" {
		mt = mime.TypeByExtension(fe)
	}

	if fe == ".gmi" {
		mt = "text/gemini"
	} else if mt == "" {
		mt = "application/octet-stream"
	}

	resp.Status = 20
	resp.Meta = mt
	resp.Body = file

	return resp
}

func main() {
	kong.Parse(&CLI)

	p := CLI.Path
	ps := strings.Split(p, "/")

	if CLI.ListenAddress != "" {
		if CLI.HostCertPEM == "" || CLI.HostKeyPEM == "" {
			fmt.Printf("When running in server mode the host-cert and host-key must be provided\n")
			os.Exit(127)
		}

		h := handler{p}

		panic(gemini.ListenAndServe(CLI.ListenAddress, CLI.HostCertPEM, CLI.HostKeyPEM, h))
	}

	u, err := url.Parse(p)

	if err == nil && u.Scheme == "gemcap" {
		// Perform SSH functions to connect to server

		// TODO handle warning / error messages about host key verification
		user := u.User
		username := "capsule"
		if user != nil {
			username = user.Username()
		}

		// We do some special setup for capsule access, otherwise,
		//  we just use the usual configuration
		if username == "capsule" {
			err := setup.AssertCapsuleConfig(u.Host)
			if err != nil {
				panic(err)
			}
		}

		path := u.Path
		if path == "" {
			path = "/"
		}
		// TODO more sanitization of the path in addition to the server sanitization

		cmd := exec.Command("ssh", fmt.Sprintf("%s@%s", username, u.Host), "gemini", path)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			panic(err)
		}
		stderr, err := cmd.StderrPipe()
		if err != nil {
			panic(err)
		}

		if err := cmd.Start(); err != nil {
			panic(err)
		}

		go func() {
			defer stdout.Close()
			io.Copy(os.Stdout, stdout)
		}()
		go func() {
			defer stderr.Close()
			io.Copy(os.Stderr, stderr)
		}()

		if err := cmd.Wait(); err != nil {
			if _, ok := err.(*exec.ExitError); !ok {
				panic(err)
			}
		}

		os.Exit(cmd.ProcessState.ExitCode())
	} else if err == nil && u.Scheme == "gemini" {
		if u.Port() == "" {
			u.Host = u.Host + ":1965"
		}
		gemini.DefaultClient.InsecureSkipVerify = true
		resp, err := gemini.Fetch(u.String())
		if err != nil {
			panic(err)
		}
		fmt.Fprintf(os.Stderr, "%d %s\r\n", resp.Status, resp.Meta)
		if resp.Body != nil {
			defer resp.Body.Close()
			if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
				panic(err)
			}
		}
	} else if len(ps) > 1 && strings.Contains(ps[0], ":") {
		// SSH style addresses
		userhost := ps[0]
		userhost = userhost[:len(userhost)-1]
		host := userhost
		username := "capsule"
		if strings.Contains(userhost, "@") {
			uhs := strings.Split(userhost, "@")
			if len(uhs) != 2 {
				fmt.Fprintf(os.Stderr, "Invalid path")
				os.Exit(127)
			}
			username = uhs[0]
			host = uhs[1]
		}
		path := p[len(userhost)+1:]
		if path == "" {
			path = "/"
		}

                // We do some special setup for capsule access, otherwise,
                //  we just use the usual configuration
                if username == "capsule" {
                        err := setup.AssertCapsuleConfig(host)
                        if err != nil {
                                panic(err)
                        }
                }

                // TODO more sanitization of the path in addition to the server sanitization

                cmd := exec.Command("ssh", fmt.Sprintf("%s@%s", username, host), "gemini", path)

                stdout, err := cmd.StdoutPipe()
                if err != nil {
                        panic(err)
                }
                stderr, err := cmd.StderrPipe()
                if err != nil {
                        panic(err)
                }

                if err := cmd.Start(); err != nil {
                        panic(err)
                }

                go func() {
                        defer stdout.Close()
                        io.Copy(os.Stdout, stdout)
                }()
                go func() {
                        defer stderr.Close()
                        io.Copy(os.Stderr, stderr)
                }()

                if err := cmd.Wait(); err != nil {
                        if _, ok := err.(*exec.ExitError); !ok {
                                panic(err)
                        }
                }

                os.Exit(cmd.ProcessState.ExitCode())
	} else if err == nil && u.Scheme != "" {
		fmt.Printf("Only gemcap:// and gemini:// URL schemes are supported\n")
		os.Exit(127)
	} else {
		req := gemini.Request{}
		h := handler{p}
		resp := h.Handle(req)
		fmt.Fprintf(os.Stderr, "%d %s\r\n", resp.Status, resp.Meta)
		if resp.Status > 29 {
			os.Exit(resp.Status)
		}
		defer resp.Body.Close()

		if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
			panic(err)
		}
	}
}
