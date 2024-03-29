// Copyright 2017-2020 The qitmeer developers
// Copyright 2015 The Decred developers
// Copyright 2013, 2014 The btcsuite developers
// Use of this source code is governed by an ISC
// license that can be found in the LICENSE file.

package types

import (
	"bytes"
	"math"
	"reflect"
	"sort"
	"testing"
)

func TestAmountCreation(t *testing.T) {
	tests := []struct {
		name     string
		amount   float64
		valid    bool
		expected Amount
	}{
		// Positive tests.
		{
			name:     "zero",
			amount:   0,
			valid:    true,
			expected: Amount{0, MEERA},
		},
		{
			name:     "max producable",
			amount:   21e6,
			valid:    true,
			expected: Amount{MaxAmount, MEERA},
		},
		{
			name:     "min producable",
			amount:   -21e6,
			valid:    true,
			expected: Amount{-MaxAmount, MEERA},
		},
		{
			name:     "exceeds max producable",
			amount:   21e6 + 1e-8,
			valid:    true,
			expected: Amount{MaxAmount + 1, MEERA},
		},
		{
			name:     "exceeds min producable",
			amount:   -21e6 - 1e-8,
			valid:    true,
			expected: Amount{-MaxAmount - 1, MEERA},
		},
		{
			name:     "one hundred",
			amount:   100,
			valid:    true,
			expected: Amount{100 * AtomsPerCoin, MEERA},
		},
		{
			name:     "fraction",
			amount:   0.01234567,
			valid:    true,
			expected: Amount{1234567, MEERA},
		},
		{
			name:     "rounding up",
			amount:   54.999999999999943157,
			valid:    true,
			expected: Amount{55 * AtomsPerCoin, MEERA},
		},
		{
			name:     "rounding down",
			amount:   55.000000000000056843,
			valid:    true,
			expected: Amount{55 * AtomsPerCoin, MEERA},
		},

		// Negative tests.
		{
			name:   "not-a-number",
			amount: math.NaN(),
			valid:  false,
		},
		{
			name:   "-infinity",
			amount: math.Inf(-1),
			valid:  false,
		},
		{
			name:   "+infinity",
			amount: math.Inf(1),
			valid:  false,
		},
	}

	for _, test := range tests {
		a, err := NewAmount(test.amount)
		switch {
		case test.valid && err != nil:
			t.Errorf("%v: Positive test Amount creation failed with: %v", test.name, err)
			continue
		case !test.valid && err == nil:
			t.Errorf("%v: Negative test Amount creation succeeded (value %v) when should fail", test.name, a)
			continue
		}

		if *a != test.expected {
			t.Errorf("%v: Created amount %v does not match expected %v", test.name, a, test.expected)
			continue
		}
	}
}

func TestAmountUnitConversions(t *testing.T) {
	tests := []struct {
		name      string
		amount    Amount
		unit      AmountUnit
		converted float64
		s         string
	}{
		{
			name:      "MMEER",
			amount:    Amount{MaxAmount, MEERA},
			unit:      AmountMegaCoin,
			converted: 21,
			s:         "21 MMEER Asset",
		},
		{
			name:      "kMEER",
			amount:    Amount{44433322211100, MEERA},
			unit:      AmountKiloCoin,
			converted: 444.33322211100,
			s:         "444.333222111 kMEER Asset",
		},
		{
			name:      "MEER",
			amount:    Amount{44433322211100, MEERA},
			unit:      AmountCoin,
			converted: 444333.22211100,
			s:         "444333.222111 MEER Asset",
		},
		{
			name:      "mMEER",
			amount:    Amount{44433322211100, MEERA},
			unit:      AmountMilliCoin,
			converted: 444333222.11100,
			s:         "444333222.111 mMEER Asset",
		},
		{

			name:      "μMEER",
			amount:    Amount{44433322211100, MEERA},
			unit:      AmountMicroCoin,
			converted: 444333222111.00,
			s:         "444333222111 μMEER Asset",
		},
		{

			name:      "atom",
			amount:    Amount{44433322211100, MEERA},
			unit:      AmountAtom,
			converted: 44433322211100,
			s:         "44433322211100 atomMEER Asset",
		},
		{

			name:      "non-standard unit",
			amount:    Amount{44433322211100, MEERA},
			unit:      AmountUnit(-1),
			converted: 4443332.2211100,
			s:         "4443332.22111 1e-1 MEER Asset",
		},
	}

	for _, test := range tests {
		f := test.amount.ToUnit(test.unit)
		if f != test.converted {
			t.Errorf("%v: converted value %v does not match expected %v", test.name, f, test.converted)
			continue
		}

		s := test.amount.Format(test.unit)
		if s != test.s {
			t.Errorf("%v: format '%v' does not match expected '%v'", test.name, s, test.s)
			continue
		}

		// Verify that Amount.ToCoin works as advertised.
		f1 := test.amount.ToUnit(AmountCoin)
		f2 := test.amount.ToCoin()
		if f1 != f2 {
			t.Errorf("%v: ToCoin does not match ToUnit(AmountCoin): %v != %v", test.name, f1, f2)
		}

		// Verify that Amount.String works as advertised.
		s1 := test.amount.Format(AmountCoin)
		s2 := test.amount.String()
		if s1 != s2 {
			t.Errorf("%v: String does not match Format(AmountCoin): %v != %v", test.name, s1, s2)
		}
	}
}

func TestAmountMulF64(t *testing.T) {
	tests := []struct {
		name string
		amt  Amount
		mul  float64
		res  Amount
	}{
		{
			name: "Multiply 0.1 MEER by 2",
			amt:  Amount{100e5, MEERA}, // 0.1 MEER
			mul:  2,
			res:  Amount{200e5, MEERA}, // 0.2 MEER
		},
		{
			name: "Multiply 0.2 MEER by 0.02",
			amt:  Amount{200e5, MEERA}, // 0.2 MEER
			mul:  1.02,
			res:  Amount{204e5, MEERA}, // 0.204 MEER
		},
		{
			name: "Multiply 0.1 MEER by -2",
			amt:  Amount{100e5, MEERA}, // 0.1 MEER
			mul:  -2,
			res:  Amount{-200e5, MEERA}, // -0.2 MEER
		},
		{
			name: "Multiply 0.2 MEER by -0.02",
			amt:  Amount{200e5, MEERA}, // 0.2 MEER
			mul:  -1.02,
			res:  Amount{-204e5, MEERA}, // -0.204 MEER
		},
		{
			name: "Multiply -0.1 MEER by 2",
			amt:  Amount{-100e5, MEERA}, // -0.1 MEER
			mul:  2,
			res:  Amount{-200e5, MEERA}, // -0.2 MEER
		},
		{
			name: "Multiply -0.2 MEER by 0.02",
			amt:  Amount{-200e5, MEERA}, // -0.2 MEER
			mul:  1.02,
			res:  Amount{-204e5, MEERA}, // -0.204 MEER
		},
		{
			name: "Multiply -0.1 MEER by -2",
			amt:  Amount{-100e5, MEERA}, // -0.1 MEER
			mul:  -2,
			res:  Amount{200e5, MEERA}, // 0.2 MEER
		},
		{
			name: "Multiply -0.2 MEER by -0.02",
			amt:  Amount{-200e5, MEERA}, // -0.2 MEER
			mul:  -1.02,
			res:  Amount{204e5, MEERA}, // 0.204 MEER
		},
		{
			name: "Round down",
			amt:  Amount{49, MEERA}, // 49 Atoms MEER
			mul:  0.01,
			res:  Amount{0, MEERA},
		},
		{
			name: "Round up",
			amt:  Amount{50, MEERA}, // 50 Atom MEER
			mul:  0.01,
			res:  Amount{1, MEERA}, // 1 Atom MEER
		},
		{
			name: "Multiply by 0.",
			amt:  Amount{1e8, MEERA}, // 1 MEER
			mul:  0,
			res:  Amount{0, MEERA}, // 0 MEER
		},
		{
			name: "Multiply 1 by 0.5.",
			amt:  Amount{1, MEERA}, // 1 Atom MEER
			mul:  0.5,
			res:  Amount{1, MEERA}, // 1 Atom MEER
		},
		{
			name: "Multiply 100 by 66%.",
			amt:  Amount{100, MEERA}, // 100 Atom MEER
			mul:  0.66,
			res:  Amount{66, MEERA}, // 66 Atom MEER
		},
		{
			name: "Multiply 100 by 66.6%.",
			amt:  Amount{100, MEERA}, // 100 Atom MEER
			mul:  0.666,
			res:  Amount{67, MEERA}, // 67 Atom MEER
		},
		{
			name: "Multiply 100 by 2/3.",
			amt:  Amount{100, MEERA}, // 100 Atom MEER
			mul:  2.0 / 3,
			res:  Amount{67, MEERA}, // 67 Atoms MEER
		},
	}

	for _, test := range tests {
		a := test.amt.MulF64(test.mul)
		if *a != test.res {
			t.Errorf("%v: expected %v got %v", test.name, test.res, a)
		}
	}
}

func TestAmountSorter(t *testing.T) {
	tests := []struct {
		name string
		as   []Amount
		want []Amount
	}{
		{
			name: "Sort zero length slice of Amounts",
			as:   []Amount{},
			want: []Amount{},
		},
		{
			name: "Sort 1-element slice of Amounts",
			as:   []Amount{{7, MEERA}},
			want: []Amount{{7, MEERA}},
		},
		{
			name: "Sort 2-element slice of Amounts",
			as:   []Amount{{7, MEERA}, {5, MEERA}},
			want: []Amount{{5, MEERA}, {7, MEERA}},
		},
		{
			name: "Sort 6-element slice of Amounts",
			as: []Amount{
				{0, MEERA},
				{9e8, MEERA},
				{4e6, MEERA},
				{4e6, MEERA},
				{3, MEERA},
				{9e12, MEERA}},
			want: []Amount{
				{0, MEERA},
				{3, MEERA},
				{4e6, MEERA},
				{4e6, MEERA},
				{9e8, MEERA},
				{9e12, MEERA}},
		},
	}

	for i, test := range tests {
		result := make([]Amount, len(test.as))
		copy(result, test.as)
		sort.Sort(AmountSorter(result))
		if !reflect.DeepEqual(result, test.want) {
			t.Errorf("AmountSorter #%d got %v want %v", i, result,
				test.want)
			continue
		}
	}
}

func TestCheckCoinID(t *testing.T) {
	tests := []struct {
		name   string
		expect bool
		coinId CoinID
	}{
		{"meer", true, CoinID(0)},
		{"unknow", false, CoinID(4)},
	}

	for i, test := range tests {
		err := CheckCoinID(test.coinId)
		if test.expect == true && err != nil {
			t.Errorf("failed test[%d]:[%v] Check [%v] expect ok, but got err: %v", i, test.name, test.coinId, err)
		}
		if test.expect == false && err == nil {
			t.Errorf("failed test[%d]:[%v] Check [%v] expect failure, but got no err.", i, test.name, test.coinId)
		}
	}
}

func TestCoinID_Bytes(t *testing.T) {
	tests := []struct {
		id    CoinID
		byte  []byte
		equal bool
	}{
		{CoinID(0), []byte{0x0, 0x0}, true},
		{CoinID(1), []byte{0x1, 0x0}, true},
		{CoinID(2), []byte{0x2, 0x0}, true},
		{CoinID(2), []byte{0x0, 0x2}, false},
		{CoinID(2), []byte{0x2, 0x0, 0x0, 0x0}, false},
		{CoinID(255), []byte{0xff, 0x00}, true},
		{CoinID(256), []byte{0x00, 0x01}, true},
		{CoinID(65535), []byte{0xff, 0xff}, true},
	}
	for _, test := range tests {
		if equal := bytes.Compare(test.id.Bytes(), test.byte); equal == 0 != test.equal {
			t.Errorf("failed on test %v == %v is %v", test.id, test.byte, test.equal)
		}
	}

}
