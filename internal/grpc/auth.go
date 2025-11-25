package grpc

import (
	"context"
	"project/domain"
	"project/shared/mapper/generated"
	"project/shared/pb"

	"google.golang.org/protobuf/types/known/emptypb"
)

type GrpcAuthHandler struct {
	ser domain.AuthService
	pb.UnimplementedAuthServiceServer
}

func NewGrpcAuthHandler(ser domain.AuthService) *GrpcAuthHandler {
	return &GrpcAuthHandler{
		ser: ser,
	}
}
func (g GrpcAuthHandler) Register(ctx context.Context, in *pb.RegisterRequest) (*pb.LoginResponse, error) {
	req := generated.ProtoToRegisterRequest(in)

	userID, err := g.ser.Register(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pb.LoginResponse{UserId: userID}, nil
}

func (g GrpcAuthHandler) Login(ctx context.Context, in *pb.LoginRequest) (*pb.LoginResponse, error) {
	req := domain.User{
		Email:    in.Email,
		Password: in.Password,
	}

	userID, err := g.ser.Login(ctx, req)
	if err != nil {
		return nil, err
	}

	return &pb.LoginResponse{UserId: userID}, nil
}

func (g GrpcAuthHandler) Logout(ctx context.Context, in *pb.SessionCookieRequest) (*emptypb.Empty, error) {
	err := g.ser.Logout(ctx, in.SessionCookie)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

func (g GrpcAuthHandler) IsLoggedIn(ctx context.Context, in *pb.SessionCookieRequest) (*pb.SessionResponse, error) {
	session, err := g.ser.IsLoggedIn(ctx, in.SessionCookie)
	if err != nil {
		return nil, err
	}
	return &pb.SessionResponse{
		UserId:    session.UserID,
		CsrfToken: session.CSRFToken,
	}, nil
}

func (g GrpcAuthHandler) AddSession(ctx context.Context, in *pb.UserIDRequest) (*pb.SIDAndCSRFToken, error) {
	sidAndCSRF, err := g.ser.AddSession(ctx, in.UserId)
	if err != nil {
		return nil, err
	}
	return &pb.SIDAndCSRFToken{
		Sid:       sidAndCSRF.SID,
		CsrfToken: sidAndCSRF.CSRFToken,
	}, nil
}

func (g GrpcAuthHandler) GetUserRole(ctx context.Context, in *pb.UserIDRequest) (*pb.UserRoleResponse, error) {
	role, err := g.ser.GetUserRole(ctx, in.UserId)
	if err != nil {
		return nil, err
	}
	return &pb.UserRoleResponse{
		Role: role,
	}, nil
}

func (g GrpcAuthHandler) IsUserExists(ctx context.Context, in *pb.UserIDRequest) (*pb.UserExistsResponse, error) {
	exists, err := g.ser.IsUserExists(ctx, in.UserId)
	if err != nil {
		return nil, err
	}
	return &pb.UserExistsResponse{
		Exists: exists,
	}, nil
}
