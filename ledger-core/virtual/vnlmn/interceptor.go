package vnlmn

import (
	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/rms/rmsreg"
)

func LRegisterRequestInterceptor(record rmsreg.GoGoSerializable) ([]rmsreg.GoGoSerializable, bool) {
	request, ok := record.(*rms.LRegisterRequest)
	if !ok {
		return nil, false
	}

	var result []rmsreg.GoGoSerializable

	switch request.Flags {
	case rms.RegistrationFlags_FastSafe:
		result = append(result, &rms.LRegisterResponse{
			Flags:              rms.RegistrationFlags_Fast,
			AnticipatedRef:     request.AnticipatedRef,
			RegistrarSignature: rms.NewBytes([]byte("dummy")),
		})

		result = append(result, &rms.LRegisterResponse{
			Flags:              rms.RegistrationFlags_Safe,
			AnticipatedRef:     request.AnticipatedRef,
			RegistrarSignature: rms.NewBytes([]byte("dummy")),
		})

	case rms.RegistrationFlags_Safe, rms.RegistrationFlags_Fast:
		result = append(result, &rms.LRegisterResponse{
			Flags:              request.Flags,
			AnticipatedRef:     request.AnticipatedRef,
			RegistrarSignature: rms.NewBytes([]byte("dummy")),
		})
	}

	return result, true
}
