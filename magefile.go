// Copyright 2023 LiveKit, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build mage

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"

	"github.com/livekit/mageutil"
	"github.com/magefile/mage/sh"
)

var Default = Install

const (
	gstVersion      = "1.22.8"
	libniceVersion  = "0.1.21"
	chromiumVersion = "117.0.5874.0"
	dockerBuild     = "docker build"
	dockerBuildX    = "docker buildx build --push --platform linux/amd64,linux/arm64"
)

type packageInfo struct {
	Dir string
}

func Proto() error {
	ctx := context.Background()
	fmt.Println("generating protobuf")

	// parse go mod output
	pkgOut, err := mageutil.Out(ctx, "go list -json -m github.com/livekit/protocol")
	if err != nil {
		return err
	}
	pi := packageInfo{}
	if err = json.Unmarshal(pkgOut, &pi); err != nil {
		return err
	}

	_, err = mageutil.GetToolPath("protoc")
	if err != nil {
		return err
	}
	protocGoPath, err := mageutil.GetToolPath("protoc-gen-go")
	if err != nil {
		return err
	}
	protocGrpcGoPath, err := mageutil.GetToolPath("protoc-gen-go-grpc")
	if err != nil {
		return err
	}

	// generate grpc-related protos
	return mageutil.RunDir(ctx, "pkg/ipc", fmt.Sprintf(
		"protoc"+
			" --go_out ."+
			" --go-grpc_out ."+
			" --go_opt=paths=source_relative"+
			" --go-grpc_opt=paths=source_relative"+
			" --plugin=go=%s"+
			" --plugin=go-grpc=%s"+
			" -I%s -I=. ipc.proto",
		protocGoPath, protocGrpcGoPath, pi.Dir,
	))
}

func Integration(configFile string) error {
	if err := Deadlock(); err != nil {
		return err
	}
	defer Dotfiles()
	defer Sync()

	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	if configFile != "" {
		if strings.HasPrefix(configFile, "test/") {
			configFile = configFile[5:]
		} else {
			oldLocation := configFile
			idx := strings.LastIndex(configFile, "/")
			if idx != -1 {
				configFile = configFile[idx+1:]
			}
			if err = os.Rename(oldLocation, "test/"+configFile); err != nil {
				return err
			}
		}

		configFile = "/out/" + configFile
	}

	defer func() {
		// for some reason, these can't be deleted from within the docker container
		files, _ := os.ReadDir("test/output")
		for _, file := range files {
			if file.IsDir() {
				_ = os.RemoveAll(path.Join("test/output", file.Name()))
			}
		}
	}()

	return mageutil.Run(context.Background(),
		"docker build -t egress-test -f build/test/Dockerfile .",
		fmt.Sprintf("docker run --rm -e EGRESS_CONFIG_FILE=%s -v %s/test:/out egress-test", configFile, dir),
	)
}

func Deadlock() error {
	ctx := context.Background()
	if err := mageutil.Run(ctx, "go get github.com/sasha-s/go-deadlock"); err != nil {
		return err
	}
	if err := mageutil.Pipe("grep -rl sync.Mutex ./pkg", "xargs sed -i  -e s/sync.Mutex/deadlock.Mutex/g"); err != nil {
		return err
	}
	if err := mageutil.Pipe("grep -rl sync.RWMutex ./pkg", "xargs sed -i  -e s/sync.RWMutex/deadlock.RWMutex/g"); err != nil {
		return err
	}
	if err := mageutil.Pipe("grep -rl deadlock.Mutex\\|deadlock.RWMutex ./pkg", "xargs goimports -w"); err != nil {
		return err
	}
	return mageutil.Run(ctx, "go mod tidy")
}

func Sync() error {
	if err := mageutil.Pipe("grep -rl deadlock.Mutex ./pkg", "xargs sed -i  -e s/deadlock.Mutex/sync.Mutex/g"); err != nil {
		return err
	}
	if err := mageutil.Pipe("grep -rl deadlock.RWMutex ./pkg", "xargs sed -i  -e s/deadlock.RWMutex/sync.RWMutex/g"); err != nil {
		return err
	}
	if err := mageutil.Pipe("grep -rl sync.Mutex\\|sync.RWMutex ./pkg", "xargs goimports -w"); err != nil {
		return err
	}
	return mageutil.Run(context.Background(), "go mod tidy")
}

func Build() error {
	return mageutil.Run(context.Background(),
		fmt.Sprintf("docker pull usamaliaqat/chrome-installer:%s", chromiumVersion),
		fmt.Sprintf("docker pull usamaliaqat/gstreamer:%s-dev", gstVersion),
		"docker pull usamaliaqat/egress-templates",
		"docker build -t usamaliaqat/egress:latest -f build/egress/Dockerfile .",
	)
}

func BuildChrome() error {
	return mageutil.Run(context.Background(),
		"docker pull ubuntu:22.04",
		"docker build -t usamaliaqat/chrome-installer ./build/chrome",
	)
}

func PublishChrome() error {
	return mageutil.Run(context.Background(),
		"docker pull ubuntu:22.04",
		fmt.Sprintf(
			"%s -t usamaliaqat/chrome-installer:%s ./build/chrome",
			dockerBuildX, chromiumVersion,
		),
	)
}

func BuildTemplate() error {
	return mageutil.Run(context.Background(),
		"docker pull ubuntu:22.04",
		"docker build -t usamaliaqat/egress-templates -f ./build/template/Dockerfile .",
	)
}

func BuildGStreamer() error {
	return buildGstreamer(dockerBuild)
}

func RunLocally() error {
	os.Setenv("CGO_ENABLED", "1")
	os.Setenv("GO111MODULE", "on")
	os.Setenv("GODEBUG", "disablethp=1")
	return sh.Run("go", "run", "./cmd/server")
}

func BuildLocally() error {
	os.Setenv("CGO_ENABLED", "1")
	os.Setenv("GO111MODULE", "on")
	os.Setenv("GODEBUG", "disablethp=1")
	return sh.Run("go", "build", "-a", "-o", "dist/egress", "./cmd/server")
}

func StopAllServices() error {
	output, err := sh.Output("sudo", "systemctl", "list-units", "egress@*.service", "--no-legend")
	if err != nil {
		return err
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Extract the service name from the line
		fields := strings.Fields(line)
		if len(fields) > 0 {
			serviceName := fields[0]

			// Stop the service
			fmt.Printf("Stopping service: %s\n", serviceName)
			err := exec.Command("sudo", "systemctl", "stop", serviceName).Run()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func StartAllServices() error {
	output, err := sh.Output("sudo", "systemctl", "list-units", "egress@*.service", "--no-legend")
	if err != nil {
		return err
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Extract the service name from the line
		fields := strings.Fields(line)
		if len(fields) > 0 {
			serviceName := fields[0]

			// Stop the service
			fmt.Printf("Stopping service: %s\n", serviceName)
			err := exec.Command("sudo", "systemctl", "start", serviceName).Run()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func RestartAllServices() error {
	output, err := sh.Output("sudo", "systemctl", "list-units", "egress@*.service", "--no-legend")
	if err != nil {
		return err
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Extract the service name from the line
		fields := strings.Fields(line)
		if len(fields) > 0 {
			serviceName := fields[0]

			// Stop the service
			fmt.Printf("Stopping service: %s\n", serviceName)
			err := exec.Command("sudo", "systemctl", "restart", serviceName).Run()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func DisableAllServices() error {
	output, err := sh.Output("sudo", "systemctl", "list-units", "egress@*.service", "--no-legend")
	if err != nil {
		return err
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Extract the service name from the line
		fields := strings.Fields(line)
		if len(fields) > 0 {
			serviceName := fields[0]

			// Stop the service
			fmt.Printf("Stopping service: %s\n", serviceName)
			err := exec.Command("sudo", "systemctl", "disable", serviceName).Run()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func StopAndDisableAllServices() error {
	output, err := sh.Output("sudo", "systemctl", "list-units", "egress@*.service", "--no-legend")
	if err != nil {
		return err
	}
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		// Extract the service name from the line
		fields := strings.Fields(line)
		if len(fields) > 0 {
			serviceName := fields[0]

			// Stop the service
			fmt.Printf("Stopping service: %s\n", serviceName)
			if err := exec.Command("sudo", "systemctl", "disable", serviceName).Run(); err != nil {
				return err
			}
			if err := exec.Command("sudo", "systemctl", "stop", serviceName).Run(); err != nil {
				return err
			}
		}
	}

	return nil

}

func EnableAndStartService(number int) error {
	serviceName := fmt.Sprintf("egress@%d", number)
	if err := sh.Run("sudo", "systemctl", "enable", serviceName); err != nil {
		return err
	}

	return sh.Run("sudo", "systemctl", "start", serviceName)
}

func DisableAndStopService(number int) error {
	serviceName := fmt.Sprintf("egress@%d", number)
	if err := sh.Run("sudo", "systemctl", "disable", serviceName); err != nil {
		return err
	}

	return sh.Run("sudo", "systemctl", "stop", serviceName)
}

func ConfigurePulseService() error {
	if err := sh.Run("sudo", "cp", "pulseaudio.service", "/etc/systemd/system/pulseaudio.service"); err != nil {
		return err
	}
	if err := sh.Run("sudo", "systemctl", "daemon-reload"); err != nil {
		return err
	}
	if err := sh.Run("sudo", "systemctl", "enable", "pulseaudio"); err != nil {
		return err
	}
	return sh.Run("sudo", "systemctl", "start", "pulseaudio")
}

func ConfigureService() error {
	if err := StopAndDisableAllServices(); err != nil {
		log.Print(err)
	}

	if err := sh.Run("sudo", "cp", "egress.service", "/etc/systemd/system/egress@.service"); err != nil {
		return err
	}
	if err := sh.Run("sudo", "systemctl", "daemon-reload"); err != nil {
		return err
	}
	return EnableAndStartService(1)
}

func Deploy() error {
	if err := sh.Run("sudo", "rm", "-rf", "/usr/local/bin/egress"); err != nil {
		return err
	}

	if err := sh.Run("sudo", "rm", "-rf", "/usr/local/etc/egress.yaml"); err != nil {
		return err
	}

	if err := sh.Run("sudo", "cp", "dist/egress", "/usr/local/bin/egress"); err != nil {
		return err
	}

	if err := sh.Run("sudo", "cp", "egress.yaml", "/usr/local/etc/egress.yaml"); err != nil {
		return err
	}

	return RestartAllServices()
}

func Install() error {
	if err := BuildLocally(); err != nil {
		return err
	}
	if err := ConfigureService(); err != nil {
		return err
	}
	return Deploy()
}

func buildGstreamer(cmd string) error {
	commands := []string{"docker pull ubuntu:23.10"}
	for _, build := range []string{"base", "dev", "prod", "prod-rs"} {
		commands = append(commands, fmt.Sprintf("%s"+
			" --build-arg GSTREAMER_VERSION=%s"+
			" --build-arg LIBNICE_VERSION=%s"+
			" -t usamaliaqat/gstreamer:%s-%s"+
			" -t usamaliaqat/gstreamer:%s-%s-%s"+
			" -f build/gstreamer/Dockerfile-%s"+
			" ./build/gstreamer",
			cmd, gstVersion, libniceVersion, gstVersion, build, gstVersion, build, runtime.GOARCH, build,
		))
	}

	return mageutil.Run(context.Background(), commands...)
}

func Dotfiles() error {
	files, err := os.ReadDir("test/output")
	if err != nil {
		return err
	}

	dots := make(map[string]bool)
	pngs := make(map[string]bool)
	for _, file := range files {
		name := file.Name()
		if strings.HasSuffix(name, ".dot") {
			dots[name[:len(name)-4]] = true
		} else if strings.HasSuffix(file.Name(), ".png") {
			pngs[name[:len(name)-4]] = true
		}
	}

	for name := range dots {
		if !pngs[name] {
			if err := mageutil.Run(context.Background(), fmt.Sprintf(
				"dot -Tpng test/output/%s.dot -o test/output/%s.png",
				name, name,
			)); err != nil {
				return err
			}
		}
	}

	return nil
}
