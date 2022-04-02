// Copyright 2017-2018 The qitmeer developers

package types

import (
	"math"
	"strconv"
	"strings"
)

type Version uint32

const VERSION_BIT_POSITION = 10

var tenToAny = map[int]string{0: "0", 1: "1", 2: "2", 3: "3", 4: "4", 5: "5", 6: "6", 7: "7", 8: "8", 9: "9", 10: "a", 11: "b", 12: "c", 13: "d", 14: "e", 15: "f", 16: "g", 17: "h", 18: "i", 19: "j", 20: "k", 21: "l", 22: "m", 23: "n", 24: "o", 25: "p", 26: "q", 27: "r", 28: "s", 29: "t", 30: "u", 31: "v", 32: "w", 33: "x", 34: "y", 35: "z", 36: ":", 37: ";", 38: "<", 39: "=", 40: ">", 41: "?", 42: "@", 43: "[", 44: "]", 45: "^", 46: "_", 47: "{", 48: "|", 49: "}", 50: "A", 51: "B", 52: "C", 53: "D", 54: "E", 55: "F", 56: "G", 57: "H", 58: "I", 59: "J", 60: "K", 61: "L", 62: "M", 63: "N", 64: "O", 65: "P", 66: "Q", 67: "R", 68: "S", 69: "T", 70: "U", 71: "V", 72: "W", 73: "X", 74: "Y", 75: "Z"}

func (this *Version) GetVersion() uint32 {
	r := decimalToAny(uint32(*this), 2)
	l := len(r)
	if l > VERSION_BIT_POSITION {
		r = r[l-VERSION_BIT_POSITION:]
	}
	return uint32(anyToDecimal(r, 10))
}

func SetVersion(v uint32) Version {
	s := decimalToAny(v, 2)
	if len(s) > VERSION_BIT_POSITION {
		return 0
	}
	return Version(v)
}

func decimalToAny(num, n uint32) string {
	new_num_str := ""
	var remainder uint32
	var remainder_string string
	for num != 0 {
		remainder = num % n
		if 76 > remainder && remainder > 9 {
			remainder_string = tenToAny[int(remainder)]
		} else {
			remainder_string = strconv.Itoa(int(remainder))
		}
		new_num_str = remainder_string + new_num_str
		num = num / n
	}
	return new_num_str
}

func findkey(in string) int {
	result := -1
	for k, v := range tenToAny {
		if in == v {
			result = k
		}
	}
	return result
}

func anyToDecimal(num string, n int) int {
	var new_num float64
	new_num = 0.0
	nNum := len(strings.Split(num, "")) - 1
	for _, value := range strings.Split(num, "") {
		tmp := float64(findkey(value))
		if tmp != -1 {
			new_num = new_num + tmp*math.Pow(float64(n), float64(nNum))
			nNum = nNum - 1
		} else {
			break
		}
	}
	return int(new_num)
}
