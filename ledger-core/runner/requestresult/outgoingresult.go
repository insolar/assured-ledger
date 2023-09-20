package requestresult

type OutgoingExecutionResult struct {
	initialized     bool
	ExecutionResult []byte
	Error           error
}

func (r OutgoingExecutionResult) IsEmpty() bool {
	return !r.initialized
}

func NewOutgoingExecutionResult(result []byte, err error) OutgoingExecutionResult {
	return OutgoingExecutionResult{
		initialized:     true,
		ExecutionResult: result,
		Error:           err,
	}
}
