/*
Copyright © 2024 Chinmay
*/
package cmd

import (
	"fmt"
	log "log/slog"
	"net"
	"os"

	pb "github.com/chinmayb/brainiac-brawl/gen/go/api"
	"github.com/chinmayb/brainiac-brawl/pkg/play"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"google.golang.org/grpc"
)

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		spawnServer(cmd)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:x
	// serveCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}

func spawnServer(cmd *cobra.Command) {
	cmd.HasPersistentFlags()
	pflag.Parse()
	port := pflag.Int("port", 8080, "port")
	lis, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *port))
	if err != nil {
		log.Error("errr", err)
		os.Exit(1)
	}
	logger := log.New(log.NewTextHandler(os.Stdout, nil))
	fmt.Print(logger)
	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterBrainiacBrawlServer(grpcServer, play.NewPlayServer())
	grpcServer.Serve(lis)
}
