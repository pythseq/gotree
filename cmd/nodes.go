package cmd

import (
	"fmt"
	"github.com/fredericlemoine/gotree/io"
	"github.com/spf13/cobra"
)

// nodesCmd represents the nodes command
var nodesCmd = &cobra.Command{
	Use:   "nodes",
	Short: "Displays statistics on nodes of input tree",
	Long: `Displays statistics on nodes of input tree.

Statistics are displayed in text format (tab separated):
1 - Id of node
2 - Nb neighbors
3 - Name of node
4 - depth of node (shortest path to a tip)

Example of usage:

gotree stats nodes -i t.nw

`,
	Run: func(cmd *cobra.Command, args []string) {
		statsout.WriteString("id\tnneigh\tname\tdepth\n")
		var depth int
		var err error
		for i, n := range statsintree.Nodes() {
			if depth, err = n.Depth(); err != nil {
				io.ExitWithMessage(err)
			}
			statsout.WriteString(fmt.Sprintf("%d\t%d\t%s\t%d\n", i, n.Nneigh(), n.Name(), depth))
		}
	},
}

func init() {
	statsCmd.AddCommand(nodesCmd)
}
