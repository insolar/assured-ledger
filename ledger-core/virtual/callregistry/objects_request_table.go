package callregistry

import (
	"github.com/insolar/assured-ledger/ledger-core/reference"
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
)

// ObjectsResultCallRegistry is used for store results for all processed requests per object by object reference.
type ObjectsResultCallRegistry struct {
	objects map[reference.Global]ObjectCallResults
}

type ObjectCallResults struct {
	CallResults map[reference.Global]CallSummary
}

func NewObjectRequestTable() ObjectsResultCallRegistry {
	return ObjectsResultCallRegistry{
		objects: make(map[reference.Global]ObjectCallResults),
	}
}

func (ort ObjectsResultCallRegistry) GetObjectCallResults(objectRef reference.Global) (ObjectCallResults, bool) {
	workingTable, ok := ort.objects[objectRef]
	return workingTable, ok
}

func (ort ObjectsResultCallRegistry) AddObjectCallResults(objectRef reference.Global, callResults ObjectCallResults) bool {
	if _, exist := ort.objects[objectRef]; exist {
		return false
	}
	ort.objects[objectRef] = callResults

	return true
}

func (ort ObjectsResultCallRegistry) AddObjectCallResult(objectRef reference.Global, reqRef reference.Global, result *rms.VCallResult) {
	requests, ok := ort.GetObjectCallResults(objectRef)
	if !ok {
		// we should have summary result for object if we finish operation.
		panic(throw.IllegalState())
	}

	callSummary, ok := requests.CallResults[reqRef]
	if !ok {
		// if we not have summary in resultMap, it mean we do not go through stepStartRequestProcessing,
		// and we can't have a result here
		panic(throw.IllegalState())
	}

	if callSummary.Result != nil {
		// we should not have result because migration was before we publish result in object
		panic(throw.IllegalState())
	}

	requests.CallResults[reqRef] = CallSummary{Result: result}
}
