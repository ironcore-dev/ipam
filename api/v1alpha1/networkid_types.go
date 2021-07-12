package v1alpha1

import (
	"fmt"
	"math/big"
	"strings"

	"k8s.io/apimachinery/pkg/util/json"
)

// NetworkID represents an incremental ID for network type.
// Effectively it is a wrapper around big.Int,
// as its Bytes() method allows to get unsigned big endian
// representation for the value.
// +kubebuilder:validation:Type=string
type NetworkID struct {
	big.Int `json:"-"`
}

func NetworkIDFromInt64(i64 int64) *NetworkID {
	return &NetworkID{
		*big.NewInt(i64),
	}
}

func NetworkIDFromBigInt(bi *big.Int) *NetworkID {
	return &NetworkID{
		*bi,
	}
}

func NetworkIDFromBytes(b []byte) *NetworkID {
	bi := &big.Int{}
	bi = bi.SetBytes(b)
	return &NetworkID{
		*bi,
	}
}

func (in NetworkID) MarshalJSON() ([]byte, error) {
	return json.Marshal(in.String())
}

func (in *NetworkID) UnmarshalJSON(b []byte) error {
	stringVal := string(b)
	if stringVal == "null" {
		return nil
	}
	// If it starts with quote, it is expected to be numeric string
	// otherwise ti is expected to be just a JSON number
	if strings.HasPrefix(stringVal, "\"") {
		if err := json.Unmarshal(b, &stringVal); err != nil {
			return err
		}
	}

	var bi big.Int
	_, ok := bi.SetString(stringVal, 10)
	if !ok {
		return fmt.Errorf("unable to set string value to big int %s", b)
	}

	in.Int = bi

	return nil
}

func (in *NetworkID) DeepCopyInto(out *NetworkID) {
	*out = *in
	bi := new(big.Int).Set(&in.Int)
	out.Int = *bi
	return
}
