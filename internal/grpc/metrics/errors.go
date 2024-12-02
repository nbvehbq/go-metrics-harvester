package metrics

import (
	"google.golang.org/genproto/googleapis/rpc/errdetails"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const Domain = "nbvehbq.ru"

func internalError(err error) error {
	st := status.New(codes.Internal, "internal error")
	ei := errdetails.ErrorInfo{
		Reason: err.Error(),
		Domain: Domain,
	}

	st, derr := st.WithDetails(&ei)
	if derr != nil {
		return derr
	}

	return st.Err()
}

func argumentError(err error) error {
	st := status.New(codes.InvalidArgument, "bad request")
	ei := errdetails.ErrorInfo{
		Reason: err.Error(),
		Domain: Domain,
	}

	st, derr := st.WithDetails(&ei)
	if derr != nil {
		return derr
	}

	return st.Err()
}
