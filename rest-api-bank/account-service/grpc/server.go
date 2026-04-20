package grpcserver

import (
	"context"
	"microservices-bank/account-service/middleware"
	"microservices-bank/account-service/pkg/logger"
	"microservices-bank/account-service/repository"
	pb "microservices-bank/proto/accountpb"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AccountGRPCServer implements the gRPC AccountService server.
type AccountGRPCServer struct {
	pb.UnimplementedAccountServiceServer
	Repo repository.AccountRepository
}

func NewAccountGRPCServer(repo repository.AccountRepository) *AccountGRPCServer {
	return &AccountGRPCServer{Repo: repo}
}

func (s *AccountGRPCServer) GetByAccountNumber(ctx context.Context, req *pb.GetByAccountNumberRequest) (*pb.AccountResponse, error) {
	ctx, span := middleware.Tracer.Start(ctx, "gRPC.GetByAccountNumber")
	defer span.End()

	logger.Logger.Info("[gRPC] GetByAccountNumber", zap.String("account_number", req.AccountNumber))

	acc, err := s.Repo.GetByAccountNumber(ctx, req.AccountNumber)
	if err != nil {
		logger.Logger.Error("[gRPC] account not found", zap.String("account_number", req.AccountNumber), zap.Error(err))
		return nil, status.Errorf(codes.NotFound, "account not found: %s", req.AccountNumber)
	}

	return &pb.AccountResponse{
		Id:            acc.ID.String(),
		AccountNumber: acc.AccountNumber,
		AccountHolder: acc.AccountHolder,
		Balance:       acc.Balance,
		CreatedAt:     acc.CreatedAt.String(),
		UpdatedAt:     acc.UpdatedAt.String(),
	}, nil
}

func (s *AccountGRPCServer) GetByID(ctx context.Context, req *pb.GetByIDRequest) (*pb.AccountResponse, error) {
	ctx, span := middleware.Tracer.Start(ctx, "gRPC.GetByID")
	defer span.End()

	logger.Logger.Info("[gRPC] GetByID", zap.String("id", req.Id))

	uid, err := parseUUID(req.Id)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid UUID: %s", req.Id)
	}

	acc, err := s.Repo.GetByID(ctx, uid)
	if err != nil {
		logger.Logger.Error("[gRPC] account not found", zap.String("id", req.Id), zap.Error(err))
		return nil, status.Errorf(codes.NotFound, "account not found: %s", req.Id)
	}

	return &pb.AccountResponse{
		Id:            acc.ID.String(),
		AccountNumber: acc.AccountNumber,
		AccountHolder: acc.AccountHolder,
		Balance:       acc.Balance,
		CreatedAt:     acc.CreatedAt.String(),
		UpdatedAt:     acc.UpdatedAt.String(),
	}, nil
}

func (s *AccountGRPCServer) UpdateBalance(ctx context.Context, req *pb.UpdateBalanceRequest) (*pb.UpdateBalanceResponse, error) {
	ctx, span := middleware.Tracer.Start(ctx, "gRPC.UpdateBalance")
	defer span.End()

	logger.Logger.Info("[gRPC] UpdateBalance",
		zap.String("account_number", req.AccountNumber),
		zap.Int64("amount", req.Amount),
	)

	acc, err := s.Repo.GetByAccountNumber(ctx, req.AccountNumber)
	if err != nil {
		logger.Logger.Error("[gRPC] account not found for balance update",
			zap.String("account_number", req.AccountNumber),
			zap.Error(err),
		)
		return nil, status.Errorf(codes.NotFound, "account not found: %s", req.AccountNumber)
	}

	newBalance := acc.Balance + req.Amount
	if newBalance < 0 {
		logger.Logger.Error("[gRPC] insufficient balance",
			zap.Int64("current_balance", acc.Balance),
			zap.Int64("requested_change", req.Amount),
		)
		return &pb.UpdateBalanceResponse{
			Success:       false,
			AccountNumber: req.AccountNumber,
			NewBalance:    acc.Balance,
			Message:       "insufficient balance",
		}, status.Errorf(codes.FailedPrecondition, "insufficient balance")
	}

	acc.Balance = newBalance
	if err := s.Repo.Update(ctx, acc); err != nil {
		logger.Logger.Error("[gRPC] failed to update balance", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "failed to update balance: %v", err)
	}

	logger.Logger.Info("[gRPC] balance updated successfully",
		zap.String("account_number", req.AccountNumber),
		zap.Int64("new_balance", newBalance),
	)

	return &pb.UpdateBalanceResponse{
		Success:       true,
		AccountNumber: req.AccountNumber,
		NewBalance:    newBalance,
		Message:       "balance updated successfully",
	}, nil
}
