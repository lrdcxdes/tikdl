package signer

import (
	"encoding/binary"
	"encoding/hex"
	"math/bits"
)

type SM3 struct {
	iv []uint32
	tj []uint32
}

func NewSM3() *SM3 {
	return &SM3{
		iv: []uint32{
			0x7380166F, 0x4914B2B9, 0x172442D7, 0xDA8A0600,
			0xA96F30BC, 0x163138AA, 0xE38DEE4D, 0xB0FB0E4E,
		},
		tj: func() []uint32 {
			tj := make([]uint32, 64)
			for j := range tj {
				if j < 16 {
					tj[j] = 0x79CC4519
				} else {
					tj[j] = 0x7A879D8A
				}
			}
			return tj
		}(),
	}
}

func (s *SM3) rotateLeft(a uint32, k int) uint32 {
	return bits.RotateLeft32(a, k)
}

func (s *SM3) ffj(x, y, z uint32, j int) uint32 {
	if j < 16 {
		return x ^ y ^ z
	}
	return (x & y) | (x & z) | (y & z)
}

func (s *SM3) ggj(x, y, z uint32, j int) uint32 {
	if j < 16 {
		return x ^ y ^ z
	}
	return (x & y) | ((^x) & z)
}

func (s *SM3) p0(x uint32) uint32 {
	return x ^ s.rotateLeft(x, 9) ^ s.rotateLeft(x, 17)
}

func (s *SM3) p1(x uint32) uint32 {
	return x ^ s.rotateLeft(x, 15) ^ s.rotateLeft(x, 23)
}

func (s *SM3) cf(vi []uint32, bi []byte) []uint32 {
	w := make([]uint32, 68)

	for i := range 16 {
		w[i] = binary.BigEndian.Uint32(bi[i*4 : (i+1)*4])
	}

	for j := range w[16:] {
		j += 16
		w[j] = s.p1(w[j-16]^w[j-9]^s.rotateLeft(w[j-3], 15)) ^
			s.rotateLeft(w[j-13], 7) ^ w[j-6]
	}

	w1 := make([]uint32, 64)
	for j := range 64 {
		w1[j] = w[j] ^ w[j+4]
	}

	a, b, c, d, e, f, g, h := vi[0], vi[1], vi[2], vi[3], vi[4], vi[5], vi[6], vi[7]

	for j := range 64 {
		ss1 := s.rotateLeft(s.rotateLeft(a, 12)+e+s.rotateLeft(s.tj[j], j), 7)
		ss2 := ss1 ^ s.rotateLeft(a, 12)
		tt1 := s.ffj(a, b, c, j) + d + ss2 + w1[j]
		tt2 := s.ggj(e, f, g, j) + h + ss1 + w[j]
		d = c
		c = s.rotateLeft(b, 9)
		b = a
		a = tt1
		h = g
		g = s.rotateLeft(f, 19)
		f = e
		e = s.p0(tt2)
	}

	return []uint32{
		a ^ vi[0], b ^ vi[1], c ^ vi[2], d ^ vi[3],
		e ^ vi[4], f ^ vi[5], g ^ vi[6], h ^ vi[7],
	}
}

func (s *SM3) SM3Hash(msg []byte) []byte {
	paddedMsg := make([]byte, len(msg))
	copy(paddedMsg, msg)

	paddedMsg = append(paddedMsg, 0x80)

	for len(paddedMsg)%64 != 56 {
		paddedMsg = append(paddedMsg, 0x00)
	}

	bitLength := uint64(len(msg) * 8)
	lengthBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(lengthBytes, bitLength)
	paddedMsg = append(paddedMsg, lengthBytes...)

	groupCount := len(paddedMsg) / 64
	blocks := make([][]byte, groupCount)

	for i := range groupCount {
		blocks[i] = paddedMsg[i*64 : (i+1)*64]
	}

	v := make([][]uint32, groupCount+1)
	v[0] = make([]uint32, 8)
	copy(v[0], s.iv)

	for i := range groupCount {
		v[i+1] = s.cf(v[i], blocks[i])
	}

	result := make([]byte, 32)
	for i, val := range v[groupCount] {
		binary.BigEndian.PutUint32(result[i*4:], val)
	}

	return result
}

func GetBodyHash(stub string) []byte {
	sm3 := NewSM3()

	if stub == "" {
		return sm3.SM3Hash(make([]byte, 16))[:6]
	}

	stubBytes, err := hex.DecodeString(stub)
	if err != nil {
		return sm3.SM3Hash(make([]byte, 16))[:6]
	}
	return sm3.SM3Hash(stubBytes)[:6]
}

func GetQueryHash(query string) []byte {
	sm3 := NewSM3()

	if query == "" {
		return sm3.SM3Hash(make([]byte, 16))[:6]
	}

	return sm3.SM3Hash([]byte(query))[:6]
}
