// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             (unknown)
// source: initia/move/v1/query.proto

package movev1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	Query_Module_FullMethodName        = "/initia.move.v1.Query/Module"
	Query_Modules_FullMethodName       = "/initia.move.v1.Query/Modules"
	Query_Resource_FullMethodName      = "/initia.move.v1.Query/Resource"
	Query_Resources_FullMethodName     = "/initia.move.v1.Query/Resources"
	Query_TableInfo_FullMethodName     = "/initia.move.v1.Query/TableInfo"
	Query_TableEntry_FullMethodName    = "/initia.move.v1.Query/TableEntry"
	Query_TableEntries_FullMethodName  = "/initia.move.v1.Query/TableEntries"
	Query_LegacyView_FullMethodName    = "/initia.move.v1.Query/LegacyView"
	Query_View_FullMethodName          = "/initia.move.v1.Query/View"
	Query_ViewBatch_FullMethodName     = "/initia.move.v1.Query/ViewBatch"
	Query_ViewJSON_FullMethodName      = "/initia.move.v1.Query/ViewJSON"
	Query_ViewJSONBatch_FullMethodName = "/initia.move.v1.Query/ViewJSONBatch"
	Query_ScriptABI_FullMethodName     = "/initia.move.v1.Query/ScriptABI"
	Query_Params_FullMethodName        = "/initia.move.v1.Query/Params"
	Query_Metadata_FullMethodName      = "/initia.move.v1.Query/Metadata"
	Query_Denom_FullMethodName         = "/initia.move.v1.Query/Denom"
)

// QueryClient is the client API for Query service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type QueryClient interface {
	// Module gets the module info
	Module(ctx context.Context, in *QueryModuleRequest, opts ...grpc.CallOption) (*QueryModuleResponse, error)
	// Modules gets the module infos
	Modules(ctx context.Context, in *QueryModulesRequest, opts ...grpc.CallOption) (*QueryModulesResponse, error)
	// Resource gets the module info
	Resource(ctx context.Context, in *QueryResourceRequest, opts ...grpc.CallOption) (*QueryResourceResponse, error)
	// Resources gets the module infos
	Resources(ctx context.Context, in *QueryResourcesRequest, opts ...grpc.CallOption) (*QueryResourcesResponse, error)
	// Query table info of the given address
	TableInfo(ctx context.Context, in *QueryTableInfoRequest, opts ...grpc.CallOption) (*QueryTableInfoResponse, error)
	// Query table entry of the given key
	TableEntry(ctx context.Context, in *QueryTableEntryRequest, opts ...grpc.CallOption) (*QueryTableEntryResponse, error)
	// Query table entries with pagination
	TableEntries(ctx context.Context, in *QueryTableEntriesRequest, opts ...grpc.CallOption) (*QueryTableEntriesResponse, error)
	// Deprecated: Do not use.
	// Deprecated: Use Query/ViewJSON or Query/ViewJSONBatch
	// LegacyView execute view function and return the view result.
	LegacyView(ctx context.Context, in *QueryLegacyViewRequest, opts ...grpc.CallOption) (*QueryLegacyViewResponse, error)
	// Deprecated: Use Query/ViewJSON or Query/ViewJSONBatch
	// View execute view function and return the view result
	View(ctx context.Context, in *QueryViewRequest, opts ...grpc.CallOption) (*QueryViewResponse, error)
	// Deprecated: Use Query/ViewJSON or Query/ViewJSONBatch
	// ViewBatch execute multiple view functions and return the view results
	ViewBatch(ctx context.Context, in *QueryViewBatchRequest, opts ...grpc.CallOption) (*QueryViewBatchResponse, error)
	// ViewJSON execute view function with json arguments and return the view result
	ViewJSON(ctx context.Context, in *QueryViewJSONRequest, opts ...grpc.CallOption) (*QueryViewJSONResponse, error)
	// ViewJSONBatch execute multiple view functions with json arguments and return the view results
	ViewJSONBatch(ctx context.Context, in *QueryViewJSONBatchRequest, opts ...grpc.CallOption) (*QueryViewJSONBatchResponse, error)
	// ScriptABI decode script bytes into ABI
	ScriptABI(ctx context.Context, in *QueryScriptABIRequest, opts ...grpc.CallOption) (*QueryScriptABIResponse, error)
	// Params queries all parameters.
	Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error)
	// Metadata converts metadata to denom
	Metadata(ctx context.Context, in *QueryMetadataRequest, opts ...grpc.CallOption) (*QueryMetadataResponse, error)
	// Denom converts denom to metadata
	Denom(ctx context.Context, in *QueryDenomRequest, opts ...grpc.CallOption) (*QueryDenomResponse, error)
}

type queryClient struct {
	cc grpc.ClientConnInterface
}

func NewQueryClient(cc grpc.ClientConnInterface) QueryClient {
	return &queryClient{cc}
}

func (c *queryClient) Module(ctx context.Context, in *QueryModuleRequest, opts ...grpc.CallOption) (*QueryModuleResponse, error) {
	out := new(QueryModuleResponse)
	err := c.cc.Invoke(ctx, Query_Module_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Modules(ctx context.Context, in *QueryModulesRequest, opts ...grpc.CallOption) (*QueryModulesResponse, error) {
	out := new(QueryModulesResponse)
	err := c.cc.Invoke(ctx, Query_Modules_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Resource(ctx context.Context, in *QueryResourceRequest, opts ...grpc.CallOption) (*QueryResourceResponse, error) {
	out := new(QueryResourceResponse)
	err := c.cc.Invoke(ctx, Query_Resource_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Resources(ctx context.Context, in *QueryResourcesRequest, opts ...grpc.CallOption) (*QueryResourcesResponse, error) {
	out := new(QueryResourcesResponse)
	err := c.cc.Invoke(ctx, Query_Resources_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) TableInfo(ctx context.Context, in *QueryTableInfoRequest, opts ...grpc.CallOption) (*QueryTableInfoResponse, error) {
	out := new(QueryTableInfoResponse)
	err := c.cc.Invoke(ctx, Query_TableInfo_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) TableEntry(ctx context.Context, in *QueryTableEntryRequest, opts ...grpc.CallOption) (*QueryTableEntryResponse, error) {
	out := new(QueryTableEntryResponse)
	err := c.cc.Invoke(ctx, Query_TableEntry_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) TableEntries(ctx context.Context, in *QueryTableEntriesRequest, opts ...grpc.CallOption) (*QueryTableEntriesResponse, error) {
	out := new(QueryTableEntriesResponse)
	err := c.cc.Invoke(ctx, Query_TableEntries_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// Deprecated: Do not use.
func (c *queryClient) LegacyView(ctx context.Context, in *QueryLegacyViewRequest, opts ...grpc.CallOption) (*QueryLegacyViewResponse, error) {
	out := new(QueryLegacyViewResponse)
	err := c.cc.Invoke(ctx, Query_LegacyView_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) View(ctx context.Context, in *QueryViewRequest, opts ...grpc.CallOption) (*QueryViewResponse, error) {
	out := new(QueryViewResponse)
	err := c.cc.Invoke(ctx, Query_View_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) ViewBatch(ctx context.Context, in *QueryViewBatchRequest, opts ...grpc.CallOption) (*QueryViewBatchResponse, error) {
	out := new(QueryViewBatchResponse)
	err := c.cc.Invoke(ctx, Query_ViewBatch_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) ViewJSON(ctx context.Context, in *QueryViewJSONRequest, opts ...grpc.CallOption) (*QueryViewJSONResponse, error) {
	out := new(QueryViewJSONResponse)
	err := c.cc.Invoke(ctx, Query_ViewJSON_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) ViewJSONBatch(ctx context.Context, in *QueryViewJSONBatchRequest, opts ...grpc.CallOption) (*QueryViewJSONBatchResponse, error) {
	out := new(QueryViewJSONBatchResponse)
	err := c.cc.Invoke(ctx, Query_ViewJSONBatch_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) ScriptABI(ctx context.Context, in *QueryScriptABIRequest, opts ...grpc.CallOption) (*QueryScriptABIResponse, error) {
	out := new(QueryScriptABIResponse)
	err := c.cc.Invoke(ctx, Query_ScriptABI_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Params(ctx context.Context, in *QueryParamsRequest, opts ...grpc.CallOption) (*QueryParamsResponse, error) {
	out := new(QueryParamsResponse)
	err := c.cc.Invoke(ctx, Query_Params_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Metadata(ctx context.Context, in *QueryMetadataRequest, opts ...grpc.CallOption) (*QueryMetadataResponse, error) {
	out := new(QueryMetadataResponse)
	err := c.cc.Invoke(ctx, Query_Metadata_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *queryClient) Denom(ctx context.Context, in *QueryDenomRequest, opts ...grpc.CallOption) (*QueryDenomResponse, error) {
	out := new(QueryDenomResponse)
	err := c.cc.Invoke(ctx, Query_Denom_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// QueryServer is the server API for Query service.
// All implementations must embed UnimplementedQueryServer
// for forward compatibility
type QueryServer interface {
	// Module gets the module info
	Module(context.Context, *QueryModuleRequest) (*QueryModuleResponse, error)
	// Modules gets the module infos
	Modules(context.Context, *QueryModulesRequest) (*QueryModulesResponse, error)
	// Resource gets the module info
	Resource(context.Context, *QueryResourceRequest) (*QueryResourceResponse, error)
	// Resources gets the module infos
	Resources(context.Context, *QueryResourcesRequest) (*QueryResourcesResponse, error)
	// Query table info of the given address
	TableInfo(context.Context, *QueryTableInfoRequest) (*QueryTableInfoResponse, error)
	// Query table entry of the given key
	TableEntry(context.Context, *QueryTableEntryRequest) (*QueryTableEntryResponse, error)
	// Query table entries with pagination
	TableEntries(context.Context, *QueryTableEntriesRequest) (*QueryTableEntriesResponse, error)
	// Deprecated: Do not use.
	// Deprecated: Use Query/ViewJSON or Query/ViewJSONBatch
	// LegacyView execute view function and return the view result.
	LegacyView(context.Context, *QueryLegacyViewRequest) (*QueryLegacyViewResponse, error)
	// Deprecated: Use Query/ViewJSON or Query/ViewJSONBatch
	// View execute view function and return the view result
	View(context.Context, *QueryViewRequest) (*QueryViewResponse, error)
	// Deprecated: Use Query/ViewJSON or Query/ViewJSONBatch
	// ViewBatch execute multiple view functions and return the view results
	ViewBatch(context.Context, *QueryViewBatchRequest) (*QueryViewBatchResponse, error)
	// ViewJSON execute view function with json arguments and return the view result
	ViewJSON(context.Context, *QueryViewJSONRequest) (*QueryViewJSONResponse, error)
	// ViewJSONBatch execute multiple view functions with json arguments and return the view results
	ViewJSONBatch(context.Context, *QueryViewJSONBatchRequest) (*QueryViewJSONBatchResponse, error)
	// ScriptABI decode script bytes into ABI
	ScriptABI(context.Context, *QueryScriptABIRequest) (*QueryScriptABIResponse, error)
	// Params queries all parameters.
	Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error)
	// Metadata converts metadata to denom
	Metadata(context.Context, *QueryMetadataRequest) (*QueryMetadataResponse, error)
	// Denom converts denom to metadata
	Denom(context.Context, *QueryDenomRequest) (*QueryDenomResponse, error)
	mustEmbedUnimplementedQueryServer()
}

// UnimplementedQueryServer must be embedded to have forward compatible implementations.
type UnimplementedQueryServer struct {
}

func (UnimplementedQueryServer) Module(context.Context, *QueryModuleRequest) (*QueryModuleResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Module not implemented")
}
func (UnimplementedQueryServer) Modules(context.Context, *QueryModulesRequest) (*QueryModulesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Modules not implemented")
}
func (UnimplementedQueryServer) Resource(context.Context, *QueryResourceRequest) (*QueryResourceResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Resource not implemented")
}
func (UnimplementedQueryServer) Resources(context.Context, *QueryResourcesRequest) (*QueryResourcesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Resources not implemented")
}
func (UnimplementedQueryServer) TableInfo(context.Context, *QueryTableInfoRequest) (*QueryTableInfoResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TableInfo not implemented")
}
func (UnimplementedQueryServer) TableEntry(context.Context, *QueryTableEntryRequest) (*QueryTableEntryResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TableEntry not implemented")
}
func (UnimplementedQueryServer) TableEntries(context.Context, *QueryTableEntriesRequest) (*QueryTableEntriesResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method TableEntries not implemented")
}
func (UnimplementedQueryServer) LegacyView(context.Context, *QueryLegacyViewRequest) (*QueryLegacyViewResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method LegacyView not implemented")
}
func (UnimplementedQueryServer) View(context.Context, *QueryViewRequest) (*QueryViewResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method View not implemented")
}
func (UnimplementedQueryServer) ViewBatch(context.Context, *QueryViewBatchRequest) (*QueryViewBatchResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ViewBatch not implemented")
}
func (UnimplementedQueryServer) ViewJSON(context.Context, *QueryViewJSONRequest) (*QueryViewJSONResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ViewJSON not implemented")
}
func (UnimplementedQueryServer) ViewJSONBatch(context.Context, *QueryViewJSONBatchRequest) (*QueryViewJSONBatchResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ViewJSONBatch not implemented")
}
func (UnimplementedQueryServer) ScriptABI(context.Context, *QueryScriptABIRequest) (*QueryScriptABIResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ScriptABI not implemented")
}
func (UnimplementedQueryServer) Params(context.Context, *QueryParamsRequest) (*QueryParamsResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Params not implemented")
}
func (UnimplementedQueryServer) Metadata(context.Context, *QueryMetadataRequest) (*QueryMetadataResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Metadata not implemented")
}
func (UnimplementedQueryServer) Denom(context.Context, *QueryDenomRequest) (*QueryDenomResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Denom not implemented")
}
func (UnimplementedQueryServer) mustEmbedUnimplementedQueryServer() {}

// UnsafeQueryServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to QueryServer will
// result in compilation errors.
type UnsafeQueryServer interface {
	mustEmbedUnimplementedQueryServer()
}

func RegisterQueryServer(s grpc.ServiceRegistrar, srv QueryServer) {
	s.RegisterService(&Query_ServiceDesc, srv)
}

func _Query_Module_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryModuleRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Module(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Module_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Module(ctx, req.(*QueryModuleRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Modules_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryModulesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Modules(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Modules_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Modules(ctx, req.(*QueryModulesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Resource_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryResourceRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Resource(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Resource_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Resource(ctx, req.(*QueryResourceRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Resources_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryResourcesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Resources(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Resources_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Resources(ctx, req.(*QueryResourcesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_TableInfo_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryTableInfoRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).TableInfo(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_TableInfo_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).TableInfo(ctx, req.(*QueryTableInfoRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_TableEntry_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryTableEntryRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).TableEntry(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_TableEntry_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).TableEntry(ctx, req.(*QueryTableEntryRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_TableEntries_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryTableEntriesRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).TableEntries(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_TableEntries_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).TableEntries(ctx, req.(*QueryTableEntriesRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_LegacyView_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryLegacyViewRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).LegacyView(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_LegacyView_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).LegacyView(ctx, req.(*QueryLegacyViewRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_View_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryViewRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).View(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_View_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).View(ctx, req.(*QueryViewRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_ViewBatch_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryViewBatchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ViewBatch(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_ViewBatch_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ViewBatch(ctx, req.(*QueryViewBatchRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_ViewJSON_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryViewJSONRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ViewJSON(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_ViewJSON_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ViewJSON(ctx, req.(*QueryViewJSONRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_ViewJSONBatch_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryViewJSONBatchRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ViewJSONBatch(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_ViewJSONBatch_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ViewJSONBatch(ctx, req.(*QueryViewJSONBatchRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_ScriptABI_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryScriptABIRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).ScriptABI(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_ScriptABI_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).ScriptABI(ctx, req.(*QueryScriptABIRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Params_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryParamsRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Params(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Params_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Params(ctx, req.(*QueryParamsRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Metadata_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryMetadataRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Metadata(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Metadata_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Metadata(ctx, req.(*QueryMetadataRequest))
	}
	return interceptor(ctx, in, info, handler)
}

func _Query_Denom_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(QueryDenomRequest)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(QueryServer).Denom(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Query_Denom_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(QueryServer).Denom(ctx, req.(*QueryDenomRequest))
	}
	return interceptor(ctx, in, info, handler)
}

// Query_ServiceDesc is the grpc.ServiceDesc for Query service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Query_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "initia.move.v1.Query",
	HandlerType: (*QueryServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Module",
			Handler:    _Query_Module_Handler,
		},
		{
			MethodName: "Modules",
			Handler:    _Query_Modules_Handler,
		},
		{
			MethodName: "Resource",
			Handler:    _Query_Resource_Handler,
		},
		{
			MethodName: "Resources",
			Handler:    _Query_Resources_Handler,
		},
		{
			MethodName: "TableInfo",
			Handler:    _Query_TableInfo_Handler,
		},
		{
			MethodName: "TableEntry",
			Handler:    _Query_TableEntry_Handler,
		},
		{
			MethodName: "TableEntries",
			Handler:    _Query_TableEntries_Handler,
		},
		{
			MethodName: "LegacyView",
			Handler:    _Query_LegacyView_Handler,
		},
		{
			MethodName: "View",
			Handler:    _Query_View_Handler,
		},
		{
			MethodName: "ViewBatch",
			Handler:    _Query_ViewBatch_Handler,
		},
		{
			MethodName: "ViewJSON",
			Handler:    _Query_ViewJSON_Handler,
		},
		{
			MethodName: "ViewJSONBatch",
			Handler:    _Query_ViewJSONBatch_Handler,
		},
		{
			MethodName: "ScriptABI",
			Handler:    _Query_ScriptABI_Handler,
		},
		{
			MethodName: "Params",
			Handler:    _Query_Params_Handler,
		},
		{
			MethodName: "Metadata",
			Handler:    _Query_Metadata_Handler,
		},
		{
			MethodName: "Denom",
			Handler:    _Query_Denom_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "initia/move/v1/query.proto",
}
