package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/improbable-eng/grpc-web/go/grpcweb"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "grpc-k8s-sample/proto/go"
)

func main() {
	log.Println("start process")

	if err := loadEnv(); err != nil {
		panic(err)
	}

	api := &cobra.Command{
		Use:   "api",
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			interceptor := func(
				ctx context.Context,
				req interface{},
				info *grpc.UnaryServerInfo,
				handler grpc.UnaryHandler) (resp interface{}, err error) {
				res, err := handler(ctx, req)
				if err != nil {
					log.Printf("error %s", err.Error())
				}
				return res, err
			}

			server := grpc.NewServer(grpc.UnaryInterceptor(interceptor))
			pb.RegisterHelloServer(server, &hello{})
			reflection.Register(server)

			runGRPCServer(server)
			runGRPCWebServer(server)

			return nil
		},
	}

	batch := &cobra.Command{
		Use:   "batch",
		Short: "",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("hello cicd batch, %s from env", os.Getenv("MESSAGE"))
			return nil
		},
	}

	cmd := &cobra.Command{
		Use:   "grpc-sample",
		Short: "",
	}

	cmd.AddCommand(api, batch)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func loadEnv() error {
	dotenvBody := os.Getenv("DOTENV_BODY")
	if len(dotenvBody) > 0 {
		envMap, err := godotenv.Parse(bytes.NewBufferString(dotenvBody))
		if err != nil {
			return err
		}
		for k, v := range envMap {
			if err := os.Setenv(k, v); err != nil {
				return err
			}
		}
		return nil
	}

	if envPath := os.Getenv("GO_ENV"); len(envPath) > 0 {
		if err := godotenv.Overload(envPath); err != nil {
			return err
		}
	}

	return nil
}

func runGRPCServer(server *grpc.Server) {
	grpcPort := os.Getenv("GRPC_PORT")
	if len(grpcPort) > 0 {
		listener, err := net.Listen("tcp", ":"+grpcPort)
		if err != nil {
			panic(err.Error())
		}

		go func() {
			log.Printf("running grpc server port: %s", grpcPort)
			if err := server.Serve(listener); err != nil {
				panic(err.Error())
			}
		}()
	}
}

func runGRPCWebServer(server *grpc.Server) {
	webPort := os.Getenv("WEB_PORT")
	if len(webPort) == 0 {
		webPort = "3000"
	}

	listener, err := net.Listen("tcp", ":"+webPort)
	if err != nil {
		panic(err.Error())
	}

	mux := http.NewServeMux()

	mux.Handle("/", grpcweb.WrapServer(
		server,
		grpcweb.WithOriginFunc(func(origin string) bool {
			return true
		}),
		grpcweb.WithAllowNonRootResource(true),
	))

	mux.HandleFunc("/health_check", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(http.StatusOK)
	})

	log.Printf("running http server port: %s", webPort)
	if err := http.Serve(listener, mux); err != nil {
		panic(err.Error())
	}
}

type hello struct {
}

func (s *hello) World(ctx context.Context, req *pb.Empty) (*pb.HelloWorld, error) {
	username := os.Getenv("SECRET_USERNAME")
	password := os.Getenv("SECRET_PASSWORD")

	log.Println(username)
	log.Println(password)

	msg := fmt.Sprintf("hello cicd, %s from env, %s", os.Getenv("MESSAGE"), username)

	return &pb.HelloWorld{
		Message: msg,
	}, nil
}
