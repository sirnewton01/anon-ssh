package main

import (
	"fmt"
	"io"
	"mime"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

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

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s [<path>|<gemssh_url>]\n", os.Args[0])
		os.Exit(127)
	}

	p := os.Args[1]

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

		cmd := exec.Command("ssh", fmt.Sprintf("%s@%s", username, u.Host), "gemini", u.Path)

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
		fmt.Printf("Only gemssh:// URL scheme is supported\n")
		os.Exit(127)
	} else {
		file, err := os.Open(p)
		if err != nil {
			fmt.Printf("51 Not Found\r\n")
			os.Exit(51)
		}
		defer file.Close()
		if info, err := file.Stat(); err != nil {
			fmt.Printf("51 Not Found\r\n")
			os.Exit(51)
		} else if info.IsDir() {
			file, err = os.Open(filepath.Join(p, "index.gmi"))
			if err != nil {
				fmt.Printf("51 Not Found\r\n")
				os.Exit(51)
			}
			p = file.Name()
			defer file.Close()
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

		fmt.Printf("20 %s\r\n", mt)

		if _, err := io.Copy(os.Stdout, file); err != nil {
			panic(err)
		}
	}
}
