package types_test

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/agaffney/glispr/types"
)

func TestNewNumberFromString(t *testing.T) {
	testDefs := []struct {
		val         string
		radix       int
		expectedNum *types.Number
	}{
		{val: "123", radix: 10, expectedNum: types.NewNumberInteger(big.NewInt(123))},
		{val: "0", radix: 10, expectedNum: types.NewNumberInteger(big.NewInt(0))},
		{val: "-0", radix: 10, expectedNum: types.NewNumberInteger(big.NewInt(0))},
		{val: "+6", radix: 10, expectedNum: types.NewNumberInteger(big.NewInt(6))},
		{val: "+28", radix: 10, expectedNum: types.NewNumberInteger(big.NewInt(28))},
		{val: "-1", radix: 10, expectedNum: types.NewNumberInteger(big.NewInt(-1))},
		{val: "15511210043330985984000000", radix: 10, expectedNum: types.NewNumberInteger(new(big.Int).MulRange(1, 25))},
		{val: "11010101", radix: 2, expectedNum: types.NewNumberInteger(big.NewInt(213))},
		{val: "325", radix: 8, expectedNum: types.NewNumberInteger(big.NewInt(213))},
		{val: "D5", radix: 16, expectedNum: types.NewNumberInteger(big.NewInt(213))},
		{val: "FF.", radix: 16, expectedNum: types.NewNumberInteger(big.NewInt(255))},
		{val: "-7H", radix: 25, expectedNum: types.NewNumberInteger(big.NewInt(-192))},
		{val: "ACCEDED", radix: 16, expectedNum: types.NewNumberInteger(big.NewInt(181202413))},
		{val: "2/3", radix: 10, expectedNum: types.NewNumberRatio(big.NewRat(2, 3))},
		{val: "4/6", radix: 10, expectedNum: types.NewNumberRatio(big.NewRat(4, 6))},
		{val: "-17/23", radix: 10, expectedNum: types.NewNumberRatio(big.NewRat(-17, 23))},
		{val: "10/5", radix: 10, expectedNum: types.NewNumberRatio(big.NewRat(10, 5))},
		{val: "-101/75", radix: 8, expectedNum: types.NewNumberRatio(big.NewRat(-65, 61))},
		{val: "120/21", radix: 3, expectedNum: types.NewNumberRatio(big.NewRat(15, 7))},
		{val: "bc/ad", radix: 16, expectedNum: types.NewNumberRatio(big.NewRat(188, 173))},
		{val: "FADED/FACADE", radix: 16, expectedNum: types.NewNumberRatio(big.NewRat(1027565, 16435934))},
		{val: "0.0", radix: 10, expectedNum: types.NewNumberFloat(big.NewFloat(0.0))},
		{val: "0E0", radix: 10, expectedNum: types.NewNumberFloat(big.NewFloat(0.0))},
		{val: "3.1415926535897932384", radix: 10, expectedNum: types.NewNumberFloat(big.NewFloat(3.1415926535897932384e0))},
	}
	for _, testDef := range testDefs {
		tmp := types.NewNumberFromString(testDef.val, testDef.radix)
		if tmp == nil {
			t.Errorf("NewNumberFromString(%q, %d) returned nil", testDef.val, testDef.radix)
			continue
		}
		if !tmp.Equal(testDef.expectedNum) {
			t.Errorf("NewNumberFromString(%q, %d) does not match expected value\n\n  Expected: %s\n  Actual: %s", testDef.val, testDef.radix, testDef.expectedNum.String(), tmp.String())
		}
	}
}

func TestNewNumberFromStringZeroDenominator(t *testing.T) {
	for _, val := range []string{"1/0", "-1/0", "0/0"} {
		assert.NotPanics(t, func() {
			assert.Nil(t, types.NewNumberFromString(val, 10), "expected %q to not be a number", val)
		})
	}
}
