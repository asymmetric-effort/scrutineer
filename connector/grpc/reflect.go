package grpc

import (
	"context"
	"fmt"
	"io"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// reflectionClient wraps the gRPC server reflection protocol to discover
// services and methods at runtime.
type reflectionClient struct {
	conn *grpc.ClientConn
}

// newReflectionClient creates a new reflection client for the given connection.
func newReflectionClient(conn *grpc.ClientConn) *reflectionClient {
	return &reflectionClient{conn: conn}
}

// resolveMethod resolves a service/method pair using gRPC server reflection.
// It returns the full method path and input/output message descriptors.
func (r *reflectionClient) resolveMethod(ctx context.Context, service, method string) (
	fullMethod string,
	inputDesc protoreflect.MessageDescriptor,
	outputDesc protoreflect.MessageDescriptor,
	err error,
) {
	fullMethod = fmt.Sprintf("/%s/%s", service, method)

	client := grpc_reflection_v1.NewServerReflectionClient(r.conn)
	stream, err := client.ServerReflectionInfo(ctx)
	if err != nil {
		return "", nil, nil, err
	}
	defer func() { _ = stream.CloseSend() }()

	// Request file descriptor for the service.
	if err := stream.Send(&grpc_reflection_v1.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1.ServerReflectionRequest_FileContainingSymbol{
			FileContainingSymbol: service,
		},
	}); err != nil {
		return "", nil, nil, err
	}

	resp, err := stream.Recv()
	if err != nil {
		return "", nil, nil, err
	}

	fdResp := resp.GetFileDescriptorResponse()
	if fdResp == nil {
		errResp := resp.GetErrorResponse()
		if errResp != nil {
			return "", nil, nil, fmt.Errorf("reflection error: %s", errResp.GetErrorMessage())
		}
		return "", nil, nil, fmt.Errorf("unexpected reflection response")
	}

	// Collect all file descriptors (we may need to resolve dependencies).
	allFiles := make(map[string]*descriptorpb.FileDescriptorProto)
	for _, fdBytes := range fdResp.GetFileDescriptorProto() {
		fd := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(fdBytes, fd); err != nil {
			return "", nil, nil, err
		}
		allFiles[fd.GetName()] = fd
	}

	// Resolve transitive dependencies via reflection.
	if err := resolveTransitiveDeps(allFiles, streamDepFetcher(stream)); err != nil {
		return "", nil, nil, err
	}

	// Build file descriptor set and resolve the method.
	files, err := buildFileDescriptors(allFiles)
	if err != nil {
		return "", nil, nil, err
	}

	inputDesc, outputDesc, err = findMethod(files, service, method)
	if err != nil {
		return "", nil, nil, err
	}

	return fullMethod, inputDesc, outputDesc, nil
}

// collectMissingDeps finds file descriptor dependencies not yet present in allFiles.
func collectMissingDeps(allFiles map[string]*descriptorpb.FileDescriptorProto) []string {
	seen := make(map[string]bool)
	for _, fd := range allFiles {
		for _, dep := range fd.GetDependency() {
			if _, ok := allFiles[dep]; !ok {
				if !seen[dep] {
					seen[dep] = true
				}
			}
		}
	}
	result := make([]string, 0, len(seen))
	for dep := range seen {
		result = append(result, dep)
	}
	return result
}

// depFetcher fetches file descriptors by filename. Used for dependency resolution.
type depFetcher func(filename string) ([]*descriptorpb.FileDescriptorProto, error)

// streamDepFetcher creates a depFetcher that uses a reflection stream.
func streamDepFetcher(stream grpc_reflection_v1.ServerReflection_ServerReflectionInfoClient) depFetcher {
	return func(filename string) ([]*descriptorpb.FileDescriptorProto, error) {
		return fetchFileByName(stream, filename)
	}
}

// fetchFileByName requests a file descriptor by filename via the reflection stream.
func fetchFileByName(stream grpc_reflection_v1.ServerReflection_ServerReflectionInfoClient, filename string) ([]*descriptorpb.FileDescriptorProto, error) {
	if err := stream.Send(&grpc_reflection_v1.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1.ServerReflectionRequest_FileByFilename{FileByFilename: filename},
	}); err != nil {
		return nil, err
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}

	fdResp := resp.GetFileDescriptorResponse()
	if fdResp == nil {
		return nil, nil
	}

	var results []*descriptorpb.FileDescriptorProto
	for _, fdBytes := range fdResp.GetFileDescriptorProto() {
		fd := &descriptorpb.FileDescriptorProto{}
		if err := proto.Unmarshal(fdBytes, fd); err != nil {
			return nil, err
		}
		results = append(results, fd)
	}
	return results, nil
}

// resolveTransitiveDeps iteratively resolves missing dependencies using the fetcher.
func resolveTransitiveDeps(allFiles map[string]*descriptorpb.FileDescriptorProto, fetch depFetcher) error {
	for {
		missing := collectMissingDeps(allFiles)
		if len(missing) == 0 {
			return nil
		}

		for _, filename := range missing {
			fds, err := fetch(filename)
			if err != nil {
				return err
			}
			for _, fd := range fds {
				allFiles[fd.GetName()] = fd
			}
		}
	}
}

// listServices lists all services available via reflection.
func (r *reflectionClient) listServices(ctx context.Context) ([]string, error) {
	client := grpc_reflection_v1.NewServerReflectionClient(r.conn)
	stream, err := client.ServerReflectionInfo(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = stream.CloseSend() }()

	if err := stream.Send(&grpc_reflection_v1.ServerReflectionRequest{
		MessageRequest: &grpc_reflection_v1.ServerReflectionRequest_ListServices{ListServices: ""},
	}); err != nil {
		return nil, err
	}

	resp, err := stream.Recv()
	if err != nil {
		return nil, err
	}

	listResp := resp.GetListServicesResponse()
	if listResp == nil {
		return nil, fmt.Errorf("unexpected response type")
	}

	var services []string
	for _, svc := range listResp.GetService() {
		services = append(services, svc.GetName())
	}
	return services, nil
}

// buildFileDescriptors creates a protoreflect file registry from descriptors.
func buildFileDescriptors(allFiles map[string]*descriptorpb.FileDescriptorProto) ([]protoreflect.FileDescriptor, error) {
	fds := &descriptorpb.FileDescriptorSet{}
	for _, fd := range allFiles {
		fds.File = append(fds.File, fd)
	}

	files, err := protodesc.NewFiles(fds)
	if err != nil {
		return nil, fmt.Errorf("creating file registry: %w", err)
	}

	var result []protoreflect.FileDescriptor
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		result = append(result, fd)
		return true
	})
	return result, nil
}

// findMethod locates a method in the resolved file descriptors and returns
// the input and output message descriptors.
func findMethod(files []protoreflect.FileDescriptor, service, method string) (
	protoreflect.MessageDescriptor, protoreflect.MessageDescriptor, error,
) {
	// Extract the simple service name (without package prefix) for matching.
	simpleService := service
	if idx := strings.LastIndex(service, "."); idx >= 0 {
		simpleService = service[idx+1:]
	}

	for _, fd := range files {
		svcs := fd.Services()
		for i := 0; i < svcs.Len(); i++ {
			svc := svcs.Get(i)
			svcFullName := string(svc.FullName())
			svcName := string(svc.Name())

			if svcFullName != service && svcName != simpleService {
				continue
			}

			methods := svc.Methods()
			for j := 0; j < methods.Len(); j++ {
				m := methods.Get(j)
				if string(m.Name()) == method {
					return m.Input(), m.Output(), nil
				}
			}
			return nil, nil, fmt.Errorf("method %s not found in service %s", method, service)
		}
	}
	return nil, nil, fmt.Errorf("service %s not found", service)
}

// buildDynamicMessage creates a dynamic protobuf message from a map of fields.
func buildDynamicMessage(desc protoreflect.MessageDescriptor, data map[string]any) (*dynamicpb.Message, error) {
	msg := dynamicpb.NewMessage(desc)
	if data == nil {
		return msg, nil
	}

	fields := desc.Fields()
	for key, val := range data {
		fd := fields.ByName(protoreflect.Name(key))
		if fd == nil {
			// Try by JSON name.
			fd = fields.ByJSONName(key)
		}
		if fd == nil {
			return nil, fmt.Errorf("unknown field: %s", key)
		}
		pv, err := coerceValue(fd, val)
		if err != nil {
			return nil, fmt.Errorf("field %s: %w", key, err)
		}
		msg.Set(fd, pv)
	}
	return msg, nil
}

// coerceValue converts a Go value to a protoreflect.Value based on the field descriptor.
func coerceValue(fd protoreflect.FieldDescriptor, val any) (protoreflect.Value, error) {
	if fd.IsList() {
		return coerceListValue(fd, val)
	}

	if fd.IsMap() {
		return coerceMapValue(fd, val)
	}

	return coerceScalar(fd, val)
}

func coerceInt32(val any) (protoreflect.Value, error) {
	switch v := val.(type) {
	case int:
		return protoreflect.ValueOfInt32(int32(v)), nil
	case int32:
		return protoreflect.ValueOfInt32(v), nil
	case int64:
		return protoreflect.ValueOfInt32(int32(v)), nil
	case float64:
		return protoreflect.ValueOfInt32(int32(v)), nil
	default:
		return protoreflect.Value{}, fmt.Errorf("expected int32, got %T", val)
	}
}

func coerceInt64(val any) (protoreflect.Value, error) {
	switch v := val.(type) {
	case int:
		return protoreflect.ValueOfInt64(int64(v)), nil
	case int32:
		return protoreflect.ValueOfInt64(int64(v)), nil
	case int64:
		return protoreflect.ValueOfInt64(v), nil
	case float64:
		return protoreflect.ValueOfInt64(int64(v)), nil
	default:
		return protoreflect.Value{}, fmt.Errorf("expected int64, got %T", val)
	}
}

func coerceUint32(val any) (protoreflect.Value, error) {
	switch v := val.(type) {
	case int:
		return protoreflect.ValueOfUint32(uint32(v)), nil
	case uint32:
		return protoreflect.ValueOfUint32(v), nil
	case float64:
		return protoreflect.ValueOfUint32(uint32(v)), nil
	default:
		return protoreflect.Value{}, fmt.Errorf("expected uint32, got %T", val)
	}
}

func coerceUint64(val any) (protoreflect.Value, error) {
	switch v := val.(type) {
	case int:
		return protoreflect.ValueOfUint64(uint64(v)), nil
	case uint64:
		return protoreflect.ValueOfUint64(v), nil
	case float64:
		return protoreflect.ValueOfUint64(uint64(v)), nil
	default:
		return protoreflect.Value{}, fmt.Errorf("expected uint64, got %T", val)
	}
}

func coerceEnum(fd protoreflect.FieldDescriptor, val any) (protoreflect.Value, error) {
	switch v := val.(type) {
	case int:
		return protoreflect.ValueOfEnum(protoreflect.EnumNumber(v)), nil
	case float64:
		return protoreflect.ValueOfEnum(protoreflect.EnumNumber(int32(v))), nil
	case string:
		enumDesc := fd.Enum()
		ev := enumDesc.Values().ByName(protoreflect.Name(v))
		if ev == nil {
			return protoreflect.Value{}, fmt.Errorf("unknown enum value: %s", v)
		}
		return protoreflect.ValueOfEnum(ev.Number()), nil
	default:
		return protoreflect.Value{}, fmt.Errorf("expected enum value, got %T", val)
	}
}

func coerceListValue(fd protoreflect.FieldDescriptor, val any) (protoreflect.Value, error) {
	items, ok := val.([]any)
	if !ok {
		return protoreflect.Value{}, fmt.Errorf("expected list, got %T", val)
	}

	list := dynamicpb.NewMessage(fd.ContainingMessage()).NewField(fd).List()
	for i, item := range items {
		// Create a single-element field descriptor to reuse coerceValue logic.
		ev, err := coerceSingleValue(fd, item)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("list element %d: %w", i, err)
		}
		list.Append(ev)
	}
	return protoreflect.ValueOfList(list), nil
}

func coerceMapValue(fd protoreflect.FieldDescriptor, val any) (protoreflect.Value, error) {
	m, ok := val.(map[string]any)
	if !ok {
		return protoreflect.Value{}, fmt.Errorf("expected map, got %T", val)
	}

	mapField := dynamicpb.NewMessage(fd.ContainingMessage()).NewField(fd).Map()
	keyDesc := fd.MapKey()
	valDesc := fd.MapValue()

	for k, v := range m {
		keyVal, err := coerceSingleValue(keyDesc, k)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("map key %q: %w", k, err)
		}
		valVal, err := coerceSingleValue(valDesc, v)
		if err != nil {
			return protoreflect.Value{}, fmt.Errorf("map value for key %q: %w", k, err)
		}
		mapField.Set(keyVal.MapKey(), valVal)
	}
	return protoreflect.ValueOfMap(mapField), nil
}

// coerceSingleValue coerces a scalar value for non-repeated context (used by list/map helpers).
// It delegates to coerceScalar which handles all proto scalar kinds.
func coerceSingleValue(fd protoreflect.FieldDescriptor, val any) (protoreflect.Value, error) {
	return coerceScalar(fd, val)
}

// coerceScalar converts a Go value to a protoreflect.Value for a scalar field.
func coerceScalar(fd protoreflect.FieldDescriptor, val any) (protoreflect.Value, error) {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		b, ok := val.(bool)
		if !ok {
			return protoreflect.Value{}, fmt.Errorf("expected bool, got %T", val)
		}
		return protoreflect.ValueOfBool(b), nil
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return coerceInt32(val)
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return coerceInt64(val)
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return coerceUint32(val)
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return coerceUint64(val)
	case protoreflect.FloatKind:
		switch v := val.(type) {
		case float64:
			return protoreflect.ValueOfFloat32(float32(v)), nil
		case float32:
			return protoreflect.ValueOfFloat32(v), nil
		case int:
			return protoreflect.ValueOfFloat32(float32(v)), nil
		default:
			return protoreflect.Value{}, fmt.Errorf("expected float, got %T", val)
		}
	case protoreflect.DoubleKind:
		switch v := val.(type) {
		case float64:
			return protoreflect.ValueOfFloat64(v), nil
		case float32:
			return protoreflect.ValueOfFloat64(float64(v)), nil
		case int:
			return protoreflect.ValueOfFloat64(float64(v)), nil
		default:
			return protoreflect.Value{}, fmt.Errorf("expected double, got %T", val)
		}
	case protoreflect.StringKind:
		s, ok := val.(string)
		if !ok {
			return protoreflect.Value{}, fmt.Errorf("expected string, got %T", val)
		}
		return protoreflect.ValueOfString(s), nil
	case protoreflect.BytesKind:
		switch v := val.(type) {
		case string:
			return protoreflect.ValueOfBytes([]byte(v)), nil
		case []byte:
			return protoreflect.ValueOfBytes(v), nil
		default:
			return protoreflect.Value{}, fmt.Errorf("expected bytes, got %T", val)
		}
	case protoreflect.EnumKind:
		return coerceEnum(fd, val)
	default:
		// MessageKind, GroupKind, and any other kinds.
		m, ok := val.(map[string]any)
		if !ok {
			return protoreflect.Value{}, fmt.Errorf("expected map for message, got %T", val)
		}
		sub, err := buildDynamicMessage(fd.Message(), m)
		if err != nil {
			return protoreflect.Value{}, err
		}
		return protoreflect.ValueOfMessage(sub), nil
	}
}

// dynamicMessageToMap converts a dynamic protobuf message to a map[string]any.
func dynamicMessageToMap(msg *dynamicpb.Message) map[string]any {
	result := make(map[string]any)
	desc := msg.Descriptor()
	fields := desc.Fields()

	for i := 0; i < fields.Len(); i++ {
		fd := fields.Get(i)
		if !msg.Has(fd) {
			continue
		}
		val := msg.Get(fd)
		result[string(fd.Name())] = protoValueToGo(fd, val)
	}
	return result
}

// protoValueToGo converts a protoreflect.Value to a Go value.
func protoValueToGo(fd protoreflect.FieldDescriptor, val protoreflect.Value) any {
	if fd.IsList() {
		list := val.List()
		result := make([]any, list.Len())
		for i := 0; i < list.Len(); i++ {
			result[i] = singleProtoValueToGo(fd, list.Get(i))
		}
		return result
	}

	if fd.IsMap() {
		m := val.Map()
		result := make(map[string]any)
		m.Range(func(key protoreflect.MapKey, value protoreflect.Value) bool {
			k := fmt.Sprintf("%v", key.Value().Interface())
			result[k] = singleProtoValueToGo(fd.MapValue(), value)
			return true
		})
		return result
	}

	return singleProtoValueToGo(fd, val)
}

func singleProtoValueToGo(fd protoreflect.FieldDescriptor, val protoreflect.Value) any {
	switch fd.Kind() {
	case protoreflect.BoolKind:
		return val.Bool()
	case protoreflect.Int32Kind, protoreflect.Sint32Kind, protoreflect.Sfixed32Kind:
		return int(val.Int())
	case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
		return val.Int()
	case protoreflect.Uint32Kind, protoreflect.Fixed32Kind:
		return uint32(val.Uint())
	case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
		return val.Uint()
	case protoreflect.FloatKind:
		return float32(val.Float())
	case protoreflect.DoubleKind:
		return val.Float()
	case protoreflect.StringKind:
		return val.String()
	case protoreflect.BytesKind:
		return val.Bytes()
	case protoreflect.EnumKind:
		return int(val.Enum())
	default:
		// MessageKind, GroupKind, and any future kinds.
		return dynamicMessageToMap(val.Message().Interface().(*dynamicpb.Message))
	}
}

// receiveStream reads messages from a gRPC stream until EOF.
func receiveStream(stream grpc.ClientStream, outputDesc protoreflect.MessageDescriptor) ([]*dynamicpb.Message, error) {
	var messages []*dynamicpb.Message
	for {
		msg := dynamicpb.NewMessage(outputDesc)
		err := stream.RecvMsg(msg)
		if err == io.EOF {
			return messages, nil
		}
		if err != nil {
			return messages, err
		}
		messages = append(messages, msg)
	}
}
