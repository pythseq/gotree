package cmd

import (
	"github.com/spf13/cobra"
)

var drawNoTipLabels bool
var drawNoBranchLengths bool
var drawInternalNodeLabels bool
var drawSupport bool
var drawSupportCutoff float64
var drawInternalNodeSymbols bool
var drawNodeComment bool

// drawCmd represents the draw command
var drawCmd = &cobra.Command{
	Use:   "draw",
	Short: "Draw trees",
	Long:  `Draw trees `,
}

func init() {
	RootCmd.AddCommand(drawCmd)

	drawCmd.PersistentFlags().StringVarP(&intreefile, "input", "i", "stdin", "Input tree")
	drawCmd.PersistentFlags().StringVarP(&outtreefile, "output", "o", "stdout", "Output file")
	drawCmd.PersistentFlags().BoolVar(&drawNoTipLabels, "no-tip-labels", false, "Draw the tree without tip labels")
	drawCmd.PersistentFlags().BoolVar(&drawNoBranchLengths, "no-branch-lengths", false, "Draw the tree without branch lengths (all the same length)")
	drawCmd.PersistentFlags().BoolVar(&drawInternalNodeLabels, "with-node-labels", false, "Draw the tree with internal node labels")
	drawCmd.PersistentFlags().BoolVar(&drawInternalNodeSymbols, "with-node-symbols", false, "Draw the tree with internal node symbols")
	drawCmd.PersistentFlags().BoolVar(&drawSupport, "with-branch-support", false, "Highlight highly supported branches")
	drawCmd.PersistentFlags().Float64Var(&drawSupportCutoff, "support-cutoff", 0.7, "Cutoff for highlithing supported branches")
	drawCmd.PersistentFlags().BoolVar(&drawNodeComment, "with-node-comments", false, "Draw the tree with internal node comments (if --with-node-labels is not set)")
}
