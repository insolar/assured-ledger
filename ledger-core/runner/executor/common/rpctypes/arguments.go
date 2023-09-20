package rpctypes

import (
	"encoding/json"

	errors "github.com/insolar/assured-ledger/ledger-core/vanilla/throw"

	"github.com/insolar/assured-ledger/ledger-core/insolar"
)

// Arguments is a dedicated type for arguments, that represented as binary cbored blob
type Arguments []byte

// MarshalJSON uncbor Arguments slice recursively
func (args *Arguments) MarshalJSON() ([]byte, error) {
	result := make([]interface{}, 0)

	err := convertArgs(*args, &result)
	if err != nil {
		return nil, err
	}

	return json.Marshal(&result)
}

func convertArgs(args []byte, result *[]interface{}) error {
	var value interface{}
	err := insolar.Deserialize(args, &value)
	if err != nil {
		return errors.W(err, "Can't deserialize record")
	}

	tmp, ok := value.([]interface{})
	if !ok {
		*result = append(*result, value)
		return nil
	}

	inner := make([]interface{}, 0)

	for _, slItem := range tmp {
		switch v := slItem.(type) {
		case []byte:
			err := convertArgs(v, result)
			if err != nil {
				return err
			}
		default:
			inner = append(inner, v)
		}
	}

	*result = append(*result, inner)

	return nil
}
