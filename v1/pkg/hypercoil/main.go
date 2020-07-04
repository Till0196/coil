package main

import (
	"io"
	"os"
	"path/filepath"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/cybozu-go/coil/v1/cni"
	controller "github.com/cybozu-go/coil/v1/pkg/coil-controller/cmd"
	installer "github.com/cybozu-go/coil/v1/pkg/coil-installer/cmd"
	coilctl "github.com/cybozu-go/coil/v1/pkg/coilctl/cmd"
	coild "github.com/cybozu-go/coil/v1/pkg/coild/cmd"
)

func usage() {
	io.WriteString(os.Stderr, `Usage: hypercoil COMMAND [ARGS ...]

COMMAND:
    - coil               CNI plugin.
    - coild              DaemonSet service.
    - coil-controller    Kubernetes controller for coil.
    - coilctl            Command-line utility.
    - coil-installer     Installs coil.
`)
}

func main() {
	name := filepath.Base(os.Args[0])
	if name == "hypercoil" {
		if len(os.Args) == 1 {
			usage()
			os.Exit(1)
		}
		name = os.Args[1]
		os.Args = os.Args[1:]
	}

	switch name {
	case "coil":
		skel.PluginMain(cni.Add, cni.Del, version.All)
	case "coild":
		coild.Execute()
	case "coil-controller":
		controller.Execute()
	case "coilctl":
		coilctl.Execute()
	case "coil-installer":
		installer.Execute()
	default:
		usage()
		os.Exit(1)
	}
}
