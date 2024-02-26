// Copyright 2023 OnMetal authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1alpha1

import (
	"fmt"
	"math/big"
	"strings"

	"k8s.io/apimachinery/pkg/util/json"
)

// +kubebuilder:validation:Type=string

// NetworkID represents an incremental ID for network type.
// Effectively it is a wrapper around big.Int,
// as its Bytes() method allows to get unsigned big endian
// representation for the value.
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

func (in *NetworkID) Eq(r *NetworkID) bool {
	if in == r {
		return true
	}
	if in == nil || r == nil {
		return false
	}
	return in.Int.Cmp(&r.Int) == 0
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
}
