package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("Usage: %s [<path>|<gemssh_url>]\n", os.Args[0])
		os.Exit(127)
	}

	p := os.Args[1]

	u, err := url.Parse(p)

	if err == nil && u.Scheme == "gemssh" {
		// Perform SSH functions to connect to server

		// TODO verify that the host is configured in ssh config
		// TODO handle warning / error messages about host key verification
		user := u.User
		username := "anonymous"
		if user != nil {
			username = user.Username()
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
                        //panic(err)
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
			defer file.Close()
		}

		if strings.HasSuffix(p,".gmi") {
			fmt.Printf("20 text/gemini\r\n")
		} else if strings.HasSuffix(p, ".txt") {
			fmt.Printf("20 text/plain\r\n")
		} else if strings.HasSuffix(p, ".md") {
			fmt.Printf("20 text/plain\r\n")
		} else {
			// TODO handle more file extensions and their mappings to media types (RFC 2046)
			fmt.Printf("20 application/octet-stream\r\n")
		}

		if _, err := io.Copy(os.Stdout, file); err != nil {
			panic(err)
		}
	}
}
