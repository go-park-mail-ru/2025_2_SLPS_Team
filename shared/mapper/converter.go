package mapper

import (
	"project/domain"
	"project/shared/pb"
)

//go:generate goverter gen .

// goverter:output:format function
// goverter:matchIgnoreCase
//
//goverter:converter
//goverter:ignoreUnexported
//goverter:extend Conv.*
//goverter:useZeroValueOnPointerInconsistency
type Converter interface {
	ToProtoUserRelationsCounts(counts domain.UserRelationsCounts) *pb.UserRelationsCounts
	FromProtoUserRelationsCounts(pbCounts *pb.UserRelationsCounts) domain.UserRelationsCounts

	ToProtoProfile(profile domain.Profile) *pb.Profile
	FromProtoProfile(pbProfile *pb.Profile) domain.Profile

	ToProtoShortProfile(sp domain.ShortProfile) *pb.ShortProfile
	FromProtoShortProfile(pbSp *pb.ShortProfile) domain.ShortProfile
	ToProtoShortProfileSlice(profiles []domain.ShortProfile) *pb.GetShortProfileByUserIDsResponse
	FromProtoShortProfileSlice(pbResp *pb.GetShortProfileByUserIDsResponse) []domain.ShortProfile

	ToProtoShortProfileMap(profiles map[int32]domain.ShortProfile) *pb.GetShortProfileMapByUserIDsResponse
	FromProtoShortProfileMap(pbResp *pb.GetShortProfileMapByUserIDsResponse) map[int32]domain.ShortProfile
}
