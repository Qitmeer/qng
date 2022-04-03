// Copyright (c) 2017-2020 The qitmeer developers
// license that can be found in the LICENSE file.
// Reference resources of rust bitVector
package pow

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/Qitmeer/qng/common/hash"
	"github.com/Qitmeer/qng/core/json"
	"github.com/Qitmeer/qng/crypto/cuckoo"
	"math"
	"math/big"
	"strconv"
	"strings"
)

type Cuckoo struct {
	Pow
}

const (
	// 8 extra bytes + 6 position
	EXTRA_DATA_START            = 8
	PROOF_DATA_EDGE_BITS_START  = 14
	PROOF_DATA_EDGE_BITS_END    = 15
	PROOF_DATA_CIRCLE_NONCE_END = 169
	MAX_EDGE_VALUE_NONCE_COUNT  = 42 - 14
	// MAX 3 bytes value BigEndian([]byte{255,255,255,0})
	MAX_CRITICAL_EDGE_VALUE = 4294967040
	FILLIN                  = "000000"
)

var tenToAny = map[int]string{0: "0", 1: "1", 2: "2", 3: "3", 4: "4", 5: "5", 6: "6", 7: "7", 8: "8", 9: "9", 10: "a", 11: "b", 12: "c", 13: "d", 14: "e", 15: "f", 16: "g", 17: "h", 18: "i", 19: "j", 20: "k", 21: "l", 22: "m", 23: "n", 24: "o", 25: "p", 26: "q", 27: "r", 28: "s", 29: "t", 30: "u", 31: "v", 32: "w", 33: "x", 34: "y", 35: "z", 36: ":", 37: ";", 38: "<", 39: "=", 40: ">", 41: "?", 42: "@", 43: "[", 44: "]", 45: "^", 46: "_", 47: "{", 48: "|", 49: "}", 50: "A", 51: "B", 52: "C", 53: "D", 54: "E", 55: "F", 56: "G", 57: "H", 58: "I", 59: "J", 60: "K", 61: "L", 62: "M", 63: "N", 64: "O", 65: "P", 66: "Q", 67: "R", 68: "S", 69: "T", 70: "U", 71: "V", 72: "W", 73: "X", 74: "Y", 75: "Z"}

func (this *Cuckoo) GetPowResult() json.PowResult {
	return json.PowResult{
		PowName: PowMapString[this.GetPowType()].(string),
		PowType: uint8(this.GetPowType()),
		Nonce:   this.GetNonce(),
		ProofData: &json.ProofData{
			EdgeBits:     int(this.ProofData[PROOF_DATA_EDGE_BITS_START:PROOF_DATA_EDGE_BITS_END][0]),
			CircleNonces: hex.EncodeToString(this.ProofData[PROOF_DATA_EDGE_BITS_END:PROOF_DATA_CIRCLE_NONCE_END]),
		},
	}
}

// set edge bits
func (this *Cuckoo) SetEdgeBits(edge_bits uint8) {
	copy(this.ProofData[PROOF_DATA_EDGE_BITS_START:PROOF_DATA_EDGE_BITS_END], []byte{edge_bits})
}

// get edge bits
func (this *Cuckoo) GetEdgeBits() uint8 {
	return uint8(this.ProofData[PROOF_DATA_EDGE_BITS_START:PROOF_DATA_EDGE_BITS_END][0])
}

// set small edge circle
func (this *Cuckoo) SetSmallEdge(b []byte, position int) {
	copy(this.ProofData[position:position+3], b[:3])
}

// set big edge circle
func (this *Cuckoo) SetBigEdge(b []byte, position int) {
	copy(this.ProofData[position:position+4], b)
}

// set edge circles
func (this *Cuckoo) SetCircleEdges(edges []uint32) {
	maxCritialEdgeCount := 0
	position := PROOF_DATA_EDGE_BITS_END
	smallIndexes := make([]int, 0)
	for i := 0; i < len(edges); i++ {
		if maxCritialEdgeCount > MAX_EDGE_VALUE_NONCE_COUNT {
			return
		}
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, edges[i])
		if edges[i] > MAX_CRITICAL_EDGE_VALUE || len(smallIndexes) >= len(edges)-MAX_EDGE_VALUE_NONCE_COUNT {
			maxCritialEdgeCount++
			this.SetBigEdge(b, i)
			position += 4
		} else {
			this.SetSmallEdge(b, i)
			position += 3
			smallIndexes = append(smallIndexes, i)
		}
	}
	positionBytes := ConvertSmallIndexesToBytes(smallIndexes)
	copy(this.ProofData[EXTRA_DATA_START:PROOF_DATA_EDGE_BITS_START], positionBytes)
}

func FillBinary(s string, n int) string {
	l := len(s)
	if l < n {
		s = FILLIN[:n-l] + s
	}
	return s
}

func ConvertSmallIndexesToBytes(smallIndexes []int) []byte {
	s := ""
	for i := 0; i < len(smallIndexes); i++ {
		s += FillBinary(decimalToAny(smallIndexes[i], 2), 6)
	}
	b := BinaryStringToBytes(s)
	for j := 0; j < 6-len(b); j++ {
		b = append([]byte{0}, b...)
	}
	return b
}

func ConvertPositionBytesToSmallIndexes(b []byte) []int {
	smallIndexes := make([]int, 0)
	s := FillBinary(BytesToBinaryString(b), 48)
	s = strings.ReplaceAll(s, "[", "")
	s = strings.ReplaceAll(s, "]", "")
	s = strings.ReplaceAll(s, " ", "")
	for i := 0; i < 48; i += 6 {
		smallIndexes = append(smallIndexes, anyToDecimal(s[i:i+6], 2))
	}
	return smallIndexes
}

func (this *Cuckoo) GetCircleNonces() (nonces [cuckoo.ProofSize]uint32) {
	arr := this.ConvertBytesToUint32Array(this.ProofData[PROOF_DATA_EDGE_BITS_END:PROOF_DATA_CIRCLE_NONCE_END])
	copy(nonces[:cuckoo.ProofSize], arr[:cuckoo.ProofSize])
	return
}

func (this *Cuckoo) ConvertBytesToUint32Array(data []byte) []uint32 {
	smallIndexes := SmallIndex(ConvertPositionBytesToSmallIndexes(this.ProofData[EXTRA_DATA_START:PROOF_DATA_EDGE_BITS_START]))
	nonces := make([]uint32, 0)
	nonceBytes := make([]byte, 0)
	l := len(data)
	for i := 0; i < l; {
		if smallIndexes.Has(i) {
			nonceBytes = append([]byte{0}, data[i:i+3]...)
			nonces = append(nonces, binary.LittleEndian.Uint32(nonceBytes))
			i += 3
		} else {
			nonceBytes = data[i : i+4]
			nonces = append(nonces, binary.LittleEndian.Uint32(nonceBytes))
			i += 4
		}
	}
	return nonces
}

func InIntArray() {

}

//get sip hash
//first header data 113 bytes hash
func (this *Cuckoo) GetSipHash(headerData []byte) hash.Hash {
	return hash.HashH(headerData[:len(headerData)-PROOF_DATA_CIRCLE_NONCE_END])
}

//cuckoo pow proof data
func (this *Cuckoo) Bytes() PowBytes {
	r := make(PowBytes, 0)
	// write pow type 1 byte
	r = append(r, []byte{byte(this.PowType)}...)

	// write nonce 8 bytes
	n := make([]byte, 8)
	binary.LittleEndian.PutUint64(n, this.Nonce)
	r = append(r, n...)

	//write ProofData 169 bytes
	r = append(r, this.ProofData[:]...)
	return PowBytes(r)
}

// compare the target
// wether target match the target diff
func (this *Cuckoo) CompareDiff(newTarget *big.Int, target *big.Int) bool {
	return newTarget.Cmp(target) >= 0
}

// pow proof data
func (this *Cuckoo) BlockData() PowBytes {
	return this.Bytes()
}

func (this *Cuckoo) GraphWeight() uint64 { return 0 }

func decimalToAny(num, n int) string {
	new_num_str := ""
	var remainder int
	var remainder_string string
	for num != 0 {
		remainder = num % n
		if 76 > remainder && remainder > 9 {
			remainder_string = tenToAny[remainder]
		} else {
			remainder_string = strconv.Itoa(remainder)
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
