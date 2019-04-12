package main

import (
	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/version"
	"github.com/cybozu-go/coil/cni"
)

func main() {
	skel.PluginMain(cni.Add, cni.Del, version.All)
}
