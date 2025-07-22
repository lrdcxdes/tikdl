package signer

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"net/url"
	"slices"
)

type Argus struct{}

func NewArgus(
	params url.Values,
	data string,
	unix int64,
	appID string,
) (string, error) {
	argus := &Argus{}
	zeroBytes := make([]byte, 8)

	var bodyHash []byte
	if data != "" {
		bodyHash = GetBodyHash(data)
	} else {
		bodyHash = GetBodyHash("")
	}

	bean := map[int]any{
		1:  0x20200929 << 1,               // magic
		2:  2,                             // version
		3:  generateRandom(0x7FFFFFFF),    // random number (consistent with Python)
		4:  appID,                         // app ID as string
		5:  params.Get("device_id"),       // device ID
		6:  licenseID,                     // license ID as string
		7:  params.Get("app_version"),     // app version
		8:  sdkVersion,                    // SDK version string
		9:  sdkVersionCode,                // SDK version int
		10: zeroBytes,                     // env code (jailbreak detection)
		12: int(unix << 1),                // create time with bit shift
		13: bodyHash,                      // body hash
		14: GetQueryHash(params.Encode()), // query hash
		16: "",                            // secure device token (unused)
		20: "none",                        // PSK version
		21: 738,                           // call type
		25: 2,                             // ?
	}

	return argus.Encrypt(bean)
}

func (a *Argus) encryptEncPb(data []byte) []byte {
	dataList := make([]byte, len(data))
	copy(dataList, data)

	xorArray := dataList[:8]

	for i, v := range dataList[8:] {
		dataList[i+8] = v ^ xorArray[(i+8)%8]
	}
	slices.Reverse(dataList)

	return dataList
}

func (a *Argus) Encrypt(xargusBean map[int]any) (string, error) {
	pb, err := NewProtoBuf(xargusBean)
	if err != nil {
		return "", err
	}
	protobufBytes, err := pb.ToBuf()
	if err != nil {
		return "", err
	}
	hexStr := hex.EncodeToString(protobufBytes)
	protobufFromHex, err := hex.DecodeString(hexStr)
	if err != nil {
		return "", err
	}
	paddedData := pkcs7Pad(protobufFromHex, aes.BlockSize)
	newLen := len(paddedData)

	signKey := []byte{
		0xac, 0x1a, 0xda, 0xae, 0x95, 0xa7, 0xaf, 0x94,
		0xa5, 0x11, 0x4a, 0xb3, 0xb3, 0xa9, 0x7d, 0xd8,
		0x00, 0x50, 0xaa, 0x0a, 0x39, 0x31, 0x4c, 0x40,
		0x52, 0x8c, 0xae, 0xc9, 0x52, 0x56, 0xc2, 0x8c,
	}

	sm3Output := []byte{
		0xfc, 0x78, 0xe0, 0xa9, 0x65, 0x7a, 0x0c, 0x74,
		0x8c, 0xe5, 0x15, 0x59, 0x90, 0x3c, 0xcf, 0x03,
		0x51, 0x0e, 0x51, 0xd3, 0xcf, 0xf2, 0x32, 0xd7,
		0x13, 0x43, 0xe8, 0x8a, 0x32, 0x1c, 0x53, 0x04,
	}

	key := sm3Output[:32]

	keyList := make([]uint64, 4)
	for i := range keyList {
		keyList[i] = binary.LittleEndian.Uint64(key[i*8 : (i+1)*8])
	}

	encPb := make([]byte, newLen)
	for i := range encPb[:newLen/16] {
		pt := make([]uint64, 2)
		pt[0] = binary.LittleEndian.Uint64(paddedData[i*16 : i*16+8])
		pt[1] = binary.LittleEndian.Uint64(paddedData[i*16+8 : i*16+16])

		ct := SimonEncrypt(pt, keyList, 0)

		binary.LittleEndian.PutUint64(encPb[i*16:i*16+8], ct[0])
		binary.LittleEndian.PutUint64(encPb[i*16+8:i*16+16], ct[1])
	}

	headerBytes := []byte{0xf2, 0xf7, 0xfc, 0xff, 0xf2, 0xf7, 0xfc, 0xff}
	combinedBuffer := headerBytes
	combinedBuffer = append(combinedBuffer, encPb...)
	bBuffer := a.encryptEncPb(combinedBuffer)

	bBuffer = append([]byte{0xa6, 0x6e, 0xad, 0x9f, 0x77, 0x01, 0xd0, 0x0c, 0x18}, bBuffer...)
	bBuffer = append(bBuffer, []byte{'a', 'o'}...)

	hash1 := md5.Sum(signKey[:16])
	hash2 := md5.Sum(signKey[16:])

	block, err := aes.NewCipher(hash1[:])
	if err != nil {
		return "", err
	}

	mode := cipher.NewCBCEncrypter(block, hash2[:])

	paddedBuffer := pkcs7Pad(bBuffer, aes.BlockSize)
	ciphertext := make([]byte, len(paddedBuffer))
	mode.CryptBlocks(ciphertext, paddedBuffer)

	finalResult := append([]byte{0xf2, 0x81}, ciphertext...)

	return base64.StdEncoding.EncodeToString(finalResult), nil
}

func generateRandom(maxValue int) int {
	b := make([]byte, 4)
	_, err := rand.Read(b)
	if err != nil {
		// fallback
		return 12345678
	}
	randomValue := int(binary.BigEndian.Uint32(b)) % maxValue
	if randomValue < 0 {
		randomValue = -randomValue
	}

	return randomValue
}
