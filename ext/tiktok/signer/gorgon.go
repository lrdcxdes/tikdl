package signer

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

type Gorgon struct {
	Params  string
	Unix    int64
	Data    string
	Cookies string
}

func NewGorgon(
	params string,
	unix int64,
	data string,
	cookies string,
) *Gorgon {
	return &Gorgon{
		Params:  params,
		Unix:    unix,
		Data:    data,
		Cookies: cookies,
	}
}

func (g *Gorgon) Hash(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func (g *Gorgon) GetBaseString() string {
	baseStr := g.Hash(g.Params)

	if g.Data != "" {
		baseStr += g.Hash(g.Data)
	} else {
		baseStr += strings.Repeat("0", 32)
	}

	if g.Cookies != "" {
		baseStr += g.Hash(g.Cookies)
	} else {
		baseStr += strings.Repeat("0", 32)
	}

	return baseStr
}

func (g *Gorgon) GetValue() map[string]string {
	return g.Encrypt(g.GetBaseString())
}

func (g *Gorgon) Encrypt(data string) map[string]string {
	length := 0x14
	key := []byte{
		0xDF, 0x77, 0xB9, 0x40, 0xB9, 0x9B, 0x84, 0x83,
		0xD1, 0xB9, 0xCB, 0xD1, 0xF7, 0xC2, 0xB9, 0x85,
		0xC3, 0xD0, 0xFB, 0xC3,
	}

	paramList := make([]int, 0, 20)
	for i := 0; i < 12; i += 4 {
		temp := data[i*8 : (i+1)*8]
		for j := range 4 {
			hexVal := 0
			fmt.Sscanf(temp[j*2:(j+1)*2], "%x", &hexVal)
			paramList = append(paramList, hexVal)
		}
	}
	paramList = append(paramList, 0x0, 0x6, 0xB, 0x1C)
	h := uint32(g.Unix)
	paramList = append(paramList,
		int((h&0xFF000000)>>24),
		int((h&0x00FF0000)>>16),
		int((h&0x0000FF00)>>8),
		int(h&0x000000FF),
	)

	eorResultList := make([]int, len(paramList))
	for i, v := range paramList {
		eorResultList[i] = v ^ int(key[i])
	}

	for i := range length {
		c := g.Reverse(eorResultList[i])
		d := eorResultList[(i+1)%length]
		e := c ^ d
		f := g.RbitAlgorithm(e)
		h := ((f ^ 0xFFFFFFFF) ^ length) & 0xFF
		eorResultList[i] = h
	}
	result := ""
	for _, param := range eorResultList {
		result += g.HexString(param)
	}
	return map[string]string{
		"ticket":  strconv.FormatInt(g.Unix*1000, 10),
		"khronos": strconv.FormatInt(g.Unix, 10),
		"gorgon":  "0404b0d30000" + result,
	}
}

func (g *Gorgon) RbitAlgorithm(num int) int {
	binaryStr := fmt.Sprintf("%08b", num&0xFF)

	result := 0
	for i, c := range binaryStr {
		if c == '1' {
			result |= 1 << (7 - i)
		}
	}

	return result
}

func (g *Gorgon) HexString(num int) string {
	return fmt.Sprintf("%02x", num&0xFF)
}

func (g *Gorgon) Reverse(num int) int {
	hexStr := g.HexString(num)
	result := 0
	fmt.Sscanf(hexStr[1:]+hexStr[:1], "%x", &result)
	return result
}
