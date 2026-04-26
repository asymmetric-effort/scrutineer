package grpc

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// statusCodeName returns the human-readable name for a gRPC status code.
func statusCodeName(code codes.Code) string {
	switch code {
	case codes.OK:
		return "OK"
	case codes.Canceled:
		return "CANCELLED"
	case codes.Unknown:
		return "UNKNOWN"
	case codes.InvalidArgument:
		return "INVALID_ARGUMENT"
	case codes.DeadlineExceeded:
		return "DEADLINE_EXCEEDED"
	case codes.NotFound:
		return "NOT_FOUND"
	case codes.AlreadyExists:
		return "ALREADY_EXISTS"
	case codes.PermissionDenied:
		return "PERMISSION_DENIED"
	case codes.ResourceExhausted:
		return "RESOURCE_EXHAUSTED"
	case codes.FailedPrecondition:
		return "FAILED_PRECONDITION"
	case codes.Aborted:
		return "ABORTED"
	case codes.OutOfRange:
		return "OUT_OF_RANGE"
	case codes.Unimplemented:
		return "UNIMPLEMENTED"
	case codes.Internal:
		return "INTERNAL"
	case codes.Unavailable:
		return "UNAVAILABLE"
	case codes.DataLoss:
		return "DATA_LOSS"
	case codes.Unauthenticated:
		return "UNAUTHENTICATED"
	default:
		return "UNKNOWN"
	}
}

// extractStatus extracts the gRPC status code and message from an error.
// If err is nil, returns OK status. If the error is not a gRPC status error,
// returns Unknown with the error message.
func extractStatus(err error) (codes.Code, string) {
	if err == nil {
		return codes.OK, ""
	}
	st, ok := status.FromError(err)
	if !ok {
		return codes.Unknown, err.Error()
	}
	return st.Code(), st.Message()
}
