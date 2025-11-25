package grpc

import (
	"context"
	"project/domain"
	"project/shared/mapper/generated"
	"project/shared/pb"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type GrpcProfileHandler struct {
	ser domain.ProfileService
	pb.UnimplementedProfileServiceServer
}

func NewGrpcProfileHandler(ser domain.ProfileService) *GrpcProfileHandler {
	return &GrpcProfileHandler{
		ser: ser,
	}
}
func (g GrpcProfileHandler) CreateProfile(ctx context.Context, in *pb.CreateProfileRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	err := g.ser.CreateProfile(ctx, generated.FromProtoProfile(in.Profile))
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
func (g GrpcProfileHandler) UpdateProfile(ctx context.Context, in *pb.UpdateProfileRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	err := g.ser.UpdateProfile(ctx, generated.FromProtoProfile(in.Profile), in.UserID, generated.ProtoToFiles(in.Files))
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil

}
func (g GrpcProfileHandler) UpdateAvatar(ctx context.Context, in *pb.UpdateAvatarRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	err := g.ser.UpdateAvatar(ctx, in.UserID, generated.ProtoToFiles(in.Avatar))
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil

}

func (g GrpcProfileHandler) UpdateHeader(ctx context.Context, in *pb.UpdateHeaderRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	err := g.ser.UpdateAvatar(ctx, in.UserID, generated.ProtoToFiles(in.Header))
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil

}

func (g GrpcProfileHandler) GetProfileByUserID(ctx context.Context, in *pb.GetProfileByUserIDRequest, opts ...grpc.CallOption) (*pb.GetProfileByUserIDResponse, error) {
	profile, err := g.ser.GetProfileByUserID(ctx, in.SelfUserID, in.UserID)
	if err != nil {
		return nil, err
	}
	return &pb.GetProfileByUserIDResponse{Profile: generated.ToProtoProfile(*profile)}, nil
}

func (g GrpcProfileHandler) DeleteAvatarByUserID(ctx context.Context, in *pb.DeleteAvatarRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	err := g.ser.DeleteAvatarByUserID(ctx, in.UserID)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil

}

func (g GrpcProfileHandler) GetShortProfileMapByUserIDs(ctx context.Context, in *pb.GetShortProfileMapByUserIDsRequest, opts ...grpc.CallOption) (*pb.GetShortProfileMapByUserIDsResponse, error) {
	profiles, err := g.ser.GetShortProfileMapByUserIDs(ctx, in.UserIDs)
	if err != nil {
		return nil, err
	}
	return generated.ToProtoShortProfileMap(profiles), nil

}

func (g GrpcProfileHandler) GetShortProfileByUserIDs(ctx context.Context, in *pb.GetShortProfileByUserIDsRequest, opts ...grpc.CallOption) (*pb.GetShortProfileByUserIDsResponse, error) {
	profiles, err := g.ser.GetShortProfileByUserIDs(ctx, in.UserIDs)
	if err != nil {
		return nil, err
	}
	return generated.ToProtoShortProfileSlice(profiles), nil

}
