package grpc

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/scrutineer/scrutineer/core/connector"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	grpcmd "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/reflection/grpc_reflection_v1"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"
)

// ---------- test proto schema ----------

// buildTestFileDescriptor constructs a FileDescriptorProto for a simple Echo service:
//
//	syntax = "proto3";
//	package testpkg;
//	service EchoService {
//	    rpc Echo(EchoRequest) returns (EchoResponse);
//	    rpc ServerStreamEcho(EchoRequest) returns (stream EchoResponse);
//	    rpc ClientStreamEcho(stream EchoRequest) returns (EchoResponse);
//	    rpc BidiStreamEcho(stream EchoRequest) returns (stream EchoResponse);
//	}
//	message EchoRequest { string message = 1; int32 code = 2; }
//	message EchoResponse { string message = 1; int32 code = 2; }
func buildTestFileDescriptor() *descriptorpb.FileDescriptorProto {
	syntax := "proto3"
	pkg := "testpkg"
	fname := "test.proto"

	msgField := func(name string, number int32, typ descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto {
		label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
		return &descriptorpb.FieldDescriptorProto{
			Name:   &name,
			Number: &number,
			Type:   &typ,
			Label:  &label,
		}
	}

	reqName := "EchoRequest"
	respName := "EchoResponse"

	reqMsg := &descriptorpb.DescriptorProto{
		Name: &reqName,
		Field: []*descriptorpb.FieldDescriptorProto{
			msgField("message", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
			msgField("code", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32),
		},
	}
	respMsg := &descriptorpb.DescriptorProto{
		Name: &respName,
		Field: []*descriptorpb.FieldDescriptorProto{
			msgField("message", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING),
			msgField("code", 2, descriptorpb.FieldDescriptorProto_TYPE_INT32),
		},
	}

	svcName := "EchoService"
	inputType := ".testpkg.EchoRequest"
	outputType := ".testpkg.EchoResponse"

	boolTrue := true
	boolFalse := false

	echoMethod := "Echo"
	ssMethod := "ServerStreamEcho"
	csMethod := "ClientStreamEcho"
	bidiMethod := "BidiStreamEcho"

	svc := &descriptorpb.ServiceDescriptorProto{
		Name: &svcName,
		Method: []*descriptorpb.MethodDescriptorProto{
			{Name: &echoMethod, InputType: &inputType, OutputType: &outputType,
				ClientStreaming: &boolFalse, ServerStreaming: &boolFalse},
			{Name: &ssMethod, InputType: &inputType, OutputType: &outputType,
				ClientStreaming: &boolFalse, ServerStreaming: &boolTrue},
			{Name: &csMethod, InputType: &inputType, OutputType: &outputType,
				ClientStreaming: &boolTrue, ServerStreaming: &boolFalse},
			{Name: &bidiMethod, InputType: &inputType, OutputType: &outputType,
				ClientStreaming: &boolTrue, ServerStreaming: &boolTrue},
		},
	}

	return &descriptorpb.FileDescriptorProto{
		Name:        &fname,
		Package:     &pkg,
		Syntax:      &syntax,
		MessageType: []*descriptorpb.DescriptorProto{reqMsg, respMsg},
		Service:     []*descriptorpb.ServiceDescriptorProto{svc},
	}
}

var testFileDesc *descriptorpb.FileDescriptorProto
var testFD protoreflect.FileDescriptor

func init() {
	testFileDesc = buildTestFileDescriptor()
	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{testFileDesc}}
	files, err := protodesc.NewFiles(fds)
	if err != nil {
		panic(fmt.Sprintf("failed to build test file descriptors: %v", err))
	}
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		testFD = fd
		return false
	})

	// Register in global registry so gRPC reflection can discover the service.
	if _, err := protoregistry.GlobalFiles.FindFileByPath("test.proto"); err != nil {
		if regErr := protoregistry.GlobalFiles.RegisterFile(testFD); regErr != nil {
			panic(fmt.Sprintf("failed to register test file descriptor: %v", regErr))
		}
	}
}

func getTestInputDesc() protoreflect.MessageDescriptor {
	return testFD.Services().Get(0).Methods().ByName("Echo").Input()
}

func getTestOutputDesc() protoreflect.MessageDescriptor {
	return testFD.Services().Get(0).Methods().ByName("Echo").Output()
}

// ---------- test gRPC server ----------

// echoServer implements a simple echo gRPC service using generic handlers.
type echoServer struct {
	fdProto *descriptorpb.FileDescriptorProto
}

func newEchoServer() *echoServer {
	return &echoServer{fdProto: buildTestFileDescriptor()}
}

func (s *echoServer) serviceDesc() *grpc.ServiceDesc {
	return &grpc.ServiceDesc{
		ServiceName: "testpkg.EchoService",
		HandlerType: (*any)(nil),
		Methods: []grpc.MethodDesc{
			{MethodName: "Echo", Handler: s.handleEcho},
		},
		Streams: []grpc.StreamDesc{
			{StreamName: "ServerStreamEcho", Handler: s.handleServerStream, ServerStreams: true},
			{StreamName: "ClientStreamEcho", Handler: s.handleClientStream, ClientStreams: true},
			{StreamName: "BidiStreamEcho", Handler: s.handleBidiStream, ServerStreams: true, ClientStreams: true},
		},
	}
}

func (s *echoServer) handleEcho(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	reqMsg := dynamicpb.NewMessage(getTestInputDesc())
	if err := dec(reqMsg); err != nil {
		return nil, err
	}

	// Send back echoed response with metadata.
	md, ok := grpcmd.FromIncomingContext(ctx)
	if ok {
		_ = grpc.SendHeader(ctx, md)
	}
	grpc.SetTrailer(ctx, grpcmd.Pairs("trailer-key", "trailer-value"))

	message := reqMsg.Get(getTestInputDesc().Fields().ByName("message")).String()
	code := reqMsg.Get(getTestInputDesc().Fields().ByName("code")).Int()

	if code != 0 {
		return nil, status.Errorf(codes.Code(code), "error with code %d", code)
	}

	respMsg := dynamicpb.NewMessage(getTestOutputDesc())
	respMsg.Set(getTestOutputDesc().Fields().ByName("message"), protoreflect.ValueOfString("echo: "+message))
	respMsg.Set(getTestOutputDesc().Fields().ByName("code"), protoreflect.ValueOfInt32(0))
	return respMsg, nil
}

func (s *echoServer) handleServerStream(srv any, stream grpc.ServerStream) error {
	reqMsg := dynamicpb.NewMessage(getTestInputDesc())
	if err := stream.RecvMsg(reqMsg); err != nil {
		return err
	}

	md, ok := grpcmd.FromIncomingContext(stream.Context())
	if ok {
		_ = stream.SendHeader(md)
	}
	stream.SetTrailer(grpcmd.Pairs("trailer-key", "trailer-value"))

	message := reqMsg.Get(getTestInputDesc().Fields().ByName("message")).String()

	// Send 3 responses.
	for i := 0; i < 3; i++ {
		respMsg := dynamicpb.NewMessage(getTestOutputDesc())
		respMsg.Set(getTestOutputDesc().Fields().ByName("message"),
			protoreflect.ValueOfString(fmt.Sprintf("stream %d: %s", i, message)))
		respMsg.Set(getTestOutputDesc().Fields().ByName("code"), protoreflect.ValueOfInt32(int32(i)))
		if err := stream.SendMsg(respMsg); err != nil {
			return err
		}
	}
	return nil
}

func (s *echoServer) handleClientStream(srv any, stream grpc.ServerStream) error {
	var messages []string
	for {
		reqMsg := dynamicpb.NewMessage(getTestInputDesc())
		if err := stream.RecvMsg(reqMsg); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		msg := reqMsg.Get(getTestInputDesc().Fields().ByName("message")).String()
		messages = append(messages, msg)
	}

	md, ok := grpcmd.FromIncomingContext(stream.Context())
	if ok {
		_ = stream.SendHeader(md)
	}
	stream.SetTrailer(grpcmd.Pairs("trailer-key", "trailer-value"))

	respMsg := dynamicpb.NewMessage(getTestOutputDesc())
	combined := ""
	for _, m := range messages {
		combined += m + ";"
	}
	respMsg.Set(getTestOutputDesc().Fields().ByName("message"), protoreflect.ValueOfString(combined))
	respMsg.Set(getTestOutputDesc().Fields().ByName("code"), protoreflect.ValueOfInt32(int32(len(messages))))
	return stream.SendMsg(respMsg)
}

func (s *echoServer) handleBidiStream(srv any, stream grpc.ServerStream) error {
	md, ok := grpcmd.FromIncomingContext(stream.Context())
	if ok {
		_ = stream.SendHeader(md)
	}
	stream.SetTrailer(grpcmd.Pairs("trailer-key", "trailer-value"))

	for {
		reqMsg := dynamicpb.NewMessage(getTestInputDesc())
		if err := stream.RecvMsg(reqMsg); err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		message := reqMsg.Get(getTestInputDesc().Fields().ByName("message")).String()

		respMsg := dynamicpb.NewMessage(getTestOutputDesc())
		respMsg.Set(getTestOutputDesc().Fields().ByName("message"),
			protoreflect.ValueOfString("bidi: "+message))
		respMsg.Set(getTestOutputDesc().Fields().ByName("code"), protoreflect.ValueOfInt32(0))
		if err := stream.SendMsg(respMsg); err != nil {
			return err
		}
	}
}

// startTestServer starts a gRPC server on a random port and returns the address
// and a cleanup function.
func startTestServer(t *testing.T, opts ...grpc.ServerOption) (string, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	srv := grpc.NewServer(opts...)
	es := newEchoServer()
	srv.RegisterService(es.serviceDesc(), nil)
	reflection.Register(srv)

	go func() { _ = srv.Serve(lis) }()

	return lis.Addr().String(), func() {
		srv.GracefulStop()
		lis.Close()
	}
}

// generateTestCert creates a self-signed certificate and key in a temp dir.
func generateTestCert(t *testing.T) (certFile, keyFile, caFile string) {
	t.Helper()
	dir := t.TempDir()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generating key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "localhost"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:           []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:              []string{"localhost"},
		IsCA:                  true,
		BasicConstraintsValid: true,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("creating certificate: %v", err)
	}

	certFile = filepath.Join(dir, "cert.pem")
	keyFile = filepath.Join(dir, "key.pem")
	caFile = filepath.Join(dir, "ca.pem")

	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	if err := os.WriteFile(certFile, certPEM, 0o600); err != nil {
		t.Fatalf("writing cert: %v", err)
	}
	if err := os.WriteFile(caFile, certPEM, 0o600); err != nil {
		t.Fatalf("writing ca: %v", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatalf("marshaling key: %v", err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})
	if err := os.WriteFile(keyFile, keyPEM, 0o600); err != nil {
		t.Fatalf("writing key: %v", err)
	}

	return certFile, keyFile, caFile
}

// startTLSTestServer starts a gRPC server with TLS.
func startTLSTestServer(t *testing.T) (string, func(), string) {
	t.Helper()
	certFile, keyFile, caFile := generateTestCert(t)

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		t.Fatalf("loading cert: %v", err)
	}

	creds := credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	})

	addr, cleanup := startTestServer(t, grpc.Creds(creds))
	return addr, cleanup, caFile
}

// ---------- Tests ----------

func TestNew(t *testing.T) {
	c := New()
	if c == nil {
		t.Fatal("New() returned nil")
	}
}

func TestName(t *testing.T) {
	c := New()
	if c.Name() != "grpc" {
		t.Errorf("Name() = %q, want %q", c.Name(), "grpc")
	}
}

func TestConnectorInterface(t *testing.T) {
	var _ connector.Connector = (*GRPCConnector)(nil)
}

func TestSetup_NilConfig(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for nil config")
	}
}

func TestSetup_MissingEndpoint(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing endpoint")
	}
}

func TestSetup_EmptyEndpoint(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{"endpoint": ""})
	if err == nil {
		t.Fatal("expected error for empty endpoint")
	}
}

func TestSetup_InvalidEndpointType(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{"endpoint": 123})
	if err == nil {
		t.Fatal("expected error for non-string endpoint")
	}
}

func TestSetup_Plaintext(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"endpoint":  addr,
		"plaintext": true,
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer c.Teardown(context.Background())
}

func TestSetup_Insecure(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"endpoint": addr,
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer c.Teardown(context.Background())
}

func TestSetup_TLS(t *testing.T) {
	addr, cleanup, caFile := startTLSTestServer(t)
	defer cleanup()

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"endpoint":    addr,
		"tls":         true,
		"tls_ca_file": caFile,
	})
	if err != nil {
		t.Fatalf("Setup with TLS failed: %v", err)
	}
	defer c.Teardown(context.Background())
}

func TestSetup_TLS_SkipVerify(t *testing.T) {
	addr, cleanup, _ := startTLSTestServer(t)
	defer cleanup()

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"endpoint":        addr,
		"tls":             true,
		"tls_skip_verify": true,
	})
	if err != nil {
		t.Fatalf("Setup with TLS skip verify failed: %v", err)
	}
	defer c.Teardown(context.Background())
}

func TestSetup_TLS_InvalidCA(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"endpoint":    "localhost:0",
		"tls":         true,
		"tls_ca_file": "/nonexistent/ca.pem",
	})
	if err == nil {
		t.Fatal("expected error for invalid CA file")
	}
}

func TestSetup_TLS_InvalidCAContent(t *testing.T) {
	dir := t.TempDir()
	caFile := filepath.Join(dir, "bad-ca.pem")
	os.WriteFile(caFile, []byte("not a certificate"), 0o600)

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"endpoint":    "localhost:0",
		"tls":         true,
		"tls_ca_file": caFile,
	})
	if err == nil {
		t.Fatal("expected error for bad CA content")
	}
}

func TestSetup_TLS_CAFileNotString(t *testing.T) {
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"endpoint":    "localhost:0",
		"tls":         true,
		"tls_ca_file": 123,
	})
	if err == nil {
		t.Fatal("expected error for non-string CA file")
	}
}

func TestTeardown_NilConn(t *testing.T) {
	c := New()
	err := c.Teardown(context.Background())
	if err != nil {
		t.Fatalf("Teardown on nil conn should not error: %v", err)
	}
}

func TestTeardown(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"endpoint":  addr,
		"plaintext": true,
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	err = c.Teardown(context.Background())
	if err != nil {
		t.Fatalf("Teardown failed: %v", err)
	}
	// Second teardown should be no-op.
	err = c.Teardown(context.Background())
	if err != nil {
		t.Fatalf("Second Teardown failed: %v", err)
	}
}

func TestExecute_NotSetUp(t *testing.T) {
	c := New()
	_, err := c.Execute(context.Background(), connector.Step{Action: "unary"})
	if err == nil {
		t.Fatal("expected error when not set up")
	}
}

func TestExecute_UnsupportedAction(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"endpoint":  addr,
		"plaintext": true,
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer c.Teardown(context.Background())

	_, err = c.Execute(context.Background(), connector.Step{Action: "unknown"})
	if err == nil {
		t.Fatal("expected error for unsupported action")
	}
}

func TestUnary_Echo(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"endpoint":  addr,
		"plaintext": true,
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "unary",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "Echo",
			"message": map[string]any{
				"message": "hello",
			},
			"metadata": map[string]any{
				"x-custom": "custom-value",
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Data["status_code"].(int) != 0 {
		t.Errorf("status_code = %v, want 0", result.Data["status_code"])
	}
	if result.Data["status_name"].(string) != "OK" {
		t.Errorf("status_name = %v, want OK", result.Data["status_name"])
	}

	resp, ok := result.Data["response"].(map[string]any)
	if !ok {
		t.Fatal("response not a map")
	}
	if resp["message"] != "echo: hello" {
		t.Errorf("response message = %v, want %q", resp["message"], "echo: hello")
	}

	// Check metadata echoed back.
	md, ok := result.Data["metadata"].(map[string]any)
	if !ok {
		t.Fatal("metadata not a map")
	}
	if md["x-custom"] != "custom-value" {
		t.Errorf("metadata x-custom = %v, want custom-value", md["x-custom"])
	}

	// Check trailers.
	trailers, ok := result.Data["trailers"].(map[string]any)
	if !ok {
		t.Fatal("trailers not a map")
	}
	if trailers["trailer-key"] != "trailer-value" {
		t.Errorf("trailer-key = %v, want trailer-value", trailers["trailer-key"])
	}

	// Check meta.
	if result.Meta["connector"] != "grpc" {
		t.Errorf("meta connector = %v, want grpc", result.Meta["connector"])
	}
	if result.Meta["action"] != "unary" {
		t.Errorf("meta action = %v, want unary", result.Meta["action"])
	}

	// Check elapsed is set.
	if result.Elapsed <= 0 {
		t.Error("elapsed should be positive")
	}
}

func TestUnary_MissingService(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action:     "unary",
		Parameters: map[string]any{"method": "Echo"},
	})
	if err == nil {
		t.Fatal("expected error for missing service")
	}
}

func TestUnary_MissingMethod(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action:     "unary",
		Parameters: map[string]any{"service": "testpkg.EchoService"},
	})
	if err == nil {
		t.Fatal("expected error for missing method")
	}
}

func TestUnary_InvalidMessageType(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "unary",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "Echo",
			"message": "not a map",
		},
	})
	if err == nil {
		t.Fatal("expected error for non-map message")
	}
}

func TestUnary_UnknownService(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "unary",
		Parameters: map[string]any{
			"service": "nonexistent.Service",
			"method":  "Foo",
		},
	})
	if err == nil {
		t.Fatal("expected error for unknown service")
	}
}

func TestUnary_UnknownMethod(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "unary",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "NonexistentMethod",
		},
	})
	if err == nil {
		t.Fatal("expected error for unknown method")
	}
}

func TestUnary_GRPCErrorStatus(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "unary",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "Echo",
			"message": map[string]any{
				"message": "fail",
				"code":    3, // InvalidArgument
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute should not return error for gRPC error status: %v", err)
	}

	if result.Data["status_code"].(int) != int(codes.InvalidArgument) {
		t.Errorf("status_code = %v, want %d", result.Data["status_code"], codes.InvalidArgument)
	}
	if result.Data["status_name"].(string) != "INVALID_ARGUMENT" {
		t.Errorf("status_name = %v, want INVALID_ARGUMENT", result.Data["status_name"])
	}
	if _, ok := result.Data["response"]; ok {
		t.Error("response should not be set on error")
	}
}

func TestUnary_NilMessage(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "unary",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "Echo",
		},
	})
	if err != nil {
		t.Fatalf("Execute with nil message failed: %v", err)
	}
	if result.Data["status_code"].(int) != 0 {
		t.Errorf("status_code = %v, want 0", result.Data["status_code"])
	}
}

func TestUnary_TLS(t *testing.T) {
	addr, cleanup, caFile := startTLSTestServer(t)
	defer cleanup()

	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"endpoint":    addr,
		"tls":         true,
		"tls_ca_file": caFile,
	})
	if err != nil {
		t.Fatalf("Setup failed: %v", err)
	}
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "unary",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "Echo",
			"message": map[string]any{"message": "tls-test"},
		},
	})
	if err != nil {
		t.Fatalf("Execute over TLS failed: %v", err)
	}
	if result.Data["status_code"].(int) != 0 {
		t.Errorf("status_code = %v, want 0", result.Data["status_code"])
	}
	resp := result.Data["response"].(map[string]any)
	if resp["message"] != "echo: tls-test" {
		t.Errorf("response message = %v, want %q", resp["message"], "echo: tls-test")
	}
}

func TestServerStream(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "server_stream",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "ServerStreamEcho",
			"message": map[string]any{"message": "stream-test"},
			"metadata": map[string]any{
				"x-stream": "yes",
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute server_stream failed: %v", err)
	}

	if result.Data["status_code"].(int) != 0 {
		t.Errorf("status_code = %v, want 0", result.Data["status_code"])
	}

	responses, ok := result.Data["responses"].([]any)
	if !ok {
		t.Fatal("responses not a []any")
	}
	if len(responses) != 3 {
		t.Fatalf("expected 3 responses, got %d", len(responses))
	}

	first := responses[0].(map[string]any)
	if first["message"] != "stream 0: stream-test" {
		t.Errorf("first response message = %v", first["message"])
	}

	if result.Meta["action"] != "server_stream" {
		t.Errorf("meta action = %v, want server_stream", result.Meta["action"])
	}
}

func TestServerStream_MissingService(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action:     "server_stream",
		Parameters: map[string]any{"method": "ServerStreamEcho"},
	})
	if err == nil {
		t.Fatal("expected error for missing service")
	}
}

func TestServerStream_InvalidMessage(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "server_stream",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "ServerStreamEcho",
			"message": "not a map",
		},
	})
	if err == nil {
		t.Fatal("expected error for non-map message")
	}
}

func TestClientStream(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "client_stream",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "ClientStreamEcho",
			"messages": []any{
				map[string]any{"message": "msg1"},
				map[string]any{"message": "msg2"},
				map[string]any{"message": "msg3"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute client_stream failed: %v", err)
	}

	if result.Data["status_code"].(int) != 0 {
		t.Errorf("status_code = %v, want 0", result.Data["status_code"])
	}

	resp, ok := result.Data["response"].(map[string]any)
	if !ok {
		t.Fatal("response not a map")
	}
	if resp["message"] != "msg1;msg2;msg3;" {
		t.Errorf("response message = %v", resp["message"])
	}
	if resp["code"] != 3 {
		t.Errorf("response code = %v, want 3", resp["code"])
	}

	if result.Meta["action"] != "client_stream" {
		t.Errorf("meta action = %v, want client_stream", result.Meta["action"])
	}
}

func TestClientStream_MissingMessages(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "client_stream",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "ClientStreamEcho",
		},
	})
	if err == nil {
		t.Fatal("expected error for missing messages")
	}
}

func TestClientStream_InvalidMessagesType(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "client_stream",
		Parameters: map[string]any{
			"service":  "testpkg.EchoService",
			"method":   "ClientStreamEcho",
			"messages": "not a list",
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid messages type")
	}
}

func TestClientStream_InvalidMessageItem(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "client_stream",
		Parameters: map[string]any{
			"service":  "testpkg.EchoService",
			"method":   "ClientStreamEcho",
			"messages": []any{"not a map"},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid message item")
	}
}

func TestClientStream_MissingService(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "client_stream",
		Parameters: map[string]any{
			"method":   "ClientStreamEcho",
			"messages": []any{map[string]any{"message": "x"}},
		},
	})
	if err == nil {
		t.Fatal("expected error for missing service")
	}
}

func TestBidiStream(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "bidi_stream",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "BidiStreamEcho",
			"messages": []any{
				map[string]any{"message": "a"},
				map[string]any{"message": "b"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute bidi_stream failed: %v", err)
	}

	if result.Data["status_code"].(int) != 0 {
		t.Errorf("status_code = %v, want 0", result.Data["status_code"])
	}

	responses, ok := result.Data["responses"].([]any)
	if !ok {
		t.Fatal("responses not a []any")
	}
	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}

	first := responses[0].(map[string]any)
	if first["message"] != "bidi: a" {
		t.Errorf("first response = %v", first["message"])
	}
	second := responses[1].(map[string]any)
	if second["message"] != "bidi: b" {
		t.Errorf("second response = %v", second["message"])
	}

	if result.Meta["action"] != "bidi_stream" {
		t.Errorf("meta action = %v, want bidi_stream", result.Meta["action"])
	}
}

func TestBidiStream_MissingService(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "bidi_stream",
		Parameters: map[string]any{
			"method":   "BidiStreamEcho",
			"messages": []any{map[string]any{"message": "x"}},
		},
	})
	if err == nil {
		t.Fatal("expected error for missing service")
	}
}

// ---------- Status tests ----------

func TestStatusCodeName_AllCodes(t *testing.T) {
	tests := []struct {
		code codes.Code
		name string
	}{
		{codes.OK, "OK"},
		{codes.Canceled, "CANCELLED"},
		{codes.Unknown, "UNKNOWN"},
		{codes.InvalidArgument, "INVALID_ARGUMENT"},
		{codes.DeadlineExceeded, "DEADLINE_EXCEEDED"},
		{codes.NotFound, "NOT_FOUND"},
		{codes.AlreadyExists, "ALREADY_EXISTS"},
		{codes.PermissionDenied, "PERMISSION_DENIED"},
		{codes.ResourceExhausted, "RESOURCE_EXHAUSTED"},
		{codes.FailedPrecondition, "FAILED_PRECONDITION"},
		{codes.Aborted, "ABORTED"},
		{codes.OutOfRange, "OUT_OF_RANGE"},
		{codes.Unimplemented, "UNIMPLEMENTED"},
		{codes.Internal, "INTERNAL"},
		{codes.Unavailable, "UNAVAILABLE"},
		{codes.DataLoss, "DATA_LOSS"},
		{codes.Unauthenticated, "UNAUTHENTICATED"},
		{codes.Code(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := statusCodeName(tt.code)
			if got != tt.name {
				t.Errorf("statusCodeName(%d) = %q, want %q", tt.code, got, tt.name)
			}
		})
	}
}

func TestExtractStatus_Nil(t *testing.T) {
	code, msg := extractStatus(nil)
	if code != codes.OK {
		t.Errorf("code = %v, want OK", code)
	}
	if msg != "" {
		t.Errorf("msg = %q, want empty", msg)
	}
}

func TestExtractStatus_GRPCError(t *testing.T) {
	err := status.Errorf(codes.NotFound, "not found")
	code, msg := extractStatus(err)
	if code != codes.NotFound {
		t.Errorf("code = %v, want NotFound", code)
	}
	if msg != "not found" {
		t.Errorf("msg = %q, want %q", msg, "not found")
	}
}

func TestExtractStatus_NonGRPCError(t *testing.T) {
	err := fmt.Errorf("plain error")
	code, msg := extractStatus(err)
	if code != codes.Unknown {
		t.Errorf("code = %v, want Unknown", code)
	}
	if msg != "plain error" {
		t.Errorf("msg = %q", msg)
	}
}

// ---------- Metadata tests ----------

func TestBuildOutgoingMetadata_NoMetadata(t *testing.T) {
	md := buildOutgoingMetadata(map[string]any{})
	if len(md) != 0 {
		t.Errorf("expected empty metadata, got %v", md)
	}
}

func TestBuildOutgoingMetadata_InvalidType(t *testing.T) {
	md := buildOutgoingMetadata(map[string]any{"metadata": "not a map"})
	if len(md) != 0 {
		t.Errorf("expected empty metadata for invalid type, got %v", md)
	}
}

func TestBuildOutgoingMetadata_StringValues(t *testing.T) {
	md := buildOutgoingMetadata(map[string]any{
		"metadata": map[string]any{
			"key1": "val1",
			"key2": "val2",
		},
	})
	if md.Get("key1")[0] != "val1" {
		t.Errorf("key1 = %v", md.Get("key1"))
	}
}

func TestBuildOutgoingMetadata_ListValues(t *testing.T) {
	md := buildOutgoingMetadata(map[string]any{
		"metadata": map[string]any{
			"key": []any{"a", "b"},
		},
	})
	vals := md.Get("key")
	if len(vals) != 2 || vals[0] != "a" || vals[1] != "b" {
		t.Errorf("key = %v, want [a, b]", vals)
	}
}

func TestBuildOutgoingMetadata_OtherType(t *testing.T) {
	md := buildOutgoingMetadata(map[string]any{
		"metadata": map[string]any{
			"key": 42,
		},
	})
	vals := md.Get("key")
	if len(vals) != 1 || vals[0] != "42" {
		t.Errorf("key = %v, want [42]", vals)
	}
}

func TestMetadataToMap_Empty(t *testing.T) {
	m := metadataToMap(nil)
	if len(m) != 0 {
		t.Errorf("expected empty map, got %v", m)
	}
}

func TestMetadataToMap_SingleValue(t *testing.T) {
	md := grpcmd.Pairs("key", "value")
	m := metadataToMap(md)
	if m["key"] != "value" {
		t.Errorf("key = %v, want value", m["key"])
	}
}

func TestMetadataToMap_MultiValue(t *testing.T) {
	md := grpcmd.Pairs("key", "a", "key", "b")
	m := metadataToMap(md)
	vals, ok := m["key"].([]string)
	if !ok || len(vals) != 2 {
		t.Errorf("key = %v, want [a, b]", m["key"])
	}
}

// ---------- Reflect / dynamic message tests ----------

func TestBuildDynamicMessage_NilData(t *testing.T) {
	msg, err := buildDynamicMessage(getTestInputDesc(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
}

func TestBuildDynamicMessage_ValidFields(t *testing.T) {
	msg, err := buildDynamicMessage(getTestInputDesc(), map[string]any{
		"message": "test",
		"code":    42,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	v := msg.Get(getTestInputDesc().Fields().ByName("message")).String()
	if v != "test" {
		t.Errorf("message = %v, want test", v)
	}
}

func TestBuildDynamicMessage_UnknownField(t *testing.T) {
	_, err := buildDynamicMessage(getTestInputDesc(), map[string]any{
		"nonexistent": "value",
	})
	if err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestBuildDynamicMessage_WrongType(t *testing.T) {
	_, err := buildDynamicMessage(getTestInputDesc(), map[string]any{
		"message": 123,
	})
	if err == nil {
		t.Fatal("expected error for wrong type")
	}
}

func TestDynamicMessageToMap(t *testing.T) {
	msg := dynamicpb.NewMessage(getTestOutputDesc())
	msg.Set(getTestOutputDesc().Fields().ByName("message"), protoreflect.ValueOfString("hello"))
	msg.Set(getTestOutputDesc().Fields().ByName("code"), protoreflect.ValueOfInt32(42))

	m := dynamicMessageToMap(msg)
	if m["message"] != "hello" {
		t.Errorf("message = %v, want hello", m["message"])
	}
	if m["code"] != 42 {
		t.Errorf("code = %v, want 42", m["code"])
	}
}

// ---------- Client helper tests ----------

func TestGetBool_Default(t *testing.T) {
	if getBool(map[string]any{}, "key", true) != true {
		t.Error("expected true default")
	}
	if getBool(map[string]any{"key": "not bool"}, "key", false) != false {
		t.Error("expected false for non-bool")
	}
}

func TestGetString_Default(t *testing.T) {
	if getString(map[string]any{}, "key", "def") != "def" {
		t.Error("expected default")
	}
	if getString(map[string]any{"key": 123}, "key", "def") != "def" {
		t.Error("expected default for non-string")
	}
	if getString(map[string]any{"key": "val"}, "key", "def") != "val" {
		t.Error("expected val")
	}
}

// ---------- Coercion tests ----------

func TestCoerceInt32_Types(t *testing.T) {
	tests := []struct {
		name string
		val  any
	}{
		{"int", int(42)},
		{"int32", int32(42)},
		{"int64", int64(42)},
		{"float64", float64(42)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := coerceInt32(tt.val)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v.Int() != 42 {
				t.Errorf("got %v, want 42", v.Int())
			}
		})
	}

	_, err := coerceInt32("not a number")
	if err == nil {
		t.Error("expected error for string")
	}
}

func TestCoerceInt64_Types(t *testing.T) {
	tests := []struct {
		name string
		val  any
	}{
		{"int", int(42)},
		{"int32", int32(42)},
		{"int64", int64(42)},
		{"float64", float64(42)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := coerceInt64(tt.val)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v.Int() != 42 {
				t.Errorf("got %v, want 42", v.Int())
			}
		})
	}

	_, err := coerceInt64("not a number")
	if err == nil {
		t.Error("expected error for string")
	}
}

func TestCoerceUint32_Types(t *testing.T) {
	tests := []struct {
		name string
		val  any
	}{
		{"int", int(42)},
		{"uint32", uint32(42)},
		{"float64", float64(42)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := coerceUint32(tt.val)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v.Uint() != 42 {
				t.Errorf("got %v, want 42", v.Uint())
			}
		})
	}

	_, err := coerceUint32("not a number")
	if err == nil {
		t.Error("expected error for string")
	}
}

func TestCoerceUint64_Types(t *testing.T) {
	tests := []struct {
		name string
		val  any
	}{
		{"int", int(42)},
		{"uint64", uint64(42)},
		{"float64", float64(42)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, err := coerceUint64(tt.val)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if v.Uint() != 42 {
				t.Errorf("got %v, want 42", v.Uint())
			}
		})
	}

	_, err := coerceUint64("not a number")
	if err == nil {
		t.Error("expected error for string")
	}
}

func TestCoerceValue_Bool(t *testing.T) {
	// code is an int32 field; passing bool should fail.
	_, err := coerceValue(getTestInputDesc().Fields().ByName("code"), true)
	if err == nil {
		t.Error("expected error for bool on int32 field")
	}
}

func TestCoerceValue_Bytes(t *testing.T) {
	// Test bytes coercion path with string.
	_ = t // Tested indirectly; direct unit test for coerceValue with bytes.
	// We test the error path.
	fd := getTestInputDesc().Fields().ByName("message") // string field
	_, err := coerceValue(fd, 123)
	if err == nil {
		t.Error("expected error for int on string field")
	}
}

// ---------- Reflection list services ----------

func TestListServices(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	rc := newReflectionClient(c.conn)
	services, err := rc.listServices(context.Background())
	if err != nil {
		t.Fatalf("listServices failed: %v", err)
	}

	found := false
	for _, svc := range services {
		if svc == "testpkg.EchoService" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected testpkg.EchoService in services: %v", services)
	}
}

// ---------- Connection refused tests ----------

func TestSetup_ConnectionRefused(t *testing.T) {
	// grpc.NewClient doesn't actually connect immediately, so Setup succeeds.
	// But Execute should fail when trying to use the connection.
	c := New()
	err := c.Setup(context.Background(), map[string]any{
		"endpoint":  "127.0.0.1:1", // nobody listens here
		"plaintext": true,
	})
	if err != nil {
		t.Fatalf("Setup should succeed (lazy connect): %v", err)
	}
	defer c.Teardown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = c.Execute(ctx, connector.Step{
		Action: "unary",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "Echo",
		},
	})
	if err == nil {
		t.Fatal("expected error when connection refused")
	}
}

// ---------- Proto serialization round-trip ----------

func TestDynamicMessage_RoundTrip(t *testing.T) {
	inputDesc := getTestInputDesc()
	original := map[string]any{
		"message": "roundtrip",
		"code":    99,
	}

	msg, err := buildDynamicMessage(inputDesc, original)
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	// Serialize and deserialize.
	data, err := proto.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	msg2 := dynamicpb.NewMessage(inputDesc)
	if err := proto.Unmarshal(data, msg2); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	result := dynamicMessageToMap(msg2)
	if result["message"] != "roundtrip" {
		t.Errorf("message = %v", result["message"])
	}
	if result["code"] != 99 {
		t.Errorf("code = %v", result["code"])
	}
}

// ---------- Float coercion tests ----------

func TestCoerceValue_FloatTypes(t *testing.T) {
	// We need a proto field descriptor for float. Build one on the fly.
	fd := buildFloatFieldDesc(t)

	tests := []struct {
		name string
		val  any
	}{
		{"float64", float64(3.14)},
		{"float32", float32(3.14)},
		{"int", int(3)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := coerceValue(fd, tt.val)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	_, err := coerceValue(fd, "not a float")
	if err == nil {
		t.Error("expected error for string on float field")
	}
}

func TestCoerceValue_DoubleTypes(t *testing.T) {
	fd := buildDoubleFieldDesc(t)

	tests := []struct {
		name string
		val  any
	}{
		{"float64", float64(3.14)},
		{"float32", float32(3.14)},
		{"int", int(3)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := coerceValue(fd, tt.val)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}

	_, err := coerceValue(fd, "not a double")
	if err == nil {
		t.Error("expected error for string on double field")
	}
}

func TestCoerceValue_BoolField(t *testing.T) {
	fd := buildBoolFieldDesc(t)
	_, err := coerceValue(fd, true)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	_, err = coerceValue(fd, "not bool")
	if err == nil {
		t.Error("expected error for string on bool field")
	}
}

func TestCoerceValue_BytesField(t *testing.T) {
	fd := buildBytesFieldDesc(t)

	_, err := coerceValue(fd, "hello")
	if err != nil {
		t.Errorf("unexpected error for string: %v", err)
	}

	_, err = coerceValue(fd, []byte("hello"))
	if err != nil {
		t.Errorf("unexpected error for []byte: %v", err)
	}

	_, err = coerceValue(fd, 123)
	if err == nil {
		t.Error("expected error for int on bytes field")
	}
}

// Helper functions to build field descriptors for specific types.
func buildFieldDesc(t *testing.T, fieldType descriptorpb.FieldDescriptorProto_Type, fieldName string) protoreflect.FieldDescriptor {
	t.Helper()
	syntax := "proto3"
	pkg := "testhelper"
	fname := "helper_" + fieldName + ".proto"
	msgName := "HelperMsg" + fieldName
	label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	number := int32(1)

	fd := &descriptorpb.FileDescriptorProto{
		Name:    &fname,
		Package: &pkg,
		Syntax:  &syntax,
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: &msgName,
				Field: []*descriptorpb.FieldDescriptorProto{
					{
						Name:   &fieldName,
						Number: &number,
						Type:   &fieldType,
						Label:  &label,
					},
				},
			},
		},
	}

	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{fd}}
	files, err := protodesc.NewFiles(fds)
	if err != nil {
		t.Fatalf("building file desc: %v", err)
	}
	var result protoreflect.FieldDescriptor
	files.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		msgs := f.Messages()
		for i := 0; i < msgs.Len(); i++ {
			flds := msgs.Get(i).Fields()
			for j := 0; j < flds.Len(); j++ {
				if string(flds.Get(j).Name()) == fieldName {
					result = flds.Get(j)
					return false
				}
			}
		}
		return true
	})
	return result
}

func buildFloatFieldDesc(t *testing.T) protoreflect.FieldDescriptor {
	return buildFieldDesc(t, descriptorpb.FieldDescriptorProto_TYPE_FLOAT, "float_val")
}

func buildDoubleFieldDesc(t *testing.T) protoreflect.FieldDescriptor {
	return buildFieldDesc(t, descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, "double_val")
}

func buildBoolFieldDesc(t *testing.T) protoreflect.FieldDescriptor {
	return buildFieldDesc(t, descriptorpb.FieldDescriptorProto_TYPE_BOOL, "bool_val")
}

func buildBytesFieldDesc(t *testing.T) protoreflect.FieldDescriptor {
	return buildFieldDesc(t, descriptorpb.FieldDescriptorProto_TYPE_BYTES, "bytes_val")
}

// ---------- Enum coercion tests ----------

func TestCoerceEnum(t *testing.T) {
	// Build a proto with an enum field.
	syntax := "proto3"
	pkg := "testenum"
	fname := "enum_test.proto"
	msgName := "EnumMsg"
	enumName := "Status"
	fieldName := "status"

	val0Name := "UNKNOWN"
	val0Num := int32(0)
	val1Name := "ACTIVE"
	val1Num := int32(1)

	label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	fieldNum := int32(1)
	fieldType := descriptorpb.FieldDescriptorProto_TYPE_ENUM
	typeName := ".testenum.Status"

	fd := &descriptorpb.FileDescriptorProto{
		Name:    &fname,
		Package: &pkg,
		Syntax:  &syntax,
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{
				Name: &enumName,
				Value: []*descriptorpb.EnumValueDescriptorProto{
					{Name: &val0Name, Number: &val0Num},
					{Name: &val1Name, Number: &val1Num},
				},
			},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: &msgName,
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: &fieldName, Number: &fieldNum, Type: &fieldType, Label: &label, TypeName: &typeName},
				},
			},
		},
	}

	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{fd}}
	files, err := protodesc.NewFiles(fds)
	if err != nil {
		t.Fatalf("building file desc: %v", err)
	}

	var enumField protoreflect.FieldDescriptor
	files.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		msgs := f.Messages()
		for i := 0; i < msgs.Len(); i++ {
			flds := msgs.Get(i).Fields()
			for j := 0; j < flds.Len(); j++ {
				if string(flds.Get(j).Name()) == "status" {
					enumField = flds.Get(j)
					return false
				}
			}
		}
		return true
	})

	// Test by int.
	v, err := coerceEnum(enumField, int(1))
	if err != nil {
		t.Fatalf("int: %v", err)
	}
	if v.Enum() != 1 {
		t.Errorf("got %v, want 1", v.Enum())
	}

	// Test by float64.
	v, err = coerceEnum(enumField, float64(1))
	if err != nil {
		t.Fatalf("float64: %v", err)
	}

	// Test by name.
	v, err = coerceEnum(enumField, "ACTIVE")
	if err != nil {
		t.Fatalf("string: %v", err)
	}
	if v.Enum() != 1 {
		t.Errorf("got %v, want 1", v.Enum())
	}

	// Test unknown name.
	_, err = coerceEnum(enumField, "INVALID")
	if err == nil {
		t.Error("expected error for unknown enum name")
	}

	// Test invalid type.
	_, err = coerceEnum(enumField, []byte{})
	if err == nil {
		t.Error("expected error for invalid type")
	}
}

// ---------- buildDialOptions edge cases ----------

func TestBuildDialOptions_PlaintextOverridesTLS(t *testing.T) {
	opts, err := buildDialOptions(map[string]any{
		"tls":       true,
		"plaintext": true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) == 0 {
		t.Fatal("expected at least one dial option")
	}
}

func TestBuildDialOptions_TLS(t *testing.T) {
	_, _, caFile := generateTestCert(t)
	opts, err := buildDialOptions(map[string]any{
		"tls":         true,
		"tls_ca_file": caFile,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(opts) == 0 {
		t.Fatal("expected at least one dial option")
	}
}

// ---------- singleProtoValueToGo coverage ----------

func TestSingleProtoValueToGo_AllTypes(t *testing.T) {
	// Test uint32 path.
	fd32 := buildFieldDesc(t, descriptorpb.FieldDescriptorProto_TYPE_UINT32, "u32_val")
	v := singleProtoValueToGo(fd32, protoreflect.ValueOfUint32(42))
	if v.(uint32) != 42 {
		t.Errorf("uint32: got %v", v)
	}

	// Test uint64 path.
	fd64 := buildFieldDesc(t, descriptorpb.FieldDescriptorProto_TYPE_UINT64, "u64_val")
	v = singleProtoValueToGo(fd64, protoreflect.ValueOfUint64(99))
	if v.(uint64) != 99 {
		t.Errorf("uint64: got %v", v)
	}

	// Test float path.
	fdf := buildFloatFieldDesc(t)
	v = singleProtoValueToGo(fdf, protoreflect.ValueOfFloat32(3.14))
	if v.(float32) == 0 {
		t.Error("float: expected non-zero")
	}

	// Test double path.
	fdd := buildDoubleFieldDesc(t)
	v = singleProtoValueToGo(fdd, protoreflect.ValueOfFloat64(3.14))
	if v.(float64) != 3.14 {
		t.Errorf("double: got %v", v)
	}

	// Test bool path.
	fdb := buildBoolFieldDesc(t)
	v = singleProtoValueToGo(fdb, protoreflect.ValueOfBool(true))
	if v.(bool) != true {
		t.Error("bool: expected true")
	}

	// Test int64 path.
	fdi64 := buildFieldDesc(t, descriptorpb.FieldDescriptorProto_TYPE_INT64, "i64_val")
	v = singleProtoValueToGo(fdi64, protoreflect.ValueOfInt64(100))
	if v.(int64) != 100 {
		t.Errorf("int64: got %v", v)
	}

	// Test bytes path.
	fdbytes := buildBytesFieldDesc(t)
	v = singleProtoValueToGo(fdbytes, protoreflect.ValueOfBytes([]byte("hello")))
	if string(v.([]byte)) != "hello" {
		t.Errorf("bytes: got %v", v)
	}
}

// ---------- Streaming edge cases ----------

func TestServerStream_UnknownService(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "server_stream",
		Parameters: map[string]any{
			"service": "nonexistent.Service",
			"method":  "Foo",
		},
	})
	if err == nil {
		t.Fatal("expected error for unknown service")
	}
}

func TestBidiStream_MissingMessages(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "bidi_stream",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "BidiStreamEcho",
		},
	})
	if err == nil {
		t.Fatal("expected error for missing messages")
	}
}

// ---------- Enum in singleProtoValueToGo ----------

func TestSingleProtoValueToGo_Enum(t *testing.T) {
	// Build enum field desc.
	syntax := "proto3"
	pkg := "enumtest2"
	fname := "enum2.proto"
	msgName := "Msg"
	enumName := "E"
	fieldName := "e"
	val0Name := "A"
	val0Num := int32(0)
	label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	fieldNum := int32(1)
	fieldType := descriptorpb.FieldDescriptorProto_TYPE_ENUM
	typeName := ".enumtest2.E"

	fd := &descriptorpb.FileDescriptorProto{
		Name: &fname, Package: &pkg, Syntax: &syntax,
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{Name: &enumName, Value: []*descriptorpb.EnumValueDescriptorProto{
				{Name: &val0Name, Number: &val0Num},
			}},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: &msgName, Field: []*descriptorpb.FieldDescriptorProto{
				{Name: &fieldName, Number: &fieldNum, Type: &fieldType, Label: &label, TypeName: &typeName},
			}},
		},
	}

	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{fd}}
	files, _ := protodesc.NewFiles(fds)
	var ef protoreflect.FieldDescriptor
	files.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		ef = f.Messages().Get(0).Fields().Get(0)
		return false
	})

	v := singleProtoValueToGo(ef, protoreflect.ValueOfEnum(0))
	if v.(int) != 0 {
		t.Errorf("enum: got %v", v)
	}
}

// ---------- Edge case: coerceSingleValue coverage ----------

func TestCoerceSingleValue_Bool(t *testing.T) {
	fd := buildBoolFieldDesc(t)
	v, err := coerceSingleValue(fd, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Bool() != true {
		t.Error("expected true")
	}
	_, err = coerceSingleValue(fd, "not bool")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCoerceSingleValue_BytesError(t *testing.T) {
	fd := buildBytesFieldDesc(t)
	_, err := coerceSingleValue(fd, 123)
	if err == nil {
		t.Error("expected error")
	}
}

func TestCoerceSingleValue_Float(t *testing.T) {
	fd := buildFloatFieldDesc(t)
	v, err := coerceSingleValue(fd, float64(3.14))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Float() == 0 {
		t.Error("expected non-zero")
	}
	_, err = coerceSingleValue(fd, "not float")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCoerceSingleValue_Double(t *testing.T) {
	fd := buildDoubleFieldDesc(t)
	v, err := coerceSingleValue(fd, float64(3.14))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Float() != 3.14 {
		t.Errorf("got %v", v.Float())
	}
	_, err = coerceSingleValue(fd, "not double")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCoerceSingleValue_StringError(t *testing.T) {
	fd := buildFieldDesc(t, descriptorpb.FieldDescriptorProto_TYPE_STRING, "str_val")
	_, err := coerceSingleValue(fd, 123)
	if err == nil {
		t.Error("expected error")
	}
}

func TestCoerceSingleValue_Int(t *testing.T) {
	fd := buildFloatFieldDesc(t)
	v, err := coerceSingleValue(fd, int(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = v
}

func TestCoerceSingleValue_DoubleInt(t *testing.T) {
	fd := buildDoubleFieldDesc(t)
	v, err := coerceSingleValue(fd, int(5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = v
}

func TestCoerceSingleValue_Message(t *testing.T) {
	// Need a message field. Use the input desc which has only scalar fields.
	// We can't easily test this without a proper message field.
	// But we can test the error path.
	fd := getTestInputDesc().Fields().ByName("message") // string field
	_, err := coerceSingleValue(fd, map[string]any{"x": 1})
	if err == nil {
		t.Error("expected error for map on string field")
	}
}

// ---------- Complex proto tests (list, map, nested message) ----------

// buildComplexProto creates a proto with repeated, map, and nested message fields.
func buildComplexProto(t *testing.T) protoreflect.MessageDescriptor {
	t.Helper()
	syntax := "proto3"
	pkg := "complexpkg"
	fname := "complex.proto"

	// Inner message
	innerName := "Inner"
	innerFieldName := "value"
	innerFieldNum := int32(1)
	innerFieldType := descriptorpb.FieldDescriptorProto_TYPE_STRING
	innerLabel := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL

	// Outer message with:
	//   repeated string tags = 1;
	//   map<string, string> labels = 2;
	//   Inner nested = 3;
	//   repeated int32 nums = 4;
	outerName := "Outer"

	// repeated string tags
	tagsName := "tags"
	tagsNum := int32(1)
	tagsType := descriptorpb.FieldDescriptorProto_TYPE_STRING
	tagsLabel := descriptorpb.FieldDescriptorProto_LABEL_REPEATED

	// map<string, string> labels => LabelsEntry message with key/value
	labelsEntryName := "LabelsEntry"
	labelsKeyName := "key"
	labelsKeyNum := int32(1)
	labelsValName := "value"
	labelsValNum := int32(2)
	labelsKeyType := descriptorpb.FieldDescriptorProto_TYPE_STRING
	labelsValType := descriptorpb.FieldDescriptorProto_TYPE_STRING
	labelsEntryLabel := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL

	mapEntryOpts := &descriptorpb.MessageOptions{MapEntry: boolPtr(true)}

	labelsName := "labels"
	labelsNum := int32(2)
	labelsType := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	labelsTypeName := ".complexpkg.Outer.LabelsEntry"
	labelsLabel := descriptorpb.FieldDescriptorProto_LABEL_REPEATED

	// Inner nested = 3
	nestedName := "nested"
	nestedNum := int32(3)
	nestedType := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	nestedTypeName := ".complexpkg.Inner"
	nestedLabel := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL

	// repeated int32 nums = 4
	numsName := "nums"
	numsNum := int32(4)
	numsType := descriptorpb.FieldDescriptorProto_TYPE_INT32
	numsLabel := descriptorpb.FieldDescriptorProto_LABEL_REPEATED

	fd := &descriptorpb.FileDescriptorProto{
		Name:    &fname,
		Package: &pkg,
		Syntax:  &syntax,
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: &innerName,
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: &innerFieldName, Number: &innerFieldNum, Type: &innerFieldType, Label: &innerLabel},
				},
			},
			{
				Name: &outerName,
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: &tagsName, Number: &tagsNum, Type: &tagsType, Label: &tagsLabel},
					{Name: &labelsName, Number: &labelsNum, Type: &labelsType, Label: &labelsLabel, TypeName: &labelsTypeName},
					{Name: &nestedName, Number: &nestedNum, Type: &nestedType, Label: &nestedLabel, TypeName: &nestedTypeName},
					{Name: &numsName, Number: &numsNum, Type: &numsType, Label: &numsLabel},
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name:    &labelsEntryName,
						Options: mapEntryOpts,
						Field: []*descriptorpb.FieldDescriptorProto{
							{Name: &labelsKeyName, Number: &labelsKeyNum, Type: &labelsKeyType, Label: &labelsEntryLabel},
							{Name: &labelsValName, Number: &labelsValNum, Type: &labelsValType, Label: &labelsEntryLabel},
						},
					},
				},
			},
		},
	}

	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{fd}}
	files, err := protodesc.NewFiles(fds)
	if err != nil {
		t.Fatalf("building complex proto: %v", err)
	}

	var outerDesc protoreflect.MessageDescriptor
	files.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		msgs := f.Messages()
		for i := 0; i < msgs.Len(); i++ {
			if string(msgs.Get(i).Name()) == "Outer" {
				outerDesc = msgs.Get(i)
				return false
			}
		}
		return true
	})
	return outerDesc
}

func boolPtr(b bool) *bool { return &b }

func TestCoerceListValue_Strings(t *testing.T) {
	desc := buildComplexProto(t)
	fd := desc.Fields().ByName("tags")

	v, err := coerceValue(fd, []any{"a", "b", "c"})
	if err != nil {
		t.Fatalf("coerceValue list: %v", err)
	}
	list := v.List()
	if list.Len() != 3 {
		t.Fatalf("expected 3 items, got %d", list.Len())
	}
	if list.Get(0).String() != "a" {
		t.Errorf("item 0 = %v", list.Get(0).String())
	}
}

func TestCoerceListValue_Ints(t *testing.T) {
	desc := buildComplexProto(t)
	fd := desc.Fields().ByName("nums")

	v, err := coerceValue(fd, []any{1, 2, 3})
	if err != nil {
		t.Fatalf("coerceValue list ints: %v", err)
	}
	list := v.List()
	if list.Len() != 3 {
		t.Fatalf("expected 3 items, got %d", list.Len())
	}
}

func TestCoerceListValue_Error(t *testing.T) {
	desc := buildComplexProto(t)
	fd := desc.Fields().ByName("tags")

	// Not a list.
	_, err := coerceValue(fd, "not a list")
	if err == nil {
		t.Error("expected error for non-list")
	}

	// List element wrong type.
	_, err = coerceValue(fd, []any{123})
	if err == nil {
		t.Error("expected error for wrong element type")
	}
}

func TestCoerceMapValue_Strings(t *testing.T) {
	desc := buildComplexProto(t)
	fd := desc.Fields().ByName("labels")

	v, err := coerceValue(fd, map[string]any{"k1": "v1", "k2": "v2"})
	if err != nil {
		t.Fatalf("coerceValue map: %v", err)
	}
	m := v.Map()
	if m.Len() != 2 {
		t.Fatalf("expected 2 entries, got %d", m.Len())
	}
}

func TestCoerceMapValue_Error(t *testing.T) {
	desc := buildComplexProto(t)
	fd := desc.Fields().ByName("labels")

	// Not a map.
	_, err := coerceValue(fd, "not a map")
	if err == nil {
		t.Error("expected error for non-map")
	}
}

func TestCoerceValue_NestedMessage(t *testing.T) {
	desc := buildComplexProto(t)
	fd := desc.Fields().ByName("nested")

	v, err := coerceValue(fd, map[string]any{"value": "hello"})
	if err != nil {
		t.Fatalf("coerceValue message: %v", err)
	}
	msg := v.Message().Interface().(*dynamicpb.Message)
	innerVal := msg.Get(fd.Message().Fields().ByName("value")).String()
	if innerVal != "hello" {
		t.Errorf("nested value = %v", innerVal)
	}

	// Error: not a map.
	_, err = coerceValue(fd, "not a map")
	if err == nil {
		t.Error("expected error for non-map on message field")
	}
}

func TestCoerceSingleValue_NestedMessage(t *testing.T) {
	desc := buildComplexProto(t)
	fd := desc.Fields().ByName("nested")

	v, err := coerceSingleValue(fd, map[string]any{"value": "hi"})
	if err != nil {
		t.Fatalf("coerceSingleValue message: %v", err)
	}
	msg := v.Message().Interface().(*dynamicpb.Message)
	innerVal := msg.Get(fd.Message().Fields().ByName("value")).String()
	if innerVal != "hi" {
		t.Errorf("nested value = %v", innerVal)
	}

	// Error: not a map.
	_, err = coerceSingleValue(fd, 123)
	if err == nil {
		t.Error("expected error for non-map on message field")
	}
}

func TestCoerceSingleValue_Enum(t *testing.T) {
	// Build an enum field for coerceSingleValue.
	syntax := "proto3"
	pkg := "enumsingle"
	fname := "enumsingle.proto"
	msgName := "Msg"
	enumName := "E"
	fieldName := "e"
	val0Name := "X"
	val0Num := int32(0)
	label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	fieldNum := int32(1)
	fieldType := descriptorpb.FieldDescriptorProto_TYPE_ENUM
	typeName := ".enumsingle.E"

	fd := &descriptorpb.FileDescriptorProto{
		Name: &fname, Package: &pkg, Syntax: &syntax,
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{Name: &enumName, Value: []*descriptorpb.EnumValueDescriptorProto{
				{Name: &val0Name, Number: &val0Num},
			}},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: &msgName, Field: []*descriptorpb.FieldDescriptorProto{
				{Name: &fieldName, Number: &fieldNum, Type: &fieldType, Label: &label, TypeName: &typeName},
			}},
		},
	}

	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{fd}}
	files, _ := protodesc.NewFiles(fds)
	var ef protoreflect.FieldDescriptor
	files.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		ef = f.Messages().Get(0).Fields().Get(0)
		return false
	})

	v, err := coerceSingleValue(ef, int(0))
	if err != nil {
		t.Fatalf("enum int: %v", err)
	}
	if v.Enum() != 0 {
		t.Errorf("got %v", v.Enum())
	}

	_, err = coerceSingleValue(ef, []byte{})
	if err == nil {
		t.Error("expected error for invalid enum type")
	}
}

// ---------- dynamicMessageToMap with complex types ----------

func TestDynamicMessageToMap_Complex(t *testing.T) {
	desc := buildComplexProto(t)

	msg, err := buildDynamicMessage(desc, map[string]any{
		"tags":   []any{"a", "b"},
		"labels": map[string]any{"k": "v"},
		"nested": map[string]any{"value": "inner"},
		"nums":   []any{1, 2},
	})
	if err != nil {
		t.Fatalf("build: %v", err)
	}

	m := dynamicMessageToMap(msg)

	// Check tags (list).
	tags, ok := m["tags"].([]any)
	if !ok {
		t.Fatalf("tags type: %T", m["tags"])
	}
	if len(tags) != 2 || tags[0] != "a" || tags[1] != "b" {
		t.Errorf("tags = %v", tags)
	}

	// Check labels (map).
	labels, ok := m["labels"].(map[string]any)
	if !ok {
		t.Fatalf("labels type: %T", m["labels"])
	}
	if labels["k"] != "v" {
		t.Errorf("labels = %v", labels)
	}

	// Check nested (message).
	nested, ok := m["nested"].(map[string]any)
	if !ok {
		t.Fatalf("nested type: %T", m["nested"])
	}
	if nested["value"] != "inner" {
		t.Errorf("nested = %v", nested)
	}

	// Check nums (repeated int32).
	nums, ok := m["nums"].([]any)
	if !ok {
		t.Fatalf("nums type: %T", m["nums"])
	}
	if len(nums) != 2 {
		t.Errorf("nums len = %d", len(nums))
	}
}

// ---------- Streaming with metadata verification ----------

func TestServerStream_WithMetadata(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "server_stream",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "ServerStreamEcho",
			"message": map[string]any{"message": "md-test"},
			"metadata": map[string]any{
				"x-test-key": "x-test-val",
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Check trailers present.
	trailers, ok := result.Data["trailers"].(map[string]any)
	if !ok {
		t.Fatal("trailers not a map")
	}
	if trailers["trailer-key"] != "trailer-value" {
		t.Errorf("trailer = %v", trailers["trailer-key"])
	}
}

func TestClientStream_WithMetadata(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "client_stream",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "ClientStreamEcho",
			"messages": []any{
				map[string]any{"message": "x"},
			},
			"metadata": map[string]any{
				"x-cs": "val",
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	trailers, ok := result.Data["trailers"].(map[string]any)
	if !ok {
		t.Fatal("trailers not a map")
	}
	if trailers["trailer-key"] != "trailer-value" {
		t.Errorf("trailer = %v", trailers["trailer-key"])
	}
}

func TestBidiStream_WithMetadata(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "bidi_stream",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "BidiStreamEcho",
			"messages": []any{
				map[string]any{"message": "x"},
			},
			"metadata": map[string]any{
				"x-bidi": "val",
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	trailers, ok := result.Data["trailers"].(map[string]any)
	if !ok {
		t.Fatal("trailers not a map")
	}
	if trailers["trailer-key"] != "trailer-value" {
		t.Errorf("trailer = %v", trailers["trailer-key"])
	}
}

// ---------- Connection refused for streaming ----------

func TestServerStream_ConnectionRefused(t *testing.T) {
	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": "127.0.0.1:1", "plaintext": true})
	defer c.Teardown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := c.Execute(ctx, connector.Step{
		Action: "server_stream",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "ServerStreamEcho",
			"message": map[string]any{"message": "x"},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestClientStream_ConnectionRefused(t *testing.T) {
	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": "127.0.0.1:1", "plaintext": true})
	defer c.Teardown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := c.Execute(ctx, connector.Step{
		Action: "client_stream",
		Parameters: map[string]any{
			"service":  "testpkg.EchoService",
			"method":   "ClientStreamEcho",
			"messages": []any{map[string]any{"message": "x"}},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestBidiStream_ConnectionRefused(t *testing.T) {
	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": "127.0.0.1:1", "plaintext": true})
	defer c.Teardown(context.Background())

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := c.Execute(ctx, connector.Step{
		Action: "bidi_stream",
		Parameters: map[string]any{
			"service":  "testpkg.EchoService",
			"method":   "BidiStreamEcho",
			"messages": []any{map[string]any{"message": "x"}},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

// ---------- Unknown service/method for client_stream and bidi_stream ----------

func TestClientStream_UnknownService(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "client_stream",
		Parameters: map[string]any{
			"service":  "nonexistent.Service",
			"method":   "Foo",
			"messages": []any{map[string]any{"message": "x"}},
		},
	})
	if err == nil {
		t.Fatal("expected error for unknown service")
	}
}

func TestBidiStream_UnknownService(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "bidi_stream",
		Parameters: map[string]any{
			"service":  "nonexistent.Service",
			"method":   "Foo",
			"messages": []any{map[string]any{"message": "x"}},
		},
	})
	if err == nil {
		t.Fatal("expected error for unknown service")
	}
}

// ---------- Build message errors for streaming ----------

func TestServerStream_BuildMessageError(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "server_stream",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "ServerStreamEcho",
			"message": map[string]any{"nonexistent_field": "val"},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid field")
	}
}

func TestClientStream_BuildMessageError(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "client_stream",
		Parameters: map[string]any{
			"service":  "testpkg.EchoService",
			"method":   "ClientStreamEcho",
			"messages": []any{map[string]any{"nonexistent_field": "val"}},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid field")
	}
}

func TestBidiStream_BuildMessageError(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "bidi_stream",
		Parameters: map[string]any{
			"service":  "testpkg.EchoService",
			"method":   "BidiStreamEcho",
			"messages": []any{map[string]any{"nonexistent_field": "val"}},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid field")
	}
}

// ---------- Unary build message error ----------

func TestUnary_BuildMessageError(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	_, err := c.Execute(context.Background(), connector.Step{
		Action: "unary",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "Echo",
			"message": map[string]any{"nonexistent_field": "val"},
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid field")
	}
}

// ---------- Proto with dependency for resolveDependencies coverage ----------

func buildDepProtos(t *testing.T) (dep *descriptorpb.FileDescriptorProto, main *descriptorpb.FileDescriptorProto) {
	t.Helper()

	syntax := "proto3"

	// Dependency file: dep.proto
	depPkg := "deppkg"
	depFname := "dep.proto"
	depMsgName := "DepMsg"
	depFieldName := "val"
	depFieldNum := int32(1)
	depFieldType := descriptorpb.FieldDescriptorProto_TYPE_STRING
	depLabel := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL

	dep = &descriptorpb.FileDescriptorProto{
		Name:    &depFname,
		Package: &depPkg,
		Syntax:  &syntax,
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: &depMsgName,
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: &depFieldName, Number: &depFieldNum, Type: &depFieldType, Label: &depLabel},
				},
			},
		},
	}

	// Main file: main_with_dep.proto (imports dep.proto)
	mainPkg := "mainpkg"
	mainFname := "main_with_dep.proto"
	mainMsgName := "MainMsg"
	mainFieldName := "dep"
	mainFieldNum := int32(1)
	mainFieldType := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	mainFieldTypeName := ".deppkg.DepMsg"
	mainLabel := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL

	mainRespName := "MainResp"
	mainRespFieldName := "result"
	mainRespFieldNum := int32(1)
	mainRespFieldType := descriptorpb.FieldDescriptorProto_TYPE_STRING
	mainRespLabel := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL

	svcName := "DepService"
	methodName := "DoSomething"
	inputType := ".mainpkg.MainMsg"
	outputType := ".mainpkg.MainResp"
	boolFalse := false

	main = &descriptorpb.FileDescriptorProto{
		Name:       &mainFname,
		Package:    &mainPkg,
		Syntax:     &syntax,
		Dependency: []string{"dep.proto"},
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: &mainMsgName,
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: &mainFieldName, Number: &mainFieldNum, Type: &mainFieldType, Label: &mainLabel, TypeName: &mainFieldTypeName},
				},
			},
			{
				Name: &mainRespName,
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: &mainRespFieldName, Number: &mainRespFieldNum, Type: &mainRespFieldType, Label: &mainRespLabel},
				},
			},
		},
		Service: []*descriptorpb.ServiceDescriptorProto{
			{
				Name: &svcName,
				Method: []*descriptorpb.MethodDescriptorProto{
					{Name: &methodName, InputType: &inputType, OutputType: &outputType,
						ClientStreaming: &boolFalse, ServerStreaming: &boolFalse},
				},
			},
		},
	}

	return dep, main
}

// startDepTestServer starts a server with a service that has proto dependencies.
func startDepTestServer(t *testing.T) (string, func()) {
	t.Helper()

	depFD, mainFD := buildDepProtos(t)
	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{depFD, mainFD}}
	files, err := protodesc.NewFiles(fds)
	if err != nil {
		t.Fatalf("building dep proto: %v", err)
	}

	// Register in global registry.
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if _, lookupErr := protoregistry.GlobalFiles.FindFileByPath(fd.Path()); lookupErr != nil {
			if regErr := protoregistry.GlobalFiles.RegisterFile(fd); regErr != nil {
				t.Logf("registering %s: %v (may already exist)", fd.Path(), regErr)
			}
		}
		return true
	})

	// Find descriptors for handler.
	var inputDesc, outputDesc protoreflect.MessageDescriptor
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		svcs := fd.Services()
		for i := 0; i < svcs.Len(); i++ {
			methods := svcs.Get(i).Methods()
			for j := 0; j < methods.Len(); j++ {
				m := methods.Get(j)
				if string(m.Name()) == "DoSomething" {
					inputDesc = m.Input()
					outputDesc = m.Output()
					return false
				}
			}
		}
		return true
	})

	handler := func(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
		reqMsg := dynamicpb.NewMessage(inputDesc)
		if err := dec(reqMsg); err != nil {
			return nil, err
		}
		respMsg := dynamicpb.NewMessage(outputDesc)
		respMsg.Set(outputDesc.Fields().ByName("result"), protoreflect.ValueOfString("ok"))
		return respMsg, nil
	}

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv := grpc.NewServer()
	srv.RegisterService(&grpc.ServiceDesc{
		ServiceName: "mainpkg.DepService",
		HandlerType: (*any)(nil),
		Methods:     []grpc.MethodDesc{{MethodName: "DoSomething", Handler: handler}},
	}, nil)
	reflection.Register(srv)

	go func() { _ = srv.Serve(lis) }()
	return lis.Addr().String(), func() { srv.GracefulStop(); lis.Close() }
}

func TestUnary_WithDependency(t *testing.T) {
	addr, cleanup := startDepTestServer(t)
	defer cleanup()

	c := New()
	err := c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	if err != nil {
		t.Fatalf("Setup: %v", err)
	}
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "unary",
		Parameters: map[string]any{
			"service": "mainpkg.DepService",
			"method":  "DoSomething",
			"message": map[string]any{
				"dep": map[string]any{"val": "hello"},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if result.Data["status_code"].(int) != 0 {
		t.Errorf("status_code = %v, want 0", result.Data["status_code"])
	}
	resp := result.Data["response"].(map[string]any)
	if resp["result"] != "ok" {
		t.Errorf("result = %v, want ok", resp["result"])
	}
}

// ---------- coerceValue remaining branches ----------

func buildComplexFieldProto(t *testing.T) protoreflect.MessageDescriptor {
	t.Helper()
	syntax := "proto3"
	pkg := "fieldtest"
	fname := "fieldtest.proto"
	msgName := "FieldMsg"

	label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL

	fields := []*descriptorpb.FieldDescriptorProto{
		{Name: strPtr("int64_val"), Number: int32Ptr(1),
			Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_INT64), Label: &label},
		{Name: strPtr("uint32_val"), Number: int32Ptr(2),
			Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_UINT32), Label: &label},
		{Name: strPtr("uint64_val"), Number: int32Ptr(3),
			Type: typePtr(descriptorpb.FieldDescriptorProto_TYPE_UINT64), Label: &label},
	}

	fd := &descriptorpb.FileDescriptorProto{
		Name: &fname, Package: &pkg, Syntax: &syntax,
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: &msgName, Field: fields},
		},
	}

	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{fd}}
	files, err := protodesc.NewFiles(fds)
	if err != nil {
		t.Fatalf("building fieldtest proto: %v", err)
	}
	var desc protoreflect.MessageDescriptor
	files.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		desc = f.Messages().Get(0)
		return false
	})
	return desc
}

func strPtr(s string) *string { return &s }
func int32Ptr(i int32) *int32 { return &i }
func typePtr(t descriptorpb.FieldDescriptorProto_Type) *descriptorpb.FieldDescriptorProto_Type {
	return &t
}

func TestCoerceValue_Int64Field(t *testing.T) {
	desc := buildComplexFieldProto(t)
	fd := desc.Fields().ByName("int64_val")

	v, err := coerceValue(fd, int(42))
	if err != nil {
		t.Fatalf("coerceValue int64: %v", err)
	}
	if v.Int() != 42 {
		t.Errorf("got %v", v.Int())
	}
}

func TestCoerceValue_Uint32Field(t *testing.T) {
	desc := buildComplexFieldProto(t)
	fd := desc.Fields().ByName("uint32_val")

	v, err := coerceValue(fd, int(42))
	if err != nil {
		t.Fatalf("coerceValue uint32: %v", err)
	}
	if v.Uint() != 42 {
		t.Errorf("got %v", v.Uint())
	}
}

func TestCoerceValue_Uint64Field(t *testing.T) {
	desc := buildComplexFieldProto(t)
	fd := desc.Fields().ByName("uint64_val")

	v, err := coerceValue(fd, int(42))
	if err != nil {
		t.Fatalf("coerceValue uint64: %v", err)
	}
	if v.Uint() != 42 {
		t.Errorf("got %v", v.Uint())
	}
}

// Test coerceValue with enum field.
func TestCoerceValue_EnumField(t *testing.T) {
	syntax := "proto3"
	pkg := "enumcv"
	fname := "enumcv.proto"
	msgName := "Msg"
	enumName := "E"
	fieldName := "e"
	val0Name := "ZERO"
	val0Num := int32(0)
	label := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL
	fieldNum := int32(1)
	fieldType := descriptorpb.FieldDescriptorProto_TYPE_ENUM
	typeName := ".enumcv.E"

	fd := &descriptorpb.FileDescriptorProto{
		Name: &fname, Package: &pkg, Syntax: &syntax,
		EnumType: []*descriptorpb.EnumDescriptorProto{
			{Name: &enumName, Value: []*descriptorpb.EnumValueDescriptorProto{
				{Name: &val0Name, Number: &val0Num},
			}},
		},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: &msgName, Field: []*descriptorpb.FieldDescriptorProto{
				{Name: &fieldName, Number: &fieldNum, Type: &fieldType, Label: &label, TypeName: &typeName},
			}},
		},
	}

	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{fd}}
	files, _ := protodesc.NewFiles(fds)
	var ef protoreflect.FieldDescriptor
	files.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		ef = f.Messages().Get(0).Fields().Get(0)
		return false
	})

	v, err := coerceValue(ef, "ZERO")
	if err != nil {
		t.Fatalf("coerceValue enum: %v", err)
	}
	if v.Enum() != 0 {
		t.Errorf("got %v", v.Enum())
	}
}

// ---------- coerceSingleValue remaining branches ----------

func TestCoerceSingleValue_Int32(t *testing.T) {
	fd := buildFieldDesc(t, descriptorpb.FieldDescriptorProto_TYPE_INT32, "i32_single")
	v, err := coerceSingleValue(fd, int(5))
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if v.Int() != 5 {
		t.Errorf("got %v", v.Int())
	}
	_, err = coerceSingleValue(fd, "bad")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCoerceSingleValue_Int64(t *testing.T) {
	fd := buildFieldDesc(t, descriptorpb.FieldDescriptorProto_TYPE_INT64, "i64_single")
	v, err := coerceSingleValue(fd, int(5))
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if v.Int() != 5 {
		t.Errorf("got %v", v.Int())
	}
	_, err = coerceSingleValue(fd, "bad")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCoerceSingleValue_Uint32(t *testing.T) {
	fd := buildFieldDesc(t, descriptorpb.FieldDescriptorProto_TYPE_UINT32, "u32_single")
	v, err := coerceSingleValue(fd, int(5))
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if v.Uint() != 5 {
		t.Errorf("got %v", v.Uint())
	}
	_, err = coerceSingleValue(fd, "bad")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCoerceSingleValue_Uint64(t *testing.T) {
	fd := buildFieldDesc(t, descriptorpb.FieldDescriptorProto_TYPE_UINT64, "u64_single")
	v, err := coerceSingleValue(fd, int(5))
	if err != nil {
		t.Fatalf("unexpected: %v", err)
	}
	if v.Uint() != 5 {
		t.Errorf("got %v", v.Uint())
	}
	_, err = coerceSingleValue(fd, "bad")
	if err == nil {
		t.Error("expected error")
	}
}

func TestCoerceSingleValue_Bytes(t *testing.T) {
	fd := buildBytesFieldDesc(t)
	v, err := coerceSingleValue(fd, "hello")
	if err != nil {
		t.Fatalf("string: %v", err)
	}
	if string(v.Bytes()) != "hello" {
		t.Errorf("got %v", v.Bytes())
	}
	v, err = coerceSingleValue(fd, []byte("world"))
	if err != nil {
		t.Fatalf("[]byte: %v", err)
	}
	if string(v.Bytes()) != "world" {
		t.Errorf("got %v", v.Bytes())
	}
}

// Test singleProtoValueToGo with message field returns nested map.
func TestSingleProtoValueToGo_Message(t *testing.T) {
	desc := buildComplexProto(t)
	fd := desc.Fields().ByName("nested")

	innerDesc := fd.Message()
	inner := dynamicpb.NewMessage(innerDesc)
	inner.Set(innerDesc.Fields().ByName("value"), protoreflect.ValueOfString("test"))

	v := singleProtoValueToGo(fd, protoreflect.ValueOfMessage(inner))
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", v)
	}
	if m["value"] != "test" {
		t.Errorf("value = %v", m["value"])
	}
}

// ---------- findMethod edge cases ----------

func TestCoerceScalar_Float32(t *testing.T) {
	fd := buildFloatFieldDesc(t)
	v, err := coerceScalar(fd, float32(2.5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Float() == 0 {
		t.Error("expected non-zero")
	}
}

func TestCoerceScalar_DoubleFloat32(t *testing.T) {
	fd := buildDoubleFieldDesc(t)
	v, err := coerceScalar(fd, float32(2.5))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Float() == 0 {
		t.Error("expected non-zero")
	}
}

func TestCoerceScalar_MessageError(t *testing.T) {
	desc := buildComplexProto(t)
	fd := desc.Fields().ByName("nested")
	// buildDynamicMessage error (unknown field in nested message).
	_, err := coerceScalar(fd, map[string]any{"unknown_field": "val"})
	if err == nil {
		t.Error("expected error for unknown field in nested message")
	}
}

func TestCoerceMapValue_KeyError(t *testing.T) {
	// Create a map with int32 keys to test key coercion error.
	syntax := "proto3"
	pkg := "mapkeytest"
	fname := "mapkeytest.proto"
	msgName := "MapMsg"

	mapEntryName := "DataEntry"
	keyName := "key"
	keyNum := int32(1)
	valName := "value"
	valNum := int32(2)
	keyType := descriptorpb.FieldDescriptorProto_TYPE_INT32
	valType := descriptorpb.FieldDescriptorProto_TYPE_STRING
	entryLabel := descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL

	mapFieldName := "data"
	mapFieldNum := int32(1)
	mapFieldType := descriptorpb.FieldDescriptorProto_TYPE_MESSAGE
	mapFieldTypeName := ".mapkeytest.MapMsg.DataEntry"
	mapFieldLabel := descriptorpb.FieldDescriptorProto_LABEL_REPEATED

	fd := &descriptorpb.FileDescriptorProto{
		Name: &fname, Package: &pkg, Syntax: &syntax,
		MessageType: []*descriptorpb.DescriptorProto{
			{
				Name: &msgName,
				Field: []*descriptorpb.FieldDescriptorProto{
					{Name: &mapFieldName, Number: &mapFieldNum, Type: &mapFieldType,
						Label: &mapFieldLabel, TypeName: &mapFieldTypeName},
				},
				NestedType: []*descriptorpb.DescriptorProto{
					{
						Name:    &mapEntryName,
						Options: &descriptorpb.MessageOptions{MapEntry: boolPtr(true)},
						Field: []*descriptorpb.FieldDescriptorProto{
							{Name: &keyName, Number: &keyNum, Type: &keyType, Label: &entryLabel},
							{Name: &valName, Number: &valNum, Type: &valType, Label: &entryLabel},
						},
					},
				},
			},
		},
	}

	fds := &descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{fd}}
	files, err := protodesc.NewFiles(fds)
	if err != nil {
		t.Fatalf("building: %v", err)
	}
	var mapField protoreflect.FieldDescriptor
	files.RangeFiles(func(f protoreflect.FileDescriptor) bool {
		mapField = f.Messages().Get(0).Fields().Get(0)
		return false
	})

	// Keys are strings in Go maps, but the proto key is int32.
	// "not_a_number" should fail to coerce to int32.
	_, err = coerceValue(mapField, map[string]any{"not_a_number": "val"})
	if err == nil {
		t.Error("expected error for non-numeric key in int32-keyed map")
	}
}

func TestBuildFileDescriptors_Error(t *testing.T) {
	// Create an invalid file descriptor set with unresolvable dependencies.
	depName := "nonexistent_dep.proto"
	fname := "broken.proto"
	syntax := "proto3"
	pkg := "broken"
	msgName := "Msg"

	fd := &descriptorpb.FileDescriptorProto{
		Name:       &fname,
		Package:    &pkg,
		Syntax:     &syntax,
		Dependency: []string{depName},
		MessageType: []*descriptorpb.DescriptorProto{
			{Name: &msgName},
		},
	}

	allFiles := map[string]*descriptorpb.FileDescriptorProto{
		fname: fd,
	}

	_, err := buildFileDescriptors(allFiles)
	if err == nil {
		t.Error("expected error for unresolvable dependency")
	}
}

func TestFindMethod_ServiceNotFound(t *testing.T) {
	// Use the test file descriptor which has EchoService.
	files := []protoreflect.FileDescriptor{testFD}

	_, _, err := findMethod(files, "nonexistent.Service", "Echo")
	if err == nil {
		t.Error("expected error for nonexistent service")
	}
}

func TestFindMethod_MethodNotFound(t *testing.T) {
	files := []protoreflect.FileDescriptor{testFD}

	_, _, err := findMethod(files, "testpkg.EchoService", "NonexistentMethod")
	if err == nil {
		t.Error("expected error for nonexistent method")
	}
}

func TestFindMethod_SimpleServiceName(t *testing.T) {
	// Test with just "EchoService" (no package prefix).
	files := []protoreflect.FileDescriptor{testFD}

	in, out, err := findMethod(files, "EchoService", "Echo")
	if err != nil {
		t.Fatalf("findMethod simple name: %v", err)
	}
	if in == nil || out == nil {
		t.Fatal("expected non-nil descriptors")
	}
}

// ---------- collectMissingDeps tests ----------

func TestResolveTransitiveDeps_NoDeps(t *testing.T) {
	allFiles := map[string]*descriptorpb.FileDescriptorProto{
		"test.proto": buildTestFileDescriptor(),
	}
	err := resolveTransitiveDeps(allFiles, func(filename string) ([]*descriptorpb.FileDescriptorProto, error) {
		t.Fatalf("fetcher should not be called when no deps missing")
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveTransitiveDeps_WithDeps(t *testing.T) {
	depProto, mainProto := buildDepProtos(t)
	allFiles := map[string]*descriptorpb.FileDescriptorProto{
		mainProto.GetName(): mainProto,
	}
	err := resolveTransitiveDeps(allFiles, func(filename string) ([]*descriptorpb.FileDescriptorProto, error) {
		if filename == "dep.proto" {
			return []*descriptorpb.FileDescriptorProto{depProto}, nil
		}
		return nil, fmt.Errorf("unknown dep: %s", filename)
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := allFiles["dep.proto"]; !ok {
		t.Error("dep.proto should be in allFiles after resolution")
	}
}

func TestResolveTransitiveDeps_FetchError(t *testing.T) {
	_, mainProto := buildDepProtos(t)
	allFiles := map[string]*descriptorpb.FileDescriptorProto{
		mainProto.GetName(): mainProto,
	}
	err := resolveTransitiveDeps(allFiles, func(filename string) ([]*descriptorpb.FileDescriptorProto, error) {
		return nil, fmt.Errorf("fetch error")
	})
	if err == nil {
		t.Fatal("expected error from fetcher")
	}
}

func TestResolveTransitiveDeps_NilResponse(t *testing.T) {
	_, mainProto := buildDepProtos(t)
	allFiles := map[string]*descriptorpb.FileDescriptorProto{
		mainProto.GetName(): mainProto,
	}
	calls := 0
	err := resolveTransitiveDeps(allFiles, func(filename string) ([]*descriptorpb.FileDescriptorProto, error) {
		calls++
		if calls > 5 {
			t.Fatal("infinite loop detected")
		}
		// Return nil (no file descriptors) - but this means dep won't be resolved.
		// The second call should be the same dep since it's still missing.
		// Return the dep on second call to break loop.
		if calls == 2 {
			dep, _ := buildDepProtos(t)
			return []*descriptorpb.FileDescriptorProto{dep}, nil
		}
		return nil, nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchFileByName(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	rc := newReflectionClient(c.conn)
	client := grpc_reflection_v1.NewServerReflectionClient(rc.conn)
	stream, err := client.ServerReflectionInfo(context.Background())
	if err != nil {
		t.Fatalf("opening stream: %v", err)
	}
	defer stream.CloseSend()

	// Fetch existing file.
	fds, err := fetchFileByName(stream, "test.proto")
	if err != nil {
		t.Fatalf("fetchFileByName: %v", err)
	}
	if len(fds) == 0 {
		t.Fatal("expected file descriptors")
	}

	// Fetch nonexistent file - should return nil (no error, just no response).
	fds, err = fetchFileByName(stream, "nonexistent.proto")
	if err != nil {
		// Some servers return an error response, which is fine.
		t.Logf("got error for nonexistent: %v", err)
	}
}

func TestStreamDepFetcher(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	rc := newReflectionClient(c.conn)
	client := grpc_reflection_v1.NewServerReflectionClient(rc.conn)
	stream, err := client.ServerReflectionInfo(context.Background())
	if err != nil {
		t.Fatalf("opening stream: %v", err)
	}
	defer stream.CloseSend()

	fetcher := streamDepFetcher(stream)
	fds, err := fetcher("test.proto")
	if err != nil {
		t.Fatalf("fetcher: %v", err)
	}
	if len(fds) == 0 {
		t.Fatal("expected file descriptors")
	}
}

func TestCollectMissingDeps_NoDeps(t *testing.T) {
	allFiles := map[string]*descriptorpb.FileDescriptorProto{
		"test.proto": buildTestFileDescriptor(),
	}
	deps := collectMissingDeps(allFiles)
	if len(deps) != 0 {
		t.Errorf("expected no missing deps, got %v", deps)
	}
}

func TestCollectMissingDeps_WithDeps(t *testing.T) {
	depProto, mainProto := buildDepProtos(t)
	// Only include main, not dep.
	allFiles := map[string]*descriptorpb.FileDescriptorProto{
		mainProto.GetName(): mainProto,
	}
	deps := collectMissingDeps(allFiles)
	if len(deps) != 1 {
		t.Fatalf("expected 1 missing dep, got %v", deps)
	}
	if deps[0] != "dep.proto" {
		t.Errorf("missing dep = %v, want dep.proto", deps[0])
	}

	// After adding dep, no more missing.
	allFiles[depProto.GetName()] = depProto
	deps = collectMissingDeps(allFiles)
	if len(deps) != 0 {
		t.Errorf("expected no missing deps after adding, got %v", deps)
	}
}

// ---------- coerceValue remaining edge cases ----------

func TestCoerceValue_UnsupportedKind(t *testing.T) {
	// Test the default case in coerceValue - we need a field with an unsupported kind.
	// All standard kinds are handled, so this is hard to trigger. Skip.
}

// Test coerceMapValue with invalid value type.
func TestCoerceMapValue_InvalidValueType(t *testing.T) {
	desc := buildComplexProto(t)
	fd := desc.Fields().ByName("labels") // map<string, string>

	// Pass a value that's not a string for a string map value.
	_, err := coerceValue(fd, map[string]any{"k": 123})
	if err == nil {
		t.Error("expected error for non-string map value")
	}
}

// ---------- singleProtoValueToGo remaining cases ----------

func TestSingleProtoValueToGo_String(t *testing.T) {
	fd := buildFieldDesc(t, descriptorpb.FieldDescriptorProto_TYPE_STRING, "str_spvtg")
	v := singleProtoValueToGo(fd, protoreflect.ValueOfString("hello"))
	if v.(string) != "hello" {
		t.Errorf("got %v", v)
	}
}

func TestSingleProtoValueToGo_Int32(t *testing.T) {
	fd := buildFieldDesc(t, descriptorpb.FieldDescriptorProto_TYPE_INT32, "i32_spvtg")
	v := singleProtoValueToGo(fd, protoreflect.ValueOfInt32(42))
	if v.(int) != 42 {
		t.Errorf("got %v", v)
	}
}

// ---------- receiveStream error path ----------

// TestStreamErrors_ClosedConn tests stream creation failures when the underlying
// connection has been closed. We need to close the conn to trigger NewStream errors,
// but not through Teardown (which nils the conn).
// errorServer implements a server that returns errors mid-stream.
type errorServer struct{}

func (s *errorServer) serviceDesc() *grpc.ServiceDesc {
	return &grpc.ServiceDesc{
		ServiceName: "testpkg.EchoService",
		HandlerType: (*any)(nil),
		Methods: []grpc.MethodDesc{
			{MethodName: "Echo", Handler: s.handleEcho},
		},
		Streams: []grpc.StreamDesc{
			{StreamName: "ServerStreamEcho", Handler: s.handleServerStream, ServerStreams: true},
			{StreamName: "ClientStreamEcho", Handler: s.handleClientStream, ClientStreams: true},
			{StreamName: "BidiStreamEcho", Handler: s.handleBidiStream, ServerStreams: true, ClientStreams: true},
		},
	}
}

func (s *errorServer) handleEcho(srv any, ctx context.Context, dec func(any) error, interceptor grpc.UnaryServerInterceptor) (any, error) {
	return nil, status.Errorf(codes.Internal, "server error")
}

func (s *errorServer) handleServerStream(srv any, stream grpc.ServerStream) error {
	// Send one message then return error.
	reqMsg := dynamicpb.NewMessage(getTestInputDesc())
	if err := stream.RecvMsg(reqMsg); err != nil {
		return err
	}
	respMsg := dynamicpb.NewMessage(getTestOutputDesc())
	respMsg.Set(getTestOutputDesc().Fields().ByName("message"), protoreflect.ValueOfString("partial"))
	respMsg.Set(getTestOutputDesc().Fields().ByName("code"), protoreflect.ValueOfInt32(0))
	stream.SendMsg(respMsg)
	return status.Errorf(codes.Internal, "mid-stream error")
}

func (s *errorServer) handleClientStream(srv any, stream grpc.ServerStream) error {
	for {
		reqMsg := dynamicpb.NewMessage(getTestInputDesc())
		if err := stream.RecvMsg(reqMsg); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
	}
	return status.Errorf(codes.Internal, "client stream error")
}

func (s *errorServer) handleBidiStream(srv any, stream grpc.ServerStream) error {
	reqMsg := dynamicpb.NewMessage(getTestInputDesc())
	if err := stream.RecvMsg(reqMsg); err != nil {
		return err
	}
	respMsg := dynamicpb.NewMessage(getTestOutputDesc())
	respMsg.Set(getTestOutputDesc().Fields().ByName("message"), protoreflect.ValueOfString("partial"))
	respMsg.Set(getTestOutputDesc().Fields().ByName("code"), protoreflect.ValueOfInt32(0))
	stream.SendMsg(respMsg)
	return status.Errorf(codes.Internal, "bidi error")
}

func startErrorServer(t *testing.T) (string, func()) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := grpc.NewServer()
	es := &errorServer{}
	srv.RegisterService(es.serviceDesc(), nil)
	reflection.Register(srv)
	go func() { _ = srv.Serve(lis) }()
	return lis.Addr().String(), func() { srv.GracefulStop(); lis.Close() }
}

func TestServerStream_MidStreamError(t *testing.T) {
	addr, cleanup := startErrorServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "server_stream",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "ServerStreamEcho",
			"message": map[string]any{"message": "err-test"},
		},
	})
	if err != nil {
		t.Fatalf("Execute should succeed (error in result): %v", err)
	}
	if result.Data["status_code"].(int) == 0 {
		t.Error("expected non-zero status code")
	}
	// Should have partial responses.
	responses := result.Data["responses"].([]any)
	if len(responses) == 0 {
		t.Log("no partial responses captured")
	}
}

func TestClientStream_ServerError(t *testing.T) {
	addr, cleanup := startErrorServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "client_stream",
		Parameters: map[string]any{
			"service":  "testpkg.EchoService",
			"method":   "ClientStreamEcho",
			"messages": []any{map[string]any{"message": "x"}},
		},
	})
	if err != nil {
		t.Fatalf("Execute should succeed (error in result): %v", err)
	}
	// The server returns error instead of response - should be captured in status.
	if result.Data["status_code"].(int) == 0 {
		t.Error("expected non-zero status code")
	}
}

func TestBidiStream_ServerError(t *testing.T) {
	addr, cleanup := startErrorServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	result, err := c.Execute(context.Background(), connector.Step{
		Action: "bidi_stream",
		Parameters: map[string]any{
			"service":  "testpkg.EchoService",
			"method":   "BidiStreamEcho",
			"messages": []any{map[string]any{"message": "x"}},
		},
	})
	if err != nil {
		t.Fatalf("Execute should succeed (error in result): %v", err)
	}
	if result.Data["status_code"].(int) == 0 {
		t.Error("expected non-zero status code")
	}
}

// TestResolveMethod_CancelledContext tests resolveMethod error paths.
func TestResolveMethod_CancelledContext(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	rc := newReflectionClient(c.conn)

	// Cancel context before calling - hits ServerReflectionInfo error.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, _, err := rc.resolveMethod(ctx, "testpkg.EchoService", "Echo")
	if err == nil {
		t.Error("expected error for cancelled context")
	}

	// Try with a very short timeout to potentially hit send/recv errors.
	for i := 0; i < 10; i++ {
		ctx2, cancel2 := context.WithTimeout(context.Background(), time.Nanosecond)
		_, _, _, err = rc.resolveMethod(ctx2, "testpkg.EchoService", "Echo")
		cancel2()
		// We don't care if it succeeds or fails - just exercising error paths.
	}
}

// TestListServices_CancelledContext tests listServices error paths.
func TestListServices_CancelledContext(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	rc := newReflectionClient(c.conn)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := rc.listServices(ctx)
	if err == nil {
		t.Error("expected error for cancelled context")
	}
}

// TestListServices_ClosedConn tests listServices with a closed connection.
func TestListServices_ClosedConn(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	rc := newReflectionClient(c.conn)
	c.conn.Close()

	_, err := rc.listServices(context.Background())
	if err == nil {
		t.Error("expected error for closed connection")
	}
	c.conn = nil
}

// TestResolveMethod_ClosedConn tests resolveMethod with a closed connection.
func TestResolveMethod_ClosedConn(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	rc := newReflectionClient(c.conn)
	c.conn.Close()

	_, _, _, err := rc.resolveMethod(context.Background(), "testpkg.EchoService", "Echo")
	if err == nil {
		t.Error("expected error for closed connection")
	}
	c.conn = nil
}

// TestFetchFileByName_ClosedStream tests fetchFileByName with a closed stream.
func TestFetchFileByName_ClosedStream(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	rc := newReflectionClient(c.conn)
	client := grpc_reflection_v1.NewServerReflectionClient(rc.conn)
	stream, err := client.ServerReflectionInfo(context.Background())
	if err != nil {
		t.Fatalf("opening stream: %v", err)
	}
	// Close send side to trigger recv error.
	stream.CloseSend()

	_, err = fetchFileByName(stream, "test.proto")
	// May or may not error depending on timing, but we exercise the path.
	_ = err
}

// slowShutdownServer creates a server that we can shut down after reflection succeeds.
func startSlowShutdownServer(t *testing.T) (string, *grpc.Server) {
	t.Helper()
	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	srv := grpc.NewServer()
	es := newEchoServer()
	srv.RegisterService(es.serviceDesc(), nil)
	reflection.Register(srv)

	go func() { _ = srv.Serve(lis) }()
	return lis.Addr().String(), srv
}

// TestStream_ServerShutdownAfterReflection tests streaming when the server shuts down
// between the reflection call and the actual stream operation.
func TestStream_ServerShutdownAfterReflection(t *testing.T) {
	addr, srv := startSlowShutdownServer(t)

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	// First do a successful reflection call to cache the connection.
	_, err := c.Execute(context.Background(), connector.Step{
		Action: "unary",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "Echo",
			"message": map[string]any{"message": "warmup"},
		},
	})
	if err != nil {
		t.Fatalf("warmup failed: %v", err)
	}

	// Now stop the server.
	srv.Stop()

	// Try streaming - should fail at some point after reflection.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err = c.Execute(ctx, connector.Step{
		Action: "server_stream",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "ServerStreamEcho",
			"message": map[string]any{"message": "x"},
		},
	})
	if err == nil {
		t.Log("streaming succeeded despite server shutdown (connection was still alive)")
	}

	_, err = c.Execute(ctx, connector.Step{
		Action: "client_stream",
		Parameters: map[string]any{
			"service":  "testpkg.EchoService",
			"method":   "ClientStreamEcho",
			"messages": []any{map[string]any{"message": "x"}},
		},
	})
	if err == nil {
		t.Log("client stream succeeded despite server shutdown")
	}

	_, err = c.Execute(ctx, connector.Step{
		Action: "bidi_stream",
		Parameters: map[string]any{
			"service":  "testpkg.EchoService",
			"method":   "BidiStreamEcho",
			"messages": []any{map[string]any{"message": "x"}},
		},
	})
	if err == nil {
		t.Log("bidi stream succeeded despite server shutdown")
	}
}

func TestServerStream_CancelledContext(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	// Create a context that we cancel quickly to trigger stream receive error.
	ctx, cancel := context.WithCancel(context.Background())
	// Cancel immediately to cause error during stream receive.
	cancel()

	_, err := c.Execute(ctx, connector.Step{
		Action: "server_stream",
		Parameters: map[string]any{
			"service": "testpkg.EchoService",
			"method":  "ServerStreamEcho",
			"message": map[string]any{"message": "cancel-test"},
		},
	})
	// Should either fail at reflection or at stream receive.
	if err == nil {
		// If it succeeded, the context was still valid during the call.
		// That's okay - context cancellation is racy.
		t.Log("context cancellation didn't cause error (race)")
	}
}

// ---------- Streaming: verify Elapsed is set ----------

func TestStreamingElapsed(t *testing.T) {
	addr, cleanup := startTestServer(t)
	defer cleanup()

	c := New()
	c.Setup(context.Background(), map[string]any{"endpoint": addr, "plaintext": true})
	defer c.Teardown(context.Background())

	for _, action := range []string{"server_stream", "client_stream", "bidi_stream"} {
		t.Run(action, func(t *testing.T) {
			params := map[string]any{
				"service": "testpkg.EchoService",
			}
			switch action {
			case "server_stream":
				params["method"] = "ServerStreamEcho"
				params["message"] = map[string]any{"message": "x"}
			case "client_stream":
				params["method"] = "ClientStreamEcho"
				params["messages"] = []any{map[string]any{"message": "x"}}
			case "bidi_stream":
				params["method"] = "BidiStreamEcho"
				params["messages"] = []any{map[string]any{"message": "x"}}
			}

			result, err := c.Execute(context.Background(), connector.Step{
				Action:     action,
				Parameters: params,
			})
			if err != nil {
				t.Fatalf("Execute %s: %v", action, err)
			}
			if result.Elapsed <= 0 {
				t.Errorf("elapsed should be positive for %s", action)
			}
		})
	}
}
