package grpc

import (
	"context"
	"project/domain"
	"project/shared/mapper/generated"
	"project/shared/pb"

	"google.golang.org/protobuf/types/known/emptypb"
)

type GrpcFriendHandler struct {
	ser domain.FriendService
	pb.UnimplementedFriendServiceServer
}

func NewGrpcFriendHandler(ser domain.FriendService) *GrpcFriendHandler {
	return &GrpcFriendHandler{
		ser: ser,
	}
}

func (g GrpcFriendHandler) SendFriendRequest(ctx context.Context, req *pb.SendFriendRequestRequest) (*emptypb.Empty, error) {
	err := g.ser.SendFriendRequest(ctx, req.ActionUserID, req.TargetUserID)
	if err != nil {
		return nil, domain.ToGrpcError(err)
	}
	return &emptypb.Empty{}, nil
}

func (g GrpcFriendHandler) AcceptFriendRequest(ctx context.Context, req *pb.UserIDsPair) (*emptypb.Empty, error) {
	err := g.ser.AcceptFriendRequest(ctx, req.UserID, req.FriendID)
	if err != nil {
		return nil, domain.ToGrpcError(err)
	}
	return &emptypb.Empty{}, nil
}

func (g GrpcFriendHandler) RejectFriendRequest(ctx context.Context, req *pb.UserIDsPair) (*emptypb.Empty, error) {
	err := g.ser.RejectFriendRequest(ctx, req.UserID, req.FriendID)
	if err != nil {
		return nil, domain.ToGrpcError(err)
	}
	return &emptypb.Empty{}, nil
}

func (g GrpcFriendHandler) RemoveFriend(ctx context.Context, req *pb.UserIDsPair) (*emptypb.Empty, error) {
	err := g.ser.RemoveFriend(ctx, req.UserID, req.FriendID)
	if err != nil {
		return nil, domain.ToGrpcError(err)
	}
	return &emptypb.Empty{}, nil
}

func (g GrpcFriendHandler) GetFriends(ctx context.Context, req *pb.GetFriendsRequest) (*pb.ShortProfileList, error) {
	profiles, err := g.ser.GetFriends(ctx, req.UserID, domain.PaginateQueryParams{Limit: req.Limit, Page: req.Page})
	if err != nil {
		return nil, domain.ToGrpcError(err)
	}
	return generated.ToPbShortProfileList(profiles), nil
}

func (g GrpcFriendHandler) GetAllUsers(ctx context.Context, req *pb.GetAllUsersRequest) (*pb.ShortProfileList, error) {
	profiles, err := g.ser.GetAllUsers(ctx, req.UserID, domain.PaginateQueryParams{Limit: req.Limit, Page: req.Page})
	if err != nil {
		return nil, domain.ToGrpcError(err)
	}
	return generated.ToPbShortProfileList(profiles), nil
}

func (g GrpcFriendHandler) SearchShortProfilesByFullNameAndRelationType(ctx context.Context, req *pb.SearchProfilesRequest) (*pb.ShortProfileList, error) {
	profiles, err := g.ser.SearchShortProfilesByFullNameAndRelationType(ctx, req.UserID, domain.PaginateQueryParams{Limit: req.Limit, Page: req.Page}, req.FullName, domain.FriendshipCountType(req.Type))
	if err != nil {
		return nil, domain.ToGrpcError(err)
	}
	return generated.ToPbShortProfileList(profiles), nil
}

func (g GrpcFriendHandler) GetFriendRequests(ctx context.Context, req *pb.GetFriendRequestsRequest) (*pb.ShortProfileList, error) {
	profiles, err := g.ser.GetFriendRequests(ctx, req.UserID, domain.PaginateQueryParams{Limit: req.Limit, Page: req.Page})
	if err != nil {
		return nil, domain.ToGrpcError(err)
	}
	return generated.ToPbShortProfileList(profiles), nil
}

func (g GrpcFriendHandler) GetSentRequests(ctx context.Context, req *pb.GetSentRequestsRequest) (*pb.ShortProfileList, error) {
	profiles, err := g.ser.GetSentRequests(ctx, req.UserID, domain.PaginateQueryParams{Limit: req.Limit, Page: req.Page})
	if err != nil {
		return nil, domain.ToGrpcError(err)
	}
	return generated.ToPbShortProfileList(profiles), nil
}

func (g GrpcFriendHandler) GetFriendshipStatus(ctx context.Context, req *pb.GetFriendshipStatusRequest) (*pb.FriendshipStatusResponse, error) {
	statusValue, err := g.ser.GetFriendshipStatus(ctx, req.UserID, req.FriendID)
	if err != nil {
		return nil, domain.ToGrpcError(err)
	}
	return &pb.FriendshipStatusResponse{Status: string(statusValue)}, nil
}

func (g GrpcFriendHandler) CountUserRelations(ctx context.Context, req *pb.CountUserRelationsRequest) (*pb.UserRelationsCountsResponse, error) {
	counts, err := g.ser.CountUserRelations(ctx, req.UserID)
	if err != nil {
		return nil, domain.ToGrpcError(err)
	}
	return &pb.UserRelationsCountsResponse{
		Accepted: counts.Accepted,
		Pending:  counts.Pending,
		Sent:     counts.Sent,
		Blocked:  counts.Blocked,
	}, nil
}
