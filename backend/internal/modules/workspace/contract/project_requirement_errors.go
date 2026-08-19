package contract

import (
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type LegacyRequirementMutationDisabledError struct{}

func (LegacyRequirementMutationDisabledError) Error() string {
	return ErrLegacyRequirementMutationDisabled.Error()
}

func (LegacyRequirementMutationDisabledError) Unwrap() error {
	return ErrLegacyRequirementMutationDisabled
}

func (LegacyRequirementMutationDisabledError) GRPCStatus() *status.Status {
	return status.New(codes.FailedPrecondition, ErrLegacyRequirementMutationDisabled.Error())
}
