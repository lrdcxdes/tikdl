package signer

import "bytes"

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - (len(data) % blockSize)
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padtext...)
}

func paddingSize(size int) int {
	mod := size % 16
	if mod > 0 {
		return size + (16 - mod)
	}
	return size
}
