//
// Created by Javid Rzayev on 11.08.26.
//

package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "kubenpuctl",
	Short: "Inspect accelerator devices and per-pod usage on this node",
	Long: `kubenpuctl inspects NPU and GPU accelerators that kubeNPU can see on 
the local node.

It reports which DRM and accel devices were found, which vendor implementation
matched each one, and which pods are using them. Device discovery reads sysfs
only and needs no elevated privileges; live views attach eBPF programs and
require CAP_BPF and CAP_PERFMON.`,
	SilenceUsage: true,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
