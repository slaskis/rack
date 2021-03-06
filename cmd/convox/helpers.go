package main

import (
	"bufio"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/convox/rack/options"
	"github.com/convox/rack/sdk"
	"github.com/convox/rack/structs"
	"github.com/convox/stdcli"
)

func app(c *stdcli.Context) string {
	wd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	return coalesce(c.String("app"), c.LocalSetting("app"), filepath.Base(wd))
}

func coalesce(ss ...string) string {
	for _, s := range ss {
		if s != "" {
			return s
		}
	}

	return ""
}

func copySystemLogs(w io.Writer, r io.Reader) {
	s := bufio.NewScanner(r)

	for s.Scan() {
		parts := strings.SplitN(s.Text(), " ", 3)

		if len(parts) < 3 {
			continue
		}

		if strings.HasPrefix(parts[1], "system/aws") {
			w.Write([]byte(fmt.Sprintf("%s\n", s.Text())))
		}
	}
}

func currentHost(c *stdcli.Context) (string, error) {
	if h := os.Getenv("CONVOX_HOST"); h != "" {
		return h, nil
	}

	if h, _ := c.SettingRead("host"); h != "" {
		return h, nil
	}

	return "", nil
}

func currentPassword(c *stdcli.Context, host string) (string, error) {
	if pw := os.Getenv("CONVOX_PASSWORD"); pw != "" {
		return pw, nil
	}

	return hostAuth(c, host)
}

func currentEndpoint(c *stdcli.Context, rack_ string) (string, error) {
	if e := os.Getenv("RACK_URL"); e != "" {
		return e, nil
	}

	if strings.HasPrefix(rack_, "local/") {
		return fmt.Sprintf("https://rack.%s", strings.SplitN(rack_, "/", 2)[1]), nil
	}

	host, err := currentHost(c)
	if err != nil {
		return "", err
	}

	if host == "" {
		if !localRackRunning() {
			return "", fmt.Errorf("no racks found, try `convox login`")
		}

		var r *rack

		if cr := currentRack(c, ""); cr != "" {
			r, err = matchRack(c, cr)
			if err != nil {
				return "", err
			}
		} else {
			r, err = matchRack(c, "local/")
			if err != nil {
				return "", err
			}
		}

		if r == nil {
			return "", fmt.Errorf("no racks found, try `convox login`")
		}

		return fmt.Sprintf("https://rack.%s", strings.SplitN(r.Name, "/", 2)[1]), nil
	}

	pw, err := currentPassword(c, host)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://convox:%s@%s", pw, host), nil
}

func currentRack(c *stdcli.Context, host string) string {
	if r := c.String("rack"); r != "" {
		return r
	}

	if r := os.Getenv("CONVOX_RACK"); r != "" {
		return r
	}

	if r := c.LocalSetting("rack"); r != "" {
		return r
	}

	if r := hostRacks(c)[host]; r != "" {
		return r
	}

	if r, _ := c.SettingRead("rack"); r != "" {
		return r
	}

	return ""
}

func executableName() string {
	switch runtime.GOOS {
	case "windows":
		return "convox.exe"
	default:
		return "convox"
	}
}

func generateTempKey() (string, error) {
	data := make([]byte, 1024)

	if _, err := rand.Read(data); err != nil {
		return "", err
	}

	hash := sha256.Sum256(data)

	return fmt.Sprintf("tmp/%s", hex.EncodeToString(hash[:])[0:30]), nil
}

func handleSignalTermination(name string) {
	sigs := make(chan os.Signal)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	for range sigs {
		fmt.Printf("\nstopping: %s\n", name)
		exec.Command("docker", "stop", name).Run()
	}
}

func hostAuth(c *stdcli.Context, host string) (string, error) {
	data, err := c.SettingRead("auth")
	if err != nil {
		return "", err
	}

	auth := map[string]string{}

	if data != "" {
		if err := json.Unmarshal([]byte(data), &auth); err != nil {
			return "", err
		}
	}

	if pass, ok := auth[host]; ok {
		return pass, nil
	}

	return "", nil
}

func hostRacks(c *stdcli.Context) map[string]string {
	data, err := c.SettingRead("racks")
	if err != nil {
		return map[string]string{}
	}

	var rs map[string]string

	if err := json.Unmarshal([]byte(data), &rs); err != nil {
		return map[string]string{}
	}

	return rs
}

func provider(c *stdcli.Context) *sdk.Client {
	host, err := currentHost(c)
	if err != nil {
		c.Fail(err)
	}

	r := currentRack(c, host)

	endpoint, err := currentEndpoint(c, r)
	if err != nil {
		c.Fail(err)
	}

	sc, err := sdk.New(endpoint)
	if err != nil {
		c.Fail(err)
	}

	sc.Rack = r

	return sc
}

func rackCommand(name string, version string, router string) (*exec.Cmd, error) {
	vol := "/var/convox"

	switch runtime.GOOS {
	case "darwin":
		vol = "/Users/Shared/convox"
	}

	image := fmt.Sprintf("convox/rack:%s", version)

	exec.Command("docker", "rm", "-f", name).Run()

	args := []string{"run", "--rm"}
	args = append(args, "-e", "COMBINED=true")
	args = append(args, "-e", "COMBINED=true")
	args = append(args, "-e", fmt.Sprintf("IMAGE=%s", image))
	args = append(args, "-e", "PROVIDER=local")
	args = append(args, "-e", fmt.Sprintf("RACK=%s", name))
	args = append(args, "-e", fmt.Sprintf("ROUTER=%s", router))
	args = append(args, "-e", fmt.Sprintf("VERSION=%s", version))
	args = append(args, "-e", fmt.Sprintf("VOLUME=%s", vol))
	args = append(args, "-i")
	args = append(args, "--label", fmt.Sprintf("convox.rack=%s", name))
	args = append(args, "--label", "convox.type=rack")
	args = append(args, "-m", "256m")
	args = append(args, "--name", name)
	args = append(args, "-p", "5443")
	args = append(args, "-v", fmt.Sprintf("%s:/var/convox", vol))
	args = append(args, "-v", "/var/run/docker.sock:/var/run/docker.sock")
	args = append(args, image)

	return exec.Command("docker", args...), nil
}

func saveAuth(c *stdcli.Context, host, password string) error {
	as, err := c.SettingRead("auth")
	if err != nil {
		return err
	}

	data := []byte(coalesce(as, "{}"))

	var auth map[string]string

	if err := json.Unmarshal(data, &auth); err != nil {
		return err
	}

	auth[host] = password

	data, err = json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return err
	}

	if err := c.SettingWrite("auth", string(data)); err != nil {
		return err
	}

	return nil
}

func streamAppSystemLogs(c *stdcli.Context, app string, done chan bool) {
	r, err := provider(c).AppLogs(app, structs.LogsOptions{Prefix: options.Bool(true), Since: options.Duration(0)})
	if err != nil {
		return
	}

	go copySystemLogs(c, r)

	<-done
}

func streamRackSystemLogs(c *stdcli.Context, done chan bool) {
	r, err := provider(c).SystemLogs(structs.LogsOptions{Prefix: options.Bool(true), Since: options.Duration(0)})
	if err != nil {
		return
	}

	go copySystemLogs(c, r)

	<-done
}

func tag(name, value string) string {
	return fmt.Sprintf("<%s>%s</%s>", name, value, name)
}

func wait(interval time.Duration, timeout time.Duration, times int, fn func() (bool, error)) error {
	count := 0
	start := time.Now().UTC()

	for {
		if start.Add(timeout).Before(time.Now().UTC()) {
			return fmt.Errorf("timeout")
		}

		success, err := fn()
		if err != nil {
			return err
		}

		if success {
			count += 1
		} else {
			count = 0
		}

		if count >= times {
			return nil
		}

		time.Sleep(interval)
	}
}

func waitForAppDeleted(c *stdcli.Context, app string) error {
	time.Sleep(5 * time.Second) // give the stack time to start updating

	return wait(5*time.Second, 30*time.Minute, 2, func() (bool, error) {
		_, err := provider(c).AppGet(app)
		if err == nil {
			return false, nil
		}
		if strings.Contains(err.Error(), "no such app") {
			return true, nil
		}
		if strings.Contains(err.Error(), "app not found") {
			return true, nil
		}
		return false, err
	})
}

func waitForAppRunning(c *stdcli.Context, app string) error {
	time.Sleep(5 * time.Second) // give the stack time to start updating

	return wait(5*time.Second, 30*time.Minute, 2, func() (bool, error) {
		a, err := provider(c).AppGet(app)
		if err != nil {
			return false, err
		}

		return a.Status == "running", nil
	})
}

func waitForAppWithLogs(c *stdcli.Context, app string) error {
	done := make(chan bool)

	c.Writef("\n")

	go streamAppSystemLogs(c, app, done)

	if err := waitForAppRunning(c, app); err != nil {
		return err
	}

	done <- true

	return nil
}

func waitForProcessRunning(c *stdcli.Context, app, pid string) error {
	return wait(1*time.Second, 5*time.Minute, 2, func() (bool, error) {
		ps, err := provider(c).ProcessGet(app, pid)
		if err != nil {
			return false, err
		}

		return ps.Status == "running", nil
	})
}

func waitForRackRunning(c *stdcli.Context) error {
	time.Sleep(5 * time.Second) // give the stack time to start updating

	return wait(5*time.Second, 30*time.Minute, 2, func() (bool, error) {
		s, err := provider(c).SystemGet()
		if err != nil {
			return false, err
		}

		return s.Status == "running", nil
	})
}

func waitForRackWithLogs(c *stdcli.Context) error {
	done := make(chan bool)

	c.Writef("\n")

	go streamRackSystemLogs(c, done)

	if err := waitForRackRunning(c); err != nil {
		return err
	}

	done <- true

	return nil
}

func waitForResourceDeleted(c *stdcli.Context, resource string) error {
	time.Sleep(5 * time.Second) // give the stack time to start updating

	return wait(5*time.Second, 30*time.Minute, 2, func() (bool, error) {
		_, err := provider(c).ResourceGet(resource)
		if err == nil {
			return false, nil
		}
		if strings.Contains(err.Error(), "no such resource") {
			return true, nil
		}
		if strings.Contains(err.Error(), "does not exist") {
			return true, nil
		}
		return false, err
	})
}

func waitForResourceRunning(c *stdcli.Context, resource string) error {
	time.Sleep(5 * time.Second) // give the stack time to start updating

	return wait(5*time.Second, 30*time.Minute, 2, func() (bool, error) {
		r, err := provider(c).ResourceGet(resource)
		if err != nil {
			return false, err
		}

		return r.Status == "running", nil
	})
}
