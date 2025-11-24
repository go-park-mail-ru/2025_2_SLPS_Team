package grpc

import (
	"context"
	"project/domain"
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

}
func (g GrpcProfileHandler) UpdateProfile(ctx context.Context, in *pb.UpdateProfileRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {

}
func (g GrpcProfileHandler) UpdateAvatar(ctx context.Context, in *pb.UpdateAvatarRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {

}
func (g GrpcProfileHandler) UpdateHeader(ctx context.Context, in *pb.UpdateHeaderRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {

}
func (g GrpcProfileHandler) GetProfileByUserID(ctx context.Context, in *pb.GetProfileByUserIDRequest, opts ...grpc.CallOption) (*pb.GetProfileByUserIDResponse, error) {

}
func (g GrpcProfileHandler) DeleteAvatarByUserID(ctx context.Context, in *pb.DeleteAvatarRequest, opts ...grpc.CallOption) (*emptypb.Empty, error) {

}
func (g GrpcProfileHandler) GetShortProfileMapByUserIDs(ctx context.Context, in *pb.GetShortProfileMapByUserIDsRequest, opts ...grpc.CallOption) (*pb.GetShortProfileMapByUserIDsResponse, error) {

}
func (g GrpcProfileHandler) GetShortProfileByUserIDs(ctx context.Context, in *pb.GetShortProfileByUserIDsRequest, opts ...grpc.CallOption) (*pb.GetShortProfileByUserIDsResponse, error) {

}
