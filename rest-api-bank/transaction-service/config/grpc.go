package config

import (
	"log"
	"os"
	"time"

	pb "microservices-bank/proto/accountpb"
	fraudpb "microservices-bank/proto/fraudpb"

	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var AccountClient pb.AccountServiceClient
var FraudClient fraudpb.FraudDetectionServiceClient

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
			// Propagate OTel trace context via gRPC metadata on every call.
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
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

func InitFraudGRPCClient() *grpc.ClientConn {
	addr := os.Getenv("FRAUD_GRPC_ADDR")
	if addr == "" {
		addr = "fraud-detection-service:50052"
	}

	var conn *grpc.ClientConn
	var err error

	for i := 0; i < 10; i++ {
		conn, err = grpc.NewClient(addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			// Propagate OTel trace context via gRPC metadata on every call.
			grpc.WithStatsHandler(otelgrpc.NewClientHandler()),
		)
		if err == nil {
			FraudClient = fraudpb.NewFraudDetectionServiceClient(conn)
			log.Printf("[gRPC] Connected to fraud-detection-service at %s", addr)
			return conn
		}
		log.Printf("[gRPC] Waiting for fraud-detection-service at %s... (%d/10)", addr, i+1)
		time.Sleep(3 * time.Second)
	}

	log.Fatalf("[gRPC] Could not connect to fraud-detection-service at %s: %v", addr, err)
	return nil
}
