/*
Copyright © 2024 Chinmay
*/
package cmd

import (
	"context"
	"fmt"
	"log/slog"
	log "log/slog"
	"net"
	"net/http"
	"os"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"

	pb "github.com/chinmayb/brainiac-brawl/gen/go/api"
	"github.com/chinmayb/brainiac-brawl/pkg/data"
	"github.com/chinmayb/brainiac-brawl/pkg/play"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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
		if err := data.ParseQuizData("./quiz-data.yaml"); err != nil {
			log.Error("error parsing data", err)
			os.Exit(1)
		}
		setConfig(cmd)
		spawnServer()
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

func spawnServer() {
	logOpts := &slog.HandlerOptions{
		AddSource: true,
		Level:     logParser(*logLevel)}
	logger := log.New(log.NewTextHandler(os.Stdout, logOpts))
	grpcListener, err := net.Listen("tcp", fmt.Sprintf("localhost:%d", *grpcPort))
	if err != nil {
		log.Error("err while listening", "reason", err)
		os.Exit(1)
	}

	var opts []grpc.ServerOption
	grpcServer := grpc.NewServer(opts...)

	// Create game server implementation
	gameServer := play.NewPlayServer(logger)
	pb.RegisterGamesServer(grpcServer, gameServer)

	gwmux := runtime.NewServeMux()
	ctx := context.Background()

	// Update registration
	if err := pb.RegisterGamesHandlerFromEndpoint(ctx, gwmux, fmt.Sprintf("localhost:%d", *grpcPort),
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}); err != nil {
		logger.Error("failed to register gateway", "error", err)
		os.Exit(1)
	}

	go func() {
		logger.Info("running GRPC Server at", "port", *grpcPort)
		if err := grpcServer.Serve(grpcListener); err != nil {
			logger.Error("failed to serve grpc", "error", err)
			os.Exit(1)
		}
	}()

	grpcConn, err := grpc.NewClient(fmt.Sprintf("localhost:%d", *grpcPort),
		[]grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}...)
	if err != nil {
		logger.Error("failed to serve http", "error", err)
		os.Exit(1)
	}
	defer grpcConn.Close()
	client := pb.NewGamesClient(grpcConn)
	httpMux := WSHandler(ctx, *logger, client)

	httpMux.Handle("/play", gwmux)
	logger.Info("Serving gRPC-Gateway & WS on http://0.0.0.0:8080")
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), httpMux); err != nil {
		logger.Error("failed to serve http", "error", err)
		os.Exit(1)
	}
}
