//go:build !windows
// +build !windows

package main

import (
	"fmt"

	_ "github.com/abmpio/registry/consul/starter"
)

func init() {
	fmt.Println("plugin registry.consul init function called")
}

type Bootstrap struct {
}

func newBootstrap() Bootstrap {
	b := Bootstrap{}
	return b
}

func (b Bootstrap) BootstrapPlugin() (err error) {
	return nil
}

var PluginBootstrap = newBootstrap()
