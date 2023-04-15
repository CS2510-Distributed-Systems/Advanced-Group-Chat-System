// Code generated by protoc-gen-go. DO NOT EDIT.
// versions:
// 	protoc-gen-go v1.28.1
// 	protoc        v3.21.12
// source: chat_service.proto

package pb

import (
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
	reflect "reflect"
	sync "sync"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type GroupChatRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	// Types that are assignable to Action:
	//
	//	*GroupChatRequest_Append
	//	*GroupChatRequest_Like
	//	*GroupChatRequest_Unlike
	//	*GroupChatRequest_Print
	//	*GroupChatRequest_Logout
	//	*GroupChatRequest_Joinchat
	//	*GroupChatRequest_Serverview
	Action isGroupChatRequest_Action `protobuf_oneof:"action"`
}

func (x *GroupChatRequest) Reset() {
	*x = GroupChatRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_chat_service_proto_msgTypes[0]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GroupChatRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GroupChatRequest) ProtoMessage() {}

func (x *GroupChatRequest) ProtoReflect() protoreflect.Message {
	mi := &file_chat_service_proto_msgTypes[0]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GroupChatRequest.ProtoReflect.Descriptor instead.
func (*GroupChatRequest) Descriptor() ([]byte, []int) {
	return file_chat_service_proto_rawDescGZIP(), []int{0}
}

func (m *GroupChatRequest) GetAction() isGroupChatRequest_Action {
	if m != nil {
		return m.Action
	}
	return nil
}

func (x *GroupChatRequest) GetAppend() *AppendChat {
	if x, ok := x.GetAction().(*GroupChatRequest_Append); ok {
		return x.Append
	}
	return nil
}

func (x *GroupChatRequest) GetLike() *LikeMessage {
	if x, ok := x.GetAction().(*GroupChatRequest_Like); ok {
		return x.Like
	}
	return nil
}

func (x *GroupChatRequest) GetUnlike() *UnLikeMessage {
	if x, ok := x.GetAction().(*GroupChatRequest_Unlike); ok {
		return x.Unlike
	}
	return nil
}

func (x *GroupChatRequest) GetPrint() *PrintChat {
	if x, ok := x.GetAction().(*GroupChatRequest_Print); ok {
		return x.Print
	}
	return nil
}

func (x *GroupChatRequest) GetLogout() *Logout {
	if x, ok := x.GetAction().(*GroupChatRequest_Logout); ok {
		return x.Logout
	}
	return nil
}

func (x *GroupChatRequest) GetJoinchat() *JoinChat {
	if x, ok := x.GetAction().(*GroupChatRequest_Joinchat); ok {
		return x.Joinchat
	}
	return nil
}

func (x *GroupChatRequest) GetServerview() *ServerViewRequest {
	if x, ok := x.GetAction().(*GroupChatRequest_Serverview); ok {
		return x.Serverview
	}
	return nil
}

type isGroupChatRequest_Action interface {
	isGroupChatRequest_Action()
}

type GroupChatRequest_Append struct {
	Append *AppendChat `protobuf:"bytes,1,opt,name=append,proto3,oneof"`
}

type GroupChatRequest_Like struct {
	Like *LikeMessage `protobuf:"bytes,2,opt,name=like,proto3,oneof"`
}

type GroupChatRequest_Unlike struct {
	Unlike *UnLikeMessage `protobuf:"bytes,3,opt,name=unlike,proto3,oneof"`
}

type GroupChatRequest_Print struct {
	Print *PrintChat `protobuf:"bytes,4,opt,name=print,proto3,oneof"`
}

type GroupChatRequest_Logout struct {
	Logout *Logout `protobuf:"bytes,5,opt,name=logout,proto3,oneof"`
}

type GroupChatRequest_Joinchat struct {
	Joinchat *JoinChat `protobuf:"bytes,6,opt,name=joinchat,proto3,oneof"`
}

type GroupChatRequest_Serverview struct {
	Serverview *ServerViewRequest `protobuf:"bytes,7,opt,name=serverview,proto3,oneof"`
}

func (*GroupChatRequest_Append) isGroupChatRequest_Action() {}

func (*GroupChatRequest_Like) isGroupChatRequest_Action() {}

func (*GroupChatRequest_Unlike) isGroupChatRequest_Action() {}

func (*GroupChatRequest_Print) isGroupChatRequest_Action() {}

func (*GroupChatRequest_Logout) isGroupChatRequest_Action() {}

func (*GroupChatRequest_Joinchat) isGroupChatRequest_Action() {}

func (*GroupChatRequest_Serverview) isGroupChatRequest_Action() {}

type GroupChatResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Group *Group `protobuf:"bytes,1,opt,name=group,proto3" json:"group,omitempty"`
	Event string `protobuf:"bytes,2,opt,name=event,proto3" json:"event,omitempty"`
}

func (x *GroupChatResponse) Reset() {
	*x = GroupChatResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_chat_service_proto_msgTypes[1]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *GroupChatResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*GroupChatResponse) ProtoMessage() {}

func (x *GroupChatResponse) ProtoReflect() protoreflect.Message {
	mi := &file_chat_service_proto_msgTypes[1]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use GroupChatResponse.ProtoReflect.Descriptor instead.
func (*GroupChatResponse) Descriptor() ([]byte, []int) {
	return file_chat_service_proto_rawDescGZIP(), []int{1}
}

func (x *GroupChatResponse) GetGroup() *Group {
	if x != nil {
		return x.Group
	}
	return nil
}

func (x *GroupChatResponse) GetEvent() string {
	if x != nil {
		return x.Event
	}
	return ""
}

type ServerViewRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Event string `protobuf:"bytes,1,opt,name=event,proto3" json:"event,omitempty"`
}

func (x *ServerViewRequest) Reset() {
	*x = ServerViewRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_chat_service_proto_msgTypes[2]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ServerViewRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerViewRequest) ProtoMessage() {}

func (x *ServerViewRequest) ProtoReflect() protoreflect.Message {
	mi := &file_chat_service_proto_msgTypes[2]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServerViewRequest.ProtoReflect.Descriptor instead.
func (*ServerViewRequest) Descriptor() ([]byte, []int) {
	return file_chat_service_proto_rawDescGZIP(), []int{2}
}

func (x *ServerViewRequest) GetEvent() string {
	if x != nil {
		return x.Event
	}
	return ""
}

type ServerViewResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Peerservers []int64 `protobuf:"varint,1,rep,packed,name=peerservers,proto3" json:"peerservers,omitempty"`
}

func (x *ServerViewResponse) Reset() {
	*x = ServerViewResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_chat_service_proto_msgTypes[3]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *ServerViewResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*ServerViewResponse) ProtoMessage() {}

func (x *ServerViewResponse) ProtoReflect() protoreflect.Message {
	mi := &file_chat_service_proto_msgTypes[3]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use ServerViewResponse.ProtoReflect.Descriptor instead.
func (*ServerViewResponse) Descriptor() ([]byte, []int) {
	return file_chat_service_proto_rawDescGZIP(), []int{3}
}

func (x *ServerViewResponse) GetPeerservers() []int64 {
	if x != nil {
		return x.Peerservers
	}
	return nil
}

type LoginRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	User *User `protobuf:"bytes,1,opt,name=user,proto3" json:"user,omitempty"`
}

func (x *LoginRequest) Reset() {
	*x = LoginRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_chat_service_proto_msgTypes[4]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LoginRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoginRequest) ProtoMessage() {}

func (x *LoginRequest) ProtoReflect() protoreflect.Message {
	mi := &file_chat_service_proto_msgTypes[4]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoginRequest.ProtoReflect.Descriptor instead.
func (*LoginRequest) Descriptor() ([]byte, []int) {
	return file_chat_service_proto_rawDescGZIP(), []int{4}
}

func (x *LoginRequest) GetUser() *User {
	if x != nil {
		return x.User
	}
	return nil
}

type LoginResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	User *User `protobuf:"bytes,1,opt,name=user,proto3" json:"user,omitempty"`
}

func (x *LoginResponse) Reset() {
	*x = LoginResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_chat_service_proto_msgTypes[5]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LoginResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LoginResponse) ProtoMessage() {}

func (x *LoginResponse) ProtoReflect() protoreflect.Message {
	mi := &file_chat_service_proto_msgTypes[5]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LoginResponse.ProtoReflect.Descriptor instead.
func (*LoginResponse) Descriptor() ([]byte, []int) {
	return file_chat_service_proto_rawDescGZIP(), []int{5}
}

func (x *LoginResponse) GetUser() *User {
	if x != nil {
		return x.User
	}
	return nil
}

type LogoutRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Logout *Logout `protobuf:"bytes,1,opt,name=logout,proto3" json:"logout,omitempty"`
}

func (x *LogoutRequest) Reset() {
	*x = LogoutRequest{}
	if protoimpl.UnsafeEnabled {
		mi := &file_chat_service_proto_msgTypes[6]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LogoutRequest) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LogoutRequest) ProtoMessage() {}

func (x *LogoutRequest) ProtoReflect() protoreflect.Message {
	mi := &file_chat_service_proto_msgTypes[6]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LogoutRequest.ProtoReflect.Descriptor instead.
func (*LogoutRequest) Descriptor() ([]byte, []int) {
	return file_chat_service_proto_rawDescGZIP(), []int{6}
}

func (x *LogoutRequest) GetLogout() *Logout {
	if x != nil {
		return x.Logout
	}
	return nil
}

type LogoutResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Status bool `protobuf:"varint,1,opt,name=status,proto3" json:"status,omitempty"`
}

func (x *LogoutResponse) Reset() {
	*x = LogoutResponse{}
	if protoimpl.UnsafeEnabled {
		mi := &file_chat_service_proto_msgTypes[7]
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		ms.StoreMessageInfo(mi)
	}
}

func (x *LogoutResponse) String() string {
	return protoimpl.X.MessageStringOf(x)
}

func (*LogoutResponse) ProtoMessage() {}

func (x *LogoutResponse) ProtoReflect() protoreflect.Message {
	mi := &file_chat_service_proto_msgTypes[7]
	if protoimpl.UnsafeEnabled && x != nil {
		ms := protoimpl.X.MessageStateOf(protoimpl.Pointer(x))
		if ms.LoadMessageInfo() == nil {
			ms.StoreMessageInfo(mi)
		}
		return ms
	}
	return mi.MessageOf(x)
}

// Deprecated: Use LogoutResponse.ProtoReflect.Descriptor instead.
func (*LogoutResponse) Descriptor() ([]byte, []int) {
	return file_chat_service_proto_rawDescGZIP(), []int{7}
}

func (x *LogoutResponse) GetStatus() bool {
	if x != nil {
		return x.Status
	}
	return false
}

var File_chat_service_proto protoreflect.FileDescriptor

var file_chat_service_proto_rawDesc = []byte{
	0x0a, 0x12, 0x63, 0x68, 0x61, 0x74, 0x5f, 0x73, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x2e, 0x70,
	0x72, 0x6f, 0x74, 0x6f, 0x12, 0x04, 0x63, 0x68, 0x61, 0x74, 0x1a, 0x0a, 0x63, 0x68, 0x61, 0x74,
	0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x1a, 0x0a, 0x75, 0x73, 0x65, 0x72, 0x2e, 0x70, 0x72, 0x6f,
	0x74, 0x6f, 0x1a, 0x0a, 0x72, 0x61, 0x66, 0x74, 0x2e, 0x70, 0x72, 0x6f, 0x74, 0x6f, 0x22, 0xda,
	0x02, 0x0a, 0x10, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x43, 0x68, 0x61, 0x74, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x12, 0x2a, 0x0a, 0x06, 0x61, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x18, 0x01, 0x20,
	0x01, 0x28, 0x0b, 0x32, 0x10, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x41, 0x70, 0x70, 0x65, 0x6e,
	0x64, 0x43, 0x68, 0x61, 0x74, 0x48, 0x00, 0x52, 0x06, 0x61, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x12,
	0x27, 0x0a, 0x04, 0x6c, 0x69, 0x6b, 0x65, 0x18, 0x02, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x11, 0x2e,
	0x63, 0x68, 0x61, 0x74, 0x2e, 0x4c, 0x69, 0x6b, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65,
	0x48, 0x00, 0x52, 0x04, 0x6c, 0x69, 0x6b, 0x65, 0x12, 0x2d, 0x0a, 0x06, 0x75, 0x6e, 0x6c, 0x69,
	0x6b, 0x65, 0x18, 0x03, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x13, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e,
	0x55, 0x6e, 0x4c, 0x69, 0x6b, 0x65, 0x4d, 0x65, 0x73, 0x73, 0x61, 0x67, 0x65, 0x48, 0x00, 0x52,
	0x06, 0x75, 0x6e, 0x6c, 0x69, 0x6b, 0x65, 0x12, 0x27, 0x0a, 0x05, 0x70, 0x72, 0x69, 0x6e, 0x74,
	0x18, 0x04, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0f, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x50, 0x72,
	0x69, 0x6e, 0x74, 0x43, 0x68, 0x61, 0x74, 0x48, 0x00, 0x52, 0x05, 0x70, 0x72, 0x69, 0x6e, 0x74,
	0x12, 0x26, 0x0a, 0x06, 0x6c, 0x6f, 0x67, 0x6f, 0x75, 0x74, 0x18, 0x05, 0x20, 0x01, 0x28, 0x0b,
	0x32, 0x0c, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x4c, 0x6f, 0x67, 0x6f, 0x75, 0x74, 0x48, 0x00,
	0x52, 0x06, 0x6c, 0x6f, 0x67, 0x6f, 0x75, 0x74, 0x12, 0x2c, 0x0a, 0x08, 0x6a, 0x6f, 0x69, 0x6e,
	0x63, 0x68, 0x61, 0x74, 0x18, 0x06, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0e, 0x2e, 0x63, 0x68, 0x61,
	0x74, 0x2e, 0x4a, 0x6f, 0x69, 0x6e, 0x43, 0x68, 0x61, 0x74, 0x48, 0x00, 0x52, 0x08, 0x6a, 0x6f,
	0x69, 0x6e, 0x63, 0x68, 0x61, 0x74, 0x12, 0x39, 0x0a, 0x0a, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72,
	0x76, 0x69, 0x65, 0x77, 0x18, 0x07, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x17, 0x2e, 0x63, 0x68, 0x61,
	0x74, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x56, 0x69, 0x65, 0x77, 0x52, 0x65, 0x71, 0x75,
	0x65, 0x73, 0x74, 0x48, 0x00, 0x52, 0x0a, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x76, 0x69, 0x65,
	0x77, 0x42, 0x08, 0x0a, 0x06, 0x61, 0x63, 0x74, 0x69, 0x6f, 0x6e, 0x22, 0x4c, 0x0a, 0x11, 0x47,
	0x72, 0x6f, 0x75, 0x70, 0x43, 0x68, 0x61, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x12, 0x21, 0x0a, 0x05, 0x67, 0x72, 0x6f, 0x75, 0x70, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32,
	0x0b, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x52, 0x05, 0x67, 0x72,
	0x6f, 0x75, 0x70, 0x12, 0x14, 0x0a, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x18, 0x02, 0x20, 0x01,
	0x28, 0x09, 0x52, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x22, 0x29, 0x0a, 0x11, 0x53, 0x65, 0x72,
	0x76, 0x65, 0x72, 0x56, 0x69, 0x65, 0x77, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x14,
	0x0a, 0x05, 0x65, 0x76, 0x65, 0x6e, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x09, 0x52, 0x05, 0x65,
	0x76, 0x65, 0x6e, 0x74, 0x22, 0x36, 0x0a, 0x12, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x56, 0x69,
	0x65, 0x77, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x20, 0x0a, 0x0b, 0x70, 0x65,
	0x65, 0x72, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x18, 0x01, 0x20, 0x03, 0x28, 0x03, 0x52,
	0x0b, 0x70, 0x65, 0x65, 0x72, 0x73, 0x65, 0x72, 0x76, 0x65, 0x72, 0x73, 0x22, 0x2e, 0x0a, 0x0c,
	0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x1e, 0x0a, 0x04,
	0x75, 0x73, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0a, 0x2e, 0x63, 0x68, 0x61,
	0x74, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x52, 0x04, 0x75, 0x73, 0x65, 0x72, 0x22, 0x2f, 0x0a, 0x0d,
	0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x1e, 0x0a,
	0x04, 0x75, 0x73, 0x65, 0x72, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0a, 0x2e, 0x63, 0x68,
	0x61, 0x74, 0x2e, 0x55, 0x73, 0x65, 0x72, 0x52, 0x04, 0x75, 0x73, 0x65, 0x72, 0x22, 0x35, 0x0a,
	0x0d, 0x4c, 0x6f, 0x67, 0x6f, 0x75, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x12, 0x24,
	0x0a, 0x06, 0x6c, 0x6f, 0x67, 0x6f, 0x75, 0x74, 0x18, 0x01, 0x20, 0x01, 0x28, 0x0b, 0x32, 0x0c,
	0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x4c, 0x6f, 0x67, 0x6f, 0x75, 0x74, 0x52, 0x06, 0x6c, 0x6f,
	0x67, 0x6f, 0x75, 0x74, 0x22, 0x28, 0x0a, 0x0e, 0x4c, 0x6f, 0x67, 0x6f, 0x75, 0x74, 0x52, 0x65,
	0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x12, 0x16, 0x0a, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73,
	0x18, 0x01, 0x20, 0x01, 0x28, 0x08, 0x52, 0x06, 0x73, 0x74, 0x61, 0x74, 0x75, 0x73, 0x32, 0x98,
	0x01, 0x0a, 0x0b, 0x43, 0x68, 0x61, 0x74, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x46,
	0x0a, 0x0d, 0x4a, 0x6f, 0x69, 0x6e, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x43, 0x68, 0x61, 0x74, 0x12,
	0x16, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x47, 0x72, 0x6f, 0x75, 0x70, 0x43, 0x68, 0x61, 0x74,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x17, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x47,
	0x72, 0x6f, 0x75, 0x70, 0x43, 0x68, 0x61, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65,
	0x22, 0x00, 0x28, 0x01, 0x30, 0x01, 0x12, 0x41, 0x0a, 0x0a, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72,
	0x56, 0x69, 0x65, 0x77, 0x12, 0x17, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x53, 0x65, 0x72, 0x76,
	0x65, 0x72, 0x56, 0x69, 0x65, 0x77, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x18, 0x2e,
	0x63, 0x68, 0x61, 0x74, 0x2e, 0x53, 0x65, 0x72, 0x76, 0x65, 0x72, 0x56, 0x69, 0x65, 0x77, 0x52,
	0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x32, 0x78, 0x0a, 0x0b, 0x41, 0x75, 0x74,
	0x68, 0x53, 0x65, 0x72, 0x76, 0x69, 0x63, 0x65, 0x12, 0x32, 0x0a, 0x05, 0x4c, 0x6f, 0x67, 0x69,
	0x6e, 0x12, 0x12, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x4c, 0x6f, 0x67, 0x69, 0x6e, 0x52, 0x65,
	0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x13, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x4c, 0x6f, 0x67,
	0x69, 0x6e, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x35, 0x0a, 0x06,
	0x4c, 0x6f, 0x67, 0x6f, 0x75, 0x74, 0x12, 0x13, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x4c, 0x6f,
	0x67, 0x6f, 0x75, 0x74, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x14, 0x2e, 0x63, 0x68,
	0x61, 0x74, 0x2e, 0x4c, 0x6f, 0x67, 0x6f, 0x75, 0x74, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73,
	0x65, 0x22, 0x00, 0x32, 0xe0, 0x02, 0x0a, 0x0b, 0x52, 0x61, 0x66, 0x74, 0x53, 0x65, 0x72, 0x76,
	0x69, 0x63, 0x65, 0x12, 0x4a, 0x0a, 0x0d, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x45, 0x6e, 0x74,
	0x72, 0x69, 0x65, 0x73, 0x12, 0x1a, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x41, 0x70, 0x70, 0x65,
	0x6e, 0x64, 0x45, 0x6e, 0x74, 0x72, 0x69, 0x65, 0x73, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74,
	0x1a, 0x1b, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x41, 0x70, 0x70, 0x65, 0x6e, 0x64, 0x45, 0x6e,
	0x74, 0x72, 0x69, 0x65, 0x73, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12,
	0x44, 0x0a, 0x0b, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x56, 0x6f, 0x74, 0x65, 0x12, 0x18,
	0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x56, 0x6f, 0x74,
	0x65, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x19, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e,
	0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x56, 0x6f, 0x74, 0x65, 0x52, 0x65, 0x73, 0x70, 0x6f,
	0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x4a, 0x0a, 0x0d, 0x46, 0x6f, 0x72, 0x77, 0x61, 0x72, 0x64,
	0x4c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x12, 0x1a, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x46, 0x6f,
	0x72, 0x77, 0x61, 0x72, 0x64, 0x4c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x52, 0x65, 0x71, 0x75, 0x65,
	0x73, 0x74, 0x1a, 0x1b, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x46, 0x6f, 0x72, 0x77, 0x61, 0x72,
	0x64, 0x4c, 0x65, 0x61, 0x64, 0x65, 0x72, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22,
	0x00, 0x12, 0x38, 0x0a, 0x0b, 0x47, 0x65, 0x74, 0x54, 0x65, 0x6d, 0x70, 0x4c, 0x6f, 0x67, 0x73,
	0x12, 0x12, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x4d, 0x65, 0x72, 0x67, 0x65, 0x52, 0x65, 0x71,
	0x75, 0x65, 0x73, 0x74, 0x1a, 0x13, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x4d, 0x65, 0x72, 0x67,
	0x65, 0x52, 0x65, 0x73, 0x70, 0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x12, 0x39, 0x0a, 0x08, 0x4c,
	0x6f, 0x63, 0x61, 0x6c, 0x41, 0x45, 0x73, 0x12, 0x14, 0x2e, 0x63, 0x68, 0x61, 0x74, 0x2e, 0x4c,
	0x6f, 0x63, 0x61, 0x6c, 0x41, 0x45, 0x52, 0x65, 0x71, 0x75, 0x65, 0x73, 0x74, 0x1a, 0x15, 0x2e,
	0x63, 0x68, 0x61, 0x74, 0x2e, 0x4c, 0x6f, 0x63, 0x61, 0x6c, 0x41, 0x45, 0x52, 0x65, 0x73, 0x70,
	0x6f, 0x6e, 0x73, 0x65, 0x22, 0x00, 0x42, 0x06, 0x5a, 0x04, 0x2e, 0x2f, 0x70, 0x62, 0x62, 0x06,
	0x70, 0x72, 0x6f, 0x74, 0x6f, 0x33,
}

var (
	file_chat_service_proto_rawDescOnce sync.Once
	file_chat_service_proto_rawDescData = file_chat_service_proto_rawDesc
)

func file_chat_service_proto_rawDescGZIP() []byte {
	file_chat_service_proto_rawDescOnce.Do(func() {
		file_chat_service_proto_rawDescData = protoimpl.X.CompressGZIP(file_chat_service_proto_rawDescData)
	})
	return file_chat_service_proto_rawDescData
}

var file_chat_service_proto_msgTypes = make([]protoimpl.MessageInfo, 8)
var file_chat_service_proto_goTypes = []interface{}{
	(*GroupChatRequest)(nil),      // 0: chat.GroupChatRequest
	(*GroupChatResponse)(nil),     // 1: chat.GroupChatResponse
	(*ServerViewRequest)(nil),     // 2: chat.ServerViewRequest
	(*ServerViewResponse)(nil),    // 3: chat.ServerViewResponse
	(*LoginRequest)(nil),          // 4: chat.LoginRequest
	(*LoginResponse)(nil),         // 5: chat.LoginResponse
	(*LogoutRequest)(nil),         // 6: chat.LogoutRequest
	(*LogoutResponse)(nil),        // 7: chat.LogoutResponse
	(*AppendChat)(nil),            // 8: chat.AppendChat
	(*LikeMessage)(nil),           // 9: chat.LikeMessage
	(*UnLikeMessage)(nil),         // 10: chat.UnLikeMessage
	(*PrintChat)(nil),             // 11: chat.PrintChat
	(*Logout)(nil),                // 12: chat.Logout
	(*JoinChat)(nil),              // 13: chat.JoinChat
	(*Group)(nil),                 // 14: chat.Group
	(*User)(nil),                  // 15: chat.User
	(*AppendEntriesRequest)(nil),  // 16: chat.AppendEntriesRequest
	(*RequestVoteRequest)(nil),    // 17: chat.RequestVoteRequest
	(*ForwardLeaderRequest)(nil),  // 18: chat.ForwardLeaderRequest
	(*MergeRequest)(nil),          // 19: chat.MergeRequest
	(*LocalAERequest)(nil),        // 20: chat.LocalAERequest
	(*AppendEntriesResponse)(nil), // 21: chat.AppendEntriesResponse
	(*RequestVoteResponse)(nil),   // 22: chat.RequestVoteResponse
	(*ForwardLeaderResponse)(nil), // 23: chat.ForwardLeaderResponse
	(*MergeResponse)(nil),         // 24: chat.MergeResponse
	(*LocalAEResponse)(nil),       // 25: chat.LocalAEResponse
}
var file_chat_service_proto_depIdxs = []int32{
	8,  // 0: chat.GroupChatRequest.append:type_name -> chat.AppendChat
	9,  // 1: chat.GroupChatRequest.like:type_name -> chat.LikeMessage
	10, // 2: chat.GroupChatRequest.unlike:type_name -> chat.UnLikeMessage
	11, // 3: chat.GroupChatRequest.print:type_name -> chat.PrintChat
	12, // 4: chat.GroupChatRequest.logout:type_name -> chat.Logout
	13, // 5: chat.GroupChatRequest.joinchat:type_name -> chat.JoinChat
	2,  // 6: chat.GroupChatRequest.serverview:type_name -> chat.ServerViewRequest
	14, // 7: chat.GroupChatResponse.group:type_name -> chat.Group
	15, // 8: chat.LoginRequest.user:type_name -> chat.User
	15, // 9: chat.LoginResponse.user:type_name -> chat.User
	12, // 10: chat.LogoutRequest.logout:type_name -> chat.Logout
	0,  // 11: chat.ChatService.JoinGroupChat:input_type -> chat.GroupChatRequest
	2,  // 12: chat.ChatService.ServerView:input_type -> chat.ServerViewRequest
	4,  // 13: chat.AuthService.Login:input_type -> chat.LoginRequest
	6,  // 14: chat.AuthService.Logout:input_type -> chat.LogoutRequest
	16, // 15: chat.RaftService.AppendEntries:input_type -> chat.AppendEntriesRequest
	17, // 16: chat.RaftService.RequestVote:input_type -> chat.RequestVoteRequest
	18, // 17: chat.RaftService.ForwardLeader:input_type -> chat.ForwardLeaderRequest
	19, // 18: chat.RaftService.GetTempLogs:input_type -> chat.MergeRequest
	20, // 19: chat.RaftService.LocalAEs:input_type -> chat.LocalAERequest
	1,  // 20: chat.ChatService.JoinGroupChat:output_type -> chat.GroupChatResponse
	3,  // 21: chat.ChatService.ServerView:output_type -> chat.ServerViewResponse
	5,  // 22: chat.AuthService.Login:output_type -> chat.LoginResponse
	7,  // 23: chat.AuthService.Logout:output_type -> chat.LogoutResponse
	21, // 24: chat.RaftService.AppendEntries:output_type -> chat.AppendEntriesResponse
	22, // 25: chat.RaftService.RequestVote:output_type -> chat.RequestVoteResponse
	23, // 26: chat.RaftService.ForwardLeader:output_type -> chat.ForwardLeaderResponse
	24, // 27: chat.RaftService.GetTempLogs:output_type -> chat.MergeResponse
	25, // 28: chat.RaftService.LocalAEs:output_type -> chat.LocalAEResponse
	20, // [20:29] is the sub-list for method output_type
	11, // [11:20] is the sub-list for method input_type
	11, // [11:11] is the sub-list for extension type_name
	11, // [11:11] is the sub-list for extension extendee
	0,  // [0:11] is the sub-list for field type_name
}

func init() { file_chat_service_proto_init() }
func file_chat_service_proto_init() {
	if File_chat_service_proto != nil {
		return
	}
	file_chat_proto_init()
	file_user_proto_init()
	file_raft_proto_init()
	if !protoimpl.UnsafeEnabled {
		file_chat_service_proto_msgTypes[0].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GroupChatRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_chat_service_proto_msgTypes[1].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*GroupChatResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_chat_service_proto_msgTypes[2].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ServerViewRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_chat_service_proto_msgTypes[3].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*ServerViewResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_chat_service_proto_msgTypes[4].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LoginRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_chat_service_proto_msgTypes[5].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LoginResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_chat_service_proto_msgTypes[6].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LogoutRequest); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
		file_chat_service_proto_msgTypes[7].Exporter = func(v interface{}, i int) interface{} {
			switch v := v.(*LogoutResponse); i {
			case 0:
				return &v.state
			case 1:
				return &v.sizeCache
			case 2:
				return &v.unknownFields
			default:
				return nil
			}
		}
	}
	file_chat_service_proto_msgTypes[0].OneofWrappers = []interface{}{
		(*GroupChatRequest_Append)(nil),
		(*GroupChatRequest_Like)(nil),
		(*GroupChatRequest_Unlike)(nil),
		(*GroupChatRequest_Print)(nil),
		(*GroupChatRequest_Logout)(nil),
		(*GroupChatRequest_Joinchat)(nil),
		(*GroupChatRequest_Serverview)(nil),
	}
	type x struct{}
	out := protoimpl.TypeBuilder{
		File: protoimpl.DescBuilder{
			GoPackagePath: reflect.TypeOf(x{}).PkgPath(),
			RawDescriptor: file_chat_service_proto_rawDesc,
			NumEnums:      0,
			NumMessages:   8,
			NumExtensions: 0,
			NumServices:   3,
		},
		GoTypes:           file_chat_service_proto_goTypes,
		DependencyIndexes: file_chat_service_proto_depIdxs,
		MessageInfos:      file_chat_service_proto_msgTypes,
	}.Build()
	File_chat_service_proto = out.File
	file_chat_service_proto_rawDesc = nil
	file_chat_service_proto_goTypes = nil
	file_chat_service_proto_depIdxs = nil
}
