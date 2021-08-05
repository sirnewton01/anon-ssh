package setup

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
)

func AssertCapsuleConfig(hostname string) error {
	// Check using ssh -G whether things appear to be set up
	cmd := exec.Command("ssh", "-G", fmt.Sprintf("capsule@%s", hostname))
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

		fmt.Fprintf(os.Stderr, `This is the first time you are using capsules.
Add the following to your ~/.ssh/config to enable capsule access:

IdentitiesOnly yes

Match user capsule
  PubkeyAuthentication yes
  PasswordAuthentication no
  PreferredAuthentications publickey
  Port 1966
  Include ~/.ssh/*_cap_config
`)

		return fmt.Errorf("Capsule access has not been configured in ~/.ssh/config Please set it up first before using this comand")
	}

	if !strings.Contains(conf, fmt.Sprintf("HOST=%s", hostname)) {
		keypath := filepath.Join(sshconfdir, fmt.Sprintf("%s_cap_id_rsa", hostname))

		if _, err := os.Stat(keypath); os.IsNotExist(err) {
			cmd := exec.Command("ssh-keygen", "-m", "PEM", "-P", "", "-f", keypath)
			err := cmd.Run()
			if err != nil {
				return err
			}
		}

		ahc, err := os.OpenFile(filepath.Join(sshconfdir, fmt.Sprintf("%s_cap_config", hostname)), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return err
		}
		defer ahc.Close()

		ahc.WriteString("\n")
		ahc.WriteString(fmt.Sprintf("Match user capsule host %s\n", hostname))
		ahc.WriteString(fmt.Sprintf("  SetEnv HOST=%s\n", hostname))
		ahc.WriteString(fmt.Sprintf("  IdentityFile ~/.ssh/%s_cap_id_rsa\n", hostname))
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

