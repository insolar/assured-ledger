package recordchecker

import (
	"context"
	"time"

	"github.com/insolar/assured-ledger/ledger-core/rms"
	"github.com/insolar/assured-ledger/ledger-core/vanilla/throw"
	"github.com/insolar/assured-ledger/ledger-core/virtual/integration/mock/publisher"
)

func RecordFromLRegisterRequest(msg rms.LRegisterRequest) rms.BasicRecord {
	rlv := msg.AnyRecordLazy.TryGetLazy()
	if rlv.IsZero() {
		panic(throw.IllegalValue())
	}

	actualRecord, err := rlv.Unmarshal()
	if err != nil {
		panic(throw.IllegalValue())
	}
	return actualRecord
}

func ProduceResponse(ctx context.Context, sender publisher.Sender) ProduceResponseFunc {
	return func(request rms.LRegisterRequest) {
		if request.Flags == rms.RegistrationFlags_FastSafe {
			sender.SendPayload(ctx, &rms.LRegisterResponse{
				Flags:              rms.RegistrationFlags_Fast,
				AnticipatedRef:     request.AnticipatedRef,
				RegistrarSignature: rms.NewBytes([]byte(request.AnticipatedRef.GetValue().String())),
			})

			time.Sleep(10 * time.Millisecond)

			sender.SendPayload(ctx, &rms.LRegisterResponse{
				Flags:              rms.RegistrationFlags_Safe,
				AnticipatedRef:     request.AnticipatedRef,
				RegistrarSignature: rms.NewBytes([]byte(request.AnticipatedRef.GetValue().String())),
			})
		} else {
			sender.SendPayload(ctx, &rms.LRegisterResponse{
				Flags:              request.Flags,
				AnticipatedRef:     request.AnticipatedRef,
				RegistrarSignature: rms.NewBytes([]byte(request.AnticipatedRef.GetValue().String())),
			})
		}
	}
}
