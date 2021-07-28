package main

import (
	"fmt"
	gemini "git.sr.ht/~yotam/go-gemini"
	"github.com/alecthomas/kong"
	"io"
	"mime"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

var CLI struct {
	Path          string `arg name:"path" help:"The path/URL of the gemini resource to start a transaction or the root of a capsule to start a server." required""`
	ListenAddress string `flag name:"listen-address" help:"Start a gemini server and listen on this address and the provided path as the root of the capsule. Example: --listen-address=:1965"`
	HostCertPEM   string `flag name:"host-cert" help:"The path to the host cert in PEM format."`
	HostKeyPEM    string `flag name:"host-key" help:"The path to the host private key in PEM format."`
}

func assertAnonConfig(username string, hostname string) error {
	// Check using ssh -G whether things appear to be set up
	cmd := exec.Command("ssh", "-G", fmt.Sprintf("%s@%s", username, hostname))
	sshconf, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	conf := string(sshconf)

	user, err := user.Current()
	if err != nil {
		return err
	}

	sshconfdir := filepath.Join(user.HomeDir, ".ssh")

	// Maybe we should check that identitiesonly yes is present too to
	// avoid problems with the ssh agent?
	if !strings.Contains(conf, "pubkeyauthentication yes") ||
		!strings.Contains(conf, "passwordauthentication no") ||
		!strings.Contains(conf, "port 1966") {

		fmt.Fprintf(os.Stderr, `Add the following to your ~/.ssh/config to enable anonymous access:

Match user anonymous
  IdentitiesOnly yes
  PubkeyAuthentication yes
  PasswordAuthentication no
  PreferredAuthentications publickey
  Port 1966
  Include ~/.ssh/*_anon_config
`)

		return fmt.Errorf("Anonymous access has not been configured in ~/.ssh/config Please set it up first before using this comand")
	}

	if !strings.Contains(conf, fmt.Sprintf("HOST=%s", hostname)) {
		keypath := filepath.Join(sshconfdir, fmt.Sprintf("%s_anon_id_rsa", hostname))

		if _, err := os.Stat(keypath); os.IsNotExist(err) {
			cmd := exec.Command("ssh-keygen", "-m", "PEM", "-P", "", "-f", keypath)
			err := cmd.Run()
			if err != nil {
				return err
			}
		}

		ahc, err := os.OpenFile(filepath.Join(sshconfdir, fmt.Sprintf("%s_anon_config", hostname)), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}
		defer ahc.Close()

		ahc.WriteString("\n")
		ahc.WriteString(fmt.Sprintf("Match user anonymous host %s\n", hostname))
		ahc.WriteString(fmt.Sprintf("  SetEnv HOST=%s\n", hostname))
		ahc.WriteString(fmt.Sprintf("  IdentityFile ~/.ssh/%s_anon_id_rsa\n", hostname))
		ahc.WriteString("\n")
	}

	// Check that there is a server key in the known hosts
	checkkeycmd := exec.Command("ssh-keygen", "-F", fmt.Sprintf("[%s]:1966", hostname))
	err = checkkeycmd.Run()
	if err != nil && (checkkeycmd.ProcessState == nil || checkkeycmd.ProcessState.ExitCode() == -1) {
		return err
	}

	// We don't yet have the host key for this host, so let's add it to the known hosts
	if checkkeycmd.ProcessState.ExitCode() == 1 {
		cmd := exec.Command("ssh-keyscan", "-p", "1966", hostname)
		hk, err := cmd.Output()
		if err != nil {
			return err
		}
		hostkey := string(hk)

		kh, err := os.OpenFile(filepath.Join(sshconfdir, "known_hosts"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}
		defer kh.Close()

		kh.WriteString(hostkey)
	}

	return nil
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

	if CLI.ListenAddress != "" {
		if CLI.HostCertPEM == "" || CLI.HostKeyPEM == "" {
			fmt.Printf("When running in server mode the host-cert and host-key must be provided\n")
			os.Exit(127)
		}

		h := handler{p}

		panic(gemini.ListenAndServe(CLI.ListenAddress, CLI.HostCertPEM, CLI.HostKeyPEM, h))
	}

	u, err := url.Parse(p)

	if err == nil && u.Scheme == "gemssh" {
		// Perform SSH functions to connect to server

		// TODO handle warning / error messages about host key verification
		user := u.User
		username := "anonymous"
		if user != nil {
			username = user.Username()
		}

		// We do some special setup for anonymous access, otherwise,
		//  we just use the usual configuration
		if username == "anonymous" {
			err := assertAnonConfig(username, u.Host)
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
		fmt.Printf("%d %s\r\n", resp.Status, resp.Meta)
		if resp.Body != nil {
			defer resp.Body.Close()
			if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
				panic(err)
			}
		}
	} else if err == nil && u.Scheme != "" {
		fmt.Printf("Only gemssh:// URL scheme is supported\n")
		os.Exit(127)
	} else {
		req := gemini.Request{}
		h := handler{p}
		resp := h.Handle(req)
		fmt.Printf("%d %s\r\n", resp.Status, resp.Meta)
		if resp.Status > 29 {
			os.Exit(resp.Status)
		}
		defer resp.Body.Close()

		if _, err := io.Copy(os.Stdout, resp.Body); err != nil {
			panic(err)
		}
	}
}
