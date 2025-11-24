package mapper

import (
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ----- int ↔ int32 -----
func Conv_Int32ToInt(in int32) int {
	return int(in)
}

func Conv_IntToInt32(in int) int32 {
	return int32(in)
}

// ProtoToTime конвертирует *timestamppb.Timestamp в time.Time
func Conv_ProtoToTime(ts *timestamppb.Timestamp) time.Time {
	if ts == nil {
		return time.Time{}
	}
	return ts.AsTime()
}

// TimeToProto конвертирует time.Time в *timestamppb.Timestamp
func Conv_TimeToProto(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

// MapIntToInt32 конвертирует map[int]interface{} в map[int32]interface{}
func Conv_MapIntToInt32(in map[int]interface{}) map[int32]interface{} {
	out := make(map[int32]interface{}, len(in))
	for k, v := range in {
		out[int32(k)] = v
	}
	return out
}

// MapInt32ToInt конвертирует map[int32]interface{} в map[int]interface{}
func Conv_MapInt32ToInt(in map[int32]interface{}) map[int]interface{} {
	out := make(map[int]interface{}, len(in))
	for k, v := range in {
		out[int(k)] = v
	}
	return out
}

// ToInterfacePointer принимает значение любого типа и возвращает указатель на него как interface{}
func Conv_ToInterfacePointer(value *interface{}) interface{} {
	return *value
}

// FromInterfacePointer принимает указатель на interface{} и возвращает само значение
func Conv_FromInterfacePointer(ptr interface{}) *interface{} {
	return &ptr
}
