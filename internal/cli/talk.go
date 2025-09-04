package cli

import (
	"github.com/Stromo01/D7024E/pkg/helloworld"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(TalkCmd)
}

var TalkCmd = &cobra.Command{
	Use:   "talk",
	Short: "Say something",
	Long:  "Say something",
	Run: func(cmd *cobra.Command, args []string) {
		hellworld := helloworld.NewHelloWorld()
		hellworld.Talk()
	},
}
