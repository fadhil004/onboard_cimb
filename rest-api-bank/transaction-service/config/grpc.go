package config

import (
	"log"
	"os"
	"time"

	pb "microservices-bank/proto/accountpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var AccountClient pb.AccountServiceClient

func InitGRPCClient() *grpc.ClientConn {
	addr := os.Getenv("ACCOUNT_GRPC_ADDR")
	if addr == "" {
		addr = "account-service:50051"
	}

	var conn *grpc.ClientConn
	var err error

	// Retry connection to account service gRPC
	for i := 0; i < 10; i++ {
		conn, err = grpc.NewClient(addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
		)
		if err == nil {
			AccountClient = pb.NewAccountServiceClient(conn)
			log.Printf("[gRPC] Connected to account-service at %s", addr)
			return conn
		}
		log.Printf("[gRPC] Waiting for account-service at %s... (%d/10)", addr, i+1)
		time.Sleep(3 * time.Second)
	}

	log.Fatalf("[gRPC] Could not connect to account-service at %s: %v", addr, err)
	return nil
}
