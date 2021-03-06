// Copyright © 2016 NAME HERE <EMAIL ADDRESS>
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

package cmd

import (
	"fmt"
	goio "io"
	"time"

	"github.com/evolbioinfo/gotree/io"
	"github.com/evolbioinfo/gotree/support"
	"github.com/evolbioinfo/gotree/tree"
	"github.com/spf13/cobra"
)

// classicalCmd represents the classical command
var classicalCmd = &cobra.Command{
	Use:   "classical",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	RunE: func(cmd *cobra.Command, args []string) (err error) {
		var refTree *tree.Tree
		var boottreefile goio.Closer
		var boottreechan <-chan tree.Trees

		writeLogClassical()
		if refTree, err = readTree(supportIntree); err != nil {
			io.LogError(err)
			return
		}
		if boottreefile, boottreechan, err = readTrees(supportBoottrees); err != nil {
			io.LogError(err)
			return
		}
		defer boottreefile.Close()

		var supporter *support.ClassicalSupporter = support.NewClassicalSupporter(false)
		err = support.ComputeSupport(refTree, boottreechan, nil, rootCpus, supporter)
		//e := support.Classical(refTree, boottreechan, rootCpus)
		if err != nil {
			io.LogError(err)
			return
		}
		supportOut.WriteString(refTree.Newick() + "\n")
		supportLog.WriteString(fmt.Sprintf("End         : %s\n", time.Now().Format(time.RFC822)))
		return
	},
}

func init() {
	computesupportCmd.AddCommand(classicalCmd)
}

func writeLogClassical() {
	supportLog.WriteString("Classical Support\n")
	supportLog.WriteString(fmt.Sprintf("Start       : %s\n", time.Now().Format(time.RFC822)))
	supportLog.WriteString(fmt.Sprintf("Input tree  : %s\n", supportIntree))
	supportLog.WriteString(fmt.Sprintf("Boot trees  : %s\n", supportBoottrees))
	supportLog.WriteString(fmt.Sprintf("Output tree : %s\n", supportOutFile))
	supportLog.WriteString(fmt.Sprintf("CPUs        : %d\n", rootCpus))
}
