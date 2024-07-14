package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/synchthia/packy/service"
)

func ListCommand() *cobra.Command {
	c := &cobra.Command{
		Use:   "list",
		Short: "List artifacts",
		Run: func(cmd *cobra.Command, args []string) {
			dir, _ := cmd.Flags().GetString("directory")
			servers, _ := cmd.Flags().GetStringSlice("servers")
			cacheSvc, err := service.InitCache(dir)
			if err != nil {
				panic(err)
			}

			r2 := service.InitR2FromEnv(dir, cacheSvc)
			for _, server := range servers {
				contents, err := r2.List(server)
				if err != nil {
					panic(err)
				}

				for _, content := range contents {
					fmt.Printf("[%s] %s (%s)\n", content.Hash, content.Name, content.Path)
				}
			}
		},
	}

	c.Flags().StringP("directory", "d", ".", "extract directory")
	c.Flags().StringSliceVarP(&[]string{"servers"}, "servers", "s", []string{"global"}, "targets (ex: 'global, <server_name>')")

	return c
}
