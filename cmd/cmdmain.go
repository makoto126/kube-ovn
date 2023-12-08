package main

import (
	"os"
	"path/filepath"

	"github.com/kubeovn/kube-ovn/cmd/cni"
	"github.com/kubeovn/kube-ovn/cmd/daemon"
	"github.com/kubeovn/kube-ovn/pkg/util"
)

const (
	CmdCNI    = "kube-ovn"
	CmdDaemon = "kube-ovn-daemon"
)

func main() {
	cmd := filepath.Base(os.Args[0])
	switch cmd {
	case CmdCNI:
		cni.CmdMain()
	case CmdDaemon:
		daemon.CmdMain()
	default:
		util.LogFatalAndExit(nil, "%s is an unknown command", cmd)
	}
}
