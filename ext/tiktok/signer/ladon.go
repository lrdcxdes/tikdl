package signer

import (
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"strconv"
)

func NewLadon(
	unix int64,
	licenseID string,
	appID string,
) (string, error) {
	randomBytes := make([]byte, 4)
	if _, err := rand.Read(randomBytes); err != nil {
		return "", err
	}
	unixString := strconv.FormatInt(unix, 10)
	data := unixString + "-" + licenseID + "-" + appID

	keygen := randomBytes
	keygen = append(keygen, []byte(appID)...)
	hash := md5.Sum(keygen)
	mdHex := hex.EncodeToString(hash[:])

	size := len(data)
	newSize := paddingSize(size)

	output := make([]byte, newSize+4)
	copy(output[:4], randomBytes)

	encryptedData := encryptLadon([]byte(mdHex), []byte(data), size)
	copy(output[4:], encryptedData)

	return base64.StdEncoding.EncodeToString(output), nil
}

func getTypeData(buffer []byte, index int) uint64 {
	return binary.LittleEndian.Uint64(buffer[index*8 : (index+1)*8])
}

func setTypeData(buffer []byte, index int, data uint64) {
	binary.LittleEndian.PutUint64(buffer[index*8:(index+1)*8], data)
}

func validate(num uint64) uint64 {
	return num & 0xFFFFFFFFFFFFFFFF
}

func ror64(value, count uint64) uint64 {
	count %= 64
	return ((value >> count) | (value << (64 - count))) & 0xFFFFFFFFFFFFFFFF
}

func encryptLadonInput(hashTable, inputData []byte) []byte {
	data0 := binary.LittleEndian.Uint64(inputData[:8])
	data1 := binary.LittleEndian.Uint64(inputData[8:])

	for i := range hashTable[:0x22*8] {
		if i%8 != 0 {
			continue
		}
		hash := binary.LittleEndian.Uint64(hashTable[i : i+8])
		data1 = validate(hash ^ (data0 + ((data1 >> 8) | (data1 << (64 - 8)))))
		data0 = validate(data1 ^ ((data0 >> 0x3D) | (data0 << (64 - 0x3D))))
	}

	outputData := make([]byte, 16)
	binary.LittleEndian.PutUint64(outputData[:8], data0)
	binary.LittleEndian.PutUint64(outputData[8:], data1)

	return outputData
}

func encryptLadon(md5hex, data []byte, size int) []byte {
	hashTable := make([]byte, 272+16)
	copy(hashTable[:32], md5hex)

	temp := make([]uint64, 0, 4)
	for i := range 4 {
		temp = append(temp, getTypeData(hashTable, i))
	}

	bufferB0 := temp[0]
	bufferB8 := temp[1]
	temp = temp[2:]

	for i := range 0x22 {
		x9 := bufferB0
		x8 := bufferB8
		x8 = validate(ror64(x8, 8))
		x8 = validate(x8 + x9)
		x8 = validate(x8 ^ uint64(i))
		temp = append(temp, x8)
		x8 = validate(x8 ^ ror64(x9, 61))
		setTypeData(hashTable, i+1, x8)
		bufferB0 = x8
		bufferB8 = temp[0]
		temp = temp[1:]
	}

	newSize := paddingSize(size)

	input := make([]byte, newSize)
	copy(input[:size], data)
	pkcs7PaddingPadBuffer(input, size, newSize, 16)

	output := make([]byte, newSize)
	for i := range newSize / 16 {
		copy(output[i*16:(i+1)*16], encryptLadonInput(hashTable, input[i*16:(i+1)*16]))
	}

	return output
}

func pkcs7PaddingPadBuffer(buffer []byte, dataLength, bufferSize, modulus int) int {
	padByte := byte(modulus - (dataLength % modulus))
	if dataLength+int(padByte) > bufferSize {
		return -int(padByte)
	}

	for i := range int(padByte) {
		buffer[dataLength+i] = padByte
	}

	return int(padByte)
}
