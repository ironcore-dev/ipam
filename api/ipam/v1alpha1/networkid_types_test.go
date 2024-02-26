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
	"math"
	"math/big"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/json"
)

var _ = Describe("NetworkID marshalling and unmarshalling", func() {
	Context("When JSON is deserialized to NetworkID", func() {
		It("Should accept integers or numeric strings", func() {
			By("Deserializing integers to NetworkID")
			integerJsons := []string{
				`-11`,
				`0`,
				`12345`,
			}

			for _, j := range integerJsons {
				expected := big.Int{}
				_, ok := expected.SetString(j, 10)
				Expect(ok).To(BeTrue())

				id := &NetworkID{}
				Expect(json.Unmarshal([]byte(j), id)).Should(Succeed())
				Expect(id.Int).To(Equal(expected))
			}

			By("Deserializing numeric strings to NetworkID")
			numericStringJsons := []string{
				`"12"`,
				`"314"`,
				`"0"`,
				`"-222"`,
			}

			for _, j := range numericStringJsons {
				expected := big.Int{}
				_, ok := expected.SetString(strings.Trim(j, "\""), 10)
				Expect(ok).To(BeTrue())

				id := &NetworkID{}
				Expect(json.Unmarshal([]byte(j), id)).Should(Succeed())
				Expect(id.Int).To(Equal(expected))
			}

			By("Deserializing null to empty struct values")
			nullStringJson := `null`
			id := NetworkID{}

			Expect(json.Unmarshal([]byte(nullStringJson), &id)).Should(Succeed())
			Expect(id).To(Equal(NetworkID{}))
		})
	})

	Context("When NetworkID is serialized to Json", func() {
		It("Should be transformed to JSON string value", func() {
			By("Serializing NetworkID to numeric string")
			expectedMap := map[int64]string{
				222:           `"222"`,
				-34222:        `"-34222"`,
				0:             `"0"`,
				math.MaxInt64: `"9223372036854775807"`,
				math.MinInt64: `"-9223372036854775808"`,
			}

			for k, v := range expectedMap {
				id := NetworkID{
					*big.NewInt(k),
				}

				b, err := json.Marshal(id)
				Expect(err).NotTo(HaveOccurred())
				Expect(string(b)).To(BeEquivalentTo(v))
			}
		})
	})

	Context("When NetworkID is deep copied", func() {
		It("Should not copy inner pointers", func() {
			By("Copying ID")
			id := NetworkID{
				*big.NewInt(100),
			}
			copyId := NetworkID{}

			id.DeepCopyInto(&copyId)

			By("Changing initial ID")
			id.Add(&id.Int, big.NewInt(10))

			Expect(id.Cmp(big.NewInt(110))).To(BeZero())
			Expect(copyId.Cmp(big.NewInt(100))).To(BeZero())
		})
	})

	Context("When NetworkIDs are compared", func() {
		It("Should match when both are nil", func() {
			var left *NetworkID
			var right *NetworkID

			Expect(left.Eq(right)).To(BeTrue())
		})

		It("Should match when comparing to itself", func() {
			one := NetworkIDFromInt64(1)

			Expect(one.Eq(one)).To(BeTrue())
		})

		It("Should match when IDs are same", func() {
			left := NetworkIDFromInt64(100)
			right := NetworkIDFromInt64(100)

			Expect(left.Eq(right)).To(BeTrue())
		})

		It("Shouldn't match when left is nil", func() {
			var left *NetworkID
			right := NetworkIDFromInt64(100)

			Expect(left.Eq(right)).To(BeFalse())
		})

		It("Shouldn't match when right is nil", func() {
			left := NetworkIDFromInt64(100)
			var right *NetworkID

			Expect(left.Eq(right)).To(BeFalse())
		})

		It("Shouldn't match when IDs are different", func() {
			left := NetworkIDFromInt64(100)
			right := NetworkIDFromInt64(1001)

			Expect(left.Eq(right)).To(BeFalse())
		})
	})
})
