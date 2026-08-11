//
// Created by Javid Rzayev on 11.08.26.
//

package main

import (
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/jrzayev/kubenpu/pkg/discovery"
	"github.com/jrzayev/kubenpu/pkg/hw"
	_ "github.com/jrzayev/kubenpu/pkg/hw/i915"
	_ "github.com/jrzayev/kubenpu/pkg/hw/ivpu"
	"github.com/spf13/cobra"
)

var devicesCmd = &cobra.Command{
	Use:   "devices",
	Short: "Discover connected devices",
	RunE: func(cmd *cobra.Command, args []string) error {
		paths := discovery.Paths{
			DriPath:        "/dev/dri",
			AccelPath:      "/dev/accel",
			SysfsDriPath:   "/sys/class/drm",
			SysfsAccelPath: "/sys/class/accel",
		}

		devices, err := discovery.Discover(paths)
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		defer func(w *tabwriter.Writer) {
			_ = w.Flush()
		}(w)

		_, _ = fmt.Fprintln(w, "DRIVER\tPCI ID\tADDRESS\tNODES\tVENDOR")

		for _, device := range devices {
			nodes := make([]string, 0, len(device.Nodes))
			for _, node := range device.Nodes {
				nodes = append(nodes, fmt.Sprintf("%d:%d", node.Major, node.Minor))
			}

			vendorName := "-"
			if vendor := hw.GetVendor(device); vendor != nil {
				vendorName = vendor.Name()
			}

			pciID := device.PciID
			if pciID == "" {
				pciID = "-"
			}

			_, _ = fmt.Fprintf(
				w,
				"%s\t%s\t%s\t%s\t%s\n",
				device.DriverName,
				pciID,
				device.PciAddress,
				strings.Join(nodes, ","),
				vendorName,
			)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(devicesCmd)
}
