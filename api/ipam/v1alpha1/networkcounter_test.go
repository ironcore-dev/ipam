// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and IronCore contributors
// SPDX-License-Identifier: Apache-2.0

package v1alpha1

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Network counter", func() {
	Context("When network counter has available intervals", func() {
		It("Should propose to book first determined value from first available interval", func() {
			cases := successfulProposeCases()

			for i, c := range cases {
				By(fmt.Sprintf("Case %d", i))

				proposedVal, err := c.initial.Propose()
				Expect(err).NotTo(HaveOccurred())
				Expect(proposedVal.Cmp(&c.testID.Int)).To(BeZero())
				Expect(c.initial.Reserve(proposedVal)).To(Succeed())
				Expect(c.initial).To(Equal(c.resulting))
			}
		})
	})

	Context("When network counter has no available intervals", func() {
		It("Should return an error on proposal attempt", func() {
			cases := failedProposeCases()

			for i, c := range cases {
				By(fmt.Sprintf("Case %d", i))

				proposedVal, err := c.initial.Propose()
				Expect(err).To(HaveOccurred())
				Expect(proposedVal).To(BeNil())
				Expect(c.initial.Reserve(NetworkIDFromInt64(1))).NotTo(Succeed())
				Expect(c.initial).To(Equal(c.resulting))
			}
		})
	})

	Context("When provided network ID is within available interval", func() {
		It("Should reserve network ID successfully", func() {
			cases := successfulReserveCases()

			for i, c := range cases {
				By(fmt.Sprintf("Case %d", i))

				Expect(c.initial.CanReserve(c.testID)).To(BeTrue())
				Expect(c.initial.Reserve(c.testID)).To(Succeed())
				Expect(c.initial).To(Equal(c.resulting))
			}
		})
	})

	Context("When provided network ID is out of bound of available intervals", func() {
		It("Should not reserve interval", func() {
			cases := failedReserveCases()

			for i, c := range cases {
				By(fmt.Sprintf("Case %d", i))

				Expect(c.initial.CanReserve(c.testID)).To(BeFalse())
				Expect(c.initial.Reserve(c.testID)).NotTo(Succeed())
				Expect(c.initial).To(Equal(c.resulting))
			}
		})
	})

	Context("When provided network ID is out of bound of available intervals", func() {
		It("Should release network ID successfully", func() {
			cases := successfulReleaseCases()

			for i, c := range cases {
				By(fmt.Sprintf("Case %d", i))

				Expect(c.initial.CanReserve(c.testID)).To(BeFalse())
				Expect(c.initial.Release(c.testID)).To(Succeed())
				Expect(c.initial).To(Equal(c.resulting))
			}
		})
	})

	Context("When provided network ID is within available interval", func() {
		It("Should not release network ID", func() {
			cases := failedReleaseCases()

			for i, c := range cases {
				By(fmt.Sprintf("Case %d", i))

				Expect(c.initial.CanReserve(c.testID)).To(BeTrue())
				Expect(c.initial.Release(c.testID)).NotTo(Succeed())
				Expect(c.initial).To(Equal(c.resulting))
			}
		})
	})
})

type testCase struct {
	initial   NetworkCounterSpec
	testID    *NetworkID
	resulting NetworkCounterSpec
}

func successfulProposeCases() []testCase {
	return []testCase{
		// init [[-inf; +inf]]
		// should propose 0
		// result [[-inf; -1], [1; +inf]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{},
				},
			},
			testID: NetworkIDFromInt64(0),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(-1),
					},
					{
						Begin: NetworkIDFromInt64(1),
					},
				},
			},
		},
		// init [[-inf; -1], [1; +inf]]
		// should propose -1
		// result [[-inf; -2], [1; +inf]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(-1),
					},
					{
						Begin: NetworkIDFromInt64(1),
					},
				},
			},
			testID: NetworkIDFromInt64(-1),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(-2),
					},
					{
						Begin: NetworkIDFromInt64(1),
					},
				},
			},
		},
		// init [[-inf; -1]]
		// should propose -1
		// result [[-inf; -2]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(-1),
					},
				},
			},
			testID: NetworkIDFromInt64(-1),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(-2),
					},
				},
			},
		},
		// init [[-5; -1]]
		// should propose -5
		// result [[-4; -1]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(-5),
						End:   NetworkIDFromInt64(-1),
					},
				},
			},
			testID: NetworkIDFromInt64(-5),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(-4),
						End:   NetworkIDFromInt64(-1),
					},
				},
			},
		},
		// init [[-3; -2]]
		// should propose -3
		// result [[-2]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(-3),
						End:   NetworkIDFromInt64(-2),
					},
				},
			},
			testID: NetworkIDFromInt64(-3),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(-2),
					},
				},
			},
		},
		// init [[-10; -8], [-6, -5]]
		// should propose -10
		// result [[-9; -8], [-6, -5]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(-10),
						End:   NetworkIDFromInt64(-8),
					},
					{
						Begin: NetworkIDFromInt64(-6),
						End:   NetworkIDFromInt64(-5),
					},
				},
			},
			testID: NetworkIDFromInt64(-10),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(-9),
						End:   NetworkIDFromInt64(-8),
					},
					{
						Begin: NetworkIDFromInt64(-6),
						End:   NetworkIDFromInt64(-5),
					},
				},
			},
		},
		// init [[1; +inf]]
		// should propose 1
		// result [[2; +inf]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
					},
				},
			},
			testID: NetworkIDFromInt64(1),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(2),
					},
				},
			},
		},
		// init [[1; 3]]
		// should propose 1
		// result [[2; 3]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(1),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(2),
						End:   NetworkIDFromInt64(3),
					},
				},
			},
		},
		// init [[2; 3]]
		// should propose 2
		// result [[3]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(2),
						End:   NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(2),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(3),
					},
				},
			},
		},
		// init [[2; 5], [8, 10]]
		// should propose 2
		// result [[3; 5], [8, 10]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(2),
						End:   NetworkIDFromInt64(5),
					},
					{
						Begin: NetworkIDFromInt64(8),
						End:   NetworkIDFromInt64(10),
					},
				},
			},
			testID: NetworkIDFromInt64(2),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(3),
						End:   NetworkIDFromInt64(5),
					},
					{
						Begin: NetworkIDFromInt64(8),
						End:   NetworkIDFromInt64(10),
					},
				},
			},
		},
		// init [[3], [8; 10]]
		// should propose 3
		// result [[8; 10]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(3),
					},
					{
						Begin: NetworkIDFromInt64(8),
						End:   NetworkIDFromInt64(10),
					},
				},
			},
			testID: NetworkIDFromInt64(3),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(8),
						End:   NetworkIDFromInt64(10),
					},
				},
			},
		},
		// init [[3]]
		// should propose 3
		// result []
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(3),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{},
			},
		},
	}
}

func failedProposeCases() []testCase {
	return []testCase{
		// init []
		// should propose nil
		// result []
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{},
			},
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{},
			},
		},
	}
}

func successfulReserveCases() []testCase {
	return []testCase{
		// init [[-inf; 0], [3]]
		// reserve 3
		// result [[-inf; 0]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(0),
					},
					{
						Exact: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(3),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(0),
					},
				},
			},
		},
		// init [[-inf; 0], [3; 5]]
		// reserve 3
		// result [[-inf; 0], [4; 5]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(0),
					},
					{
						Begin: NetworkIDFromInt64(3),
						End:   NetworkIDFromInt64(5),
					},
				},
			},
			testID: NetworkIDFromInt64(3),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(0),
					},
					{
						Begin: NetworkIDFromInt64(4),
						End:   NetworkIDFromInt64(5),
					},
				},
			},
		},
		// init [[-inf; 0], [3; 5]]
		// reserve 4
		// result [[-inf; 0], [3], [5]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(0),
					},
					{
						Begin: NetworkIDFromInt64(3),
						End:   NetworkIDFromInt64(5),
					},
				},
			},
			testID: NetworkIDFromInt64(4),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(0),
					},
					{
						Exact: NetworkIDFromInt64(3),
					},
					{
						Exact: NetworkIDFromInt64(5),
					},
				},
			},
		},
		// init [[-inf; 0], [3; 5]]
		// reserve 5
		// result [[-inf; 0], [3; 4]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(0),
					},
					{
						Begin: NetworkIDFromInt64(3),
						End:   NetworkIDFromInt64(5),
					},
				},
			},
			testID: NetworkIDFromInt64(5),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(0),
					},
					{
						Begin: NetworkIDFromInt64(3),
						End:   NetworkIDFromInt64(4),
					},
				},
			},
		},
		// init [[-inf; 0], [3; 5]]
		// reserve -100
		// result [[-inf; -101], [-99; 0], [3; 5]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(0),
					},
					{
						Begin: NetworkIDFromInt64(3),
						End:   NetworkIDFromInt64(5),
					},
				},
			},
			testID: NetworkIDFromInt64(-100),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(-101),
					},
					{
						Begin: NetworkIDFromInt64(-99),
						End:   NetworkIDFromInt64(0),
					},
					{
						Begin: NetworkIDFromInt64(3),
						End:   NetworkIDFromInt64(5),
					},
				},
			},
		},
		// init [[-inf; 0], [3], [6; 7]]
		// reserve 3
		// result [[-inf; 0], [6; 7]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(0),
					},
					{
						Exact: NetworkIDFromInt64(3),
					},
					{
						Begin: NetworkIDFromInt64(6),
						End:   NetworkIDFromInt64(7),
					},
				},
			},
			testID: NetworkIDFromInt64(3),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(0),
					},
					{
						Begin: NetworkIDFromInt64(6),
						End:   NetworkIDFromInt64(7),
					},
				},
			},
		},
		// init [[-inf; 0], [3; 5], [8; 10]]
		// reserve 4
		// result [[-inf; 0], [3], [5], [8; 10]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(0),
					},
					{
						Begin: NetworkIDFromInt64(3),
						End:   NetworkIDFromInt64(5),
					},
					{
						Begin: NetworkIDFromInt64(8),
						End:   NetworkIDFromInt64(10),
					},
				},
			},
			testID: NetworkIDFromInt64(4),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(0),
					},
					{
						Exact: NetworkIDFromInt64(3),
					},
					{
						Exact: NetworkIDFromInt64(5),
					},
					{
						Begin: NetworkIDFromInt64(8),
						End:   NetworkIDFromInt64(10),
					},
				},
			},
		},
	}
}

func failedReserveCases() []testCase {
	return []testCase{
		// init []
		// reserve 0
		// result []
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{},
			},
			testID: NetworkIDFromInt64(0),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{},
			},
		},
		// init [[3]]
		// reserve 2
		// result [[3]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(2),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(3),
					},
				},
			},
		},
		// init [[3]]
		// reserve 4
		// result [[3]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(4),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(3),
					},
				},
			},
		},
		// init [[3; +inf]]
		// reserve 2
		// result [[3]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(2),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(3),
					},
				},
			},
		},
		// init [[-inf; 3]]
		// reserve 4
		// result [[3]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(4),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(3),
					},
				},
			},
		},
		// init [[1; 3]]
		// reserve 4
		// result [[1; 3]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(4),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(3),
					},
				},
			},
		},
		// init [[1; 3]]
		// reserve 0
		// result [[1; 3]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(0),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(3),
					},
				},
			},
		},
		// init [[1; 3], [5; 8]]
		// reserve 4
		// result [[1; 3], [5; 8]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(3),
					},
					{
						Begin: NetworkIDFromInt64(5),
						End:   NetworkIDFromInt64(8),
					},
				},
			},
			testID: NetworkIDFromInt64(4),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(3),
					},
					{
						Begin: NetworkIDFromInt64(5),
						End:   NetworkIDFromInt64(8),
					},
				},
			},
		},
	}
}

func successfulReleaseCases() []testCase {
	return []testCase{
		// initial [[3; +inf]]
		// release 2
		// result [[2; +inf]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(2),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(2),
					},
				},
			},
		},
		// initial [[3; +inf]]
		// release 1
		// result [[1], [3; +inf]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(1),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(1),
					},
					{
						Begin: NetworkIDFromInt64(3),
					},
				},
			},
		},
		// initial [[-inf; 3]]
		// release 4
		// result [[-inf; 4]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(4),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(4),
					},
				},
			},
		},
		// initial [[-inf; 3]]
		// release 5
		// result [[-inf; 3], [5]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(5),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(3),
					},
					{
						Exact: NetworkIDFromInt64(5),
					},
				},
			},
		},
		// initial [[-inf; 3], [5; +inf]]
		// release 4
		// result [[-inf; +inf]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(3),
					},
					{
						Begin: NetworkIDFromInt64(5),
					},
				},
			},
			testID: NetworkIDFromInt64(4),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{},
				},
			},
		},
		// initial [[1; 3], [5; 8]]
		// release 4
		// result [[1; 8]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(3),
					},
					{
						Begin: NetworkIDFromInt64(5),
						End:   NetworkIDFromInt64(8),
					},
				},
			},
			testID: NetworkIDFromInt64(4),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(8),
					},
				},
			},
		},
		// initial [[1; 3], [7; 8]]
		// release 5
		// result [[1; 3], [5], [6; 8]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(3),
					},
					{
						Begin: NetworkIDFromInt64(7),
						End:   NetworkIDFromInt64(8),
					},
				},
			},
			testID: NetworkIDFromInt64(5),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(3),
					},
					{
						Exact: NetworkIDFromInt64(5),
					},
					{
						Begin: NetworkIDFromInt64(7),
						End:   NetworkIDFromInt64(8),
					},
				},
			},
		},
		// initial [[1; 2], [5; 8]]
		// release 4
		// result [[1; 2], [4; 8]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(2),
					},
					{
						Begin: NetworkIDFromInt64(5),
						End:   NetworkIDFromInt64(8),
					},
				},
			},
			testID: NetworkIDFromInt64(4),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(2),
					},
					{
						Begin: NetworkIDFromInt64(4),
						End:   NetworkIDFromInt64(8),
					},
				},
			},
		},
		// initial [[1; 2], [5; 8]]
		// release 3
		// result [[1; 3], [5; 8]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(2),
					},
					{
						Begin: NetworkIDFromInt64(5),
						End:   NetworkIDFromInt64(8),
					},
				},
			},
			testID: NetworkIDFromInt64(3),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(1),
						End:   NetworkIDFromInt64(3),
					},
					{
						Begin: NetworkIDFromInt64(5),
						End:   NetworkIDFromInt64(8),
					},
				},
			},
		},
		// initial [[3]]
		// release 4
		// result [[3; 4]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(4),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(3),
						End:   NetworkIDFromInt64(4),
					},
				},
			},
		},
		// initial [[3]]
		// release 2
		// result [[2; 3]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(2),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(2),
						End:   NetworkIDFromInt64(3),
					},
				},
			},
		},
		// initial [[3]]
		// release 1
		// result [[1], [3]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(1),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(1),
					},
					{
						Exact: NetworkIDFromInt64(3),
					},
				},
			},
		},
		// initial [[3]]
		// release 5
		// result [[3], [5]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(5),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(3),
					},
					{
						Exact: NetworkIDFromInt64(5),
					},
				},
			},
		},
		// initial [[100], [102; 1000]]
		// release 101
		// result [[100; 1000]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(100),
					},
					{
						Begin: NetworkIDFromInt64(102),
						End:   NetworkIDFromInt64(1000),
					},
				},
			},
			testID: NetworkIDFromInt64(101),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(100),
						End:   NetworkIDFromInt64(1000),
					},
				},
			},
		},
	}
}

func failedReleaseCases() []testCase {
	return []testCase{
		// init [[3]]
		// release 3
		// result [[3]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(3),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Exact: NetworkIDFromInt64(3),
					},
				},
			},
		},
		// init [[-inf; 3]]
		// release 0
		// result [[-inf; 3]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(0),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(3),
					},
				},
			},
		},
		// init [[-inf; 3]]
		// release 3
		// result [[-inf; 3]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(3),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						End: NetworkIDFromInt64(3),
					},
				},
			},
		},
		// init [[3; +inf]]
		// release 10
		// result [[3; +inf]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(10),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(3),
					},
				},
			},
		},
		// init [[3; +inf]]
		// release 3
		// result [[3; +inf]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(3),
					},
				},
			},
			testID: NetworkIDFromInt64(3),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(3),
					},
				},
			},
		},
		// init [[-inf; +inf]]
		// release 3
		// result [[-inf; +inf]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{},
				},
			},
			testID: NetworkIDFromInt64(3),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{},
				},
			},
		},
		// init [[-5; 5]]
		// release 3
		// result [[-5; 5]]
		{
			initial: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(-5),
						End:   NetworkIDFromInt64(5),
					},
				},
			},
			testID: NetworkIDFromInt64(3),
			resulting: NetworkCounterSpec{
				Vacant: []NetworkIDInterval{
					{
						Begin: NetworkIDFromInt64(-5),
						End:   NetworkIDFromInt64(5),
					},
				},
			},
		},
	}
}
