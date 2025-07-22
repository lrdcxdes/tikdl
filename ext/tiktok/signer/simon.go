package signer

func getBit(val uint64, pos int) uint64 {
	if val&(1<<pos) != 0 {
		return 1
	}
	return 0
}

func rotateLeft64(v uint64, n int) uint64 {
	n %= 64
	return ((v << n) | (v >> (64 - n))) & 0xFFFFFFFFFFFFFFFF
}

func rotateRight64(v uint64, n int) uint64 {
	n %= 64
	return ((v >> n) | (v << (64 - n))) & 0xFFFFFFFFFFFFFFFF
}

func keyExpansion(key []uint64) []uint64 {
	expandedKey := make([]uint64, 72)
	copy(expandedKey[:4], key)

	var tmp uint64
	for i := 4; i < 72; i++ {
		tmp = rotateRight64(expandedKey[i-1], 3)
		tmp ^= expandedKey[i-3]
		tmp ^= rotateRight64(tmp, 1)
		expandedKey[i] = ^expandedKey[i-4] & 0xFFFFFFFFFFFFFFFF
		expandedKey[i] ^= tmp
		expandedKey[i] ^= getBit(0x3DC94C3A046D678B, (i-4)%62)
		expandedKey[i] ^= 3
	}

	return expandedKey
}

func SimonDecrypt(ct []uint64, k []uint64, c int) []uint64 {
	key := keyExpansion(k)

	xi := ct[0]
	xi1 := ct[1]

	var tmp, f uint64

	for i := range key {
		j := 71 - i
		tmp = xi
		if c == 1 {
			f = rotateLeft64(xi, 1)
		} else {
			f = rotateLeft64(xi, 1) & rotateLeft64(xi, 8)
		}
		xi = xi1 ^ f ^ rotateLeft64(xi, 2) ^ key[j]
		xi1 = tmp
	}

	return []uint64{xi, xi1}
}

func SimonEncrypt(pt []uint64, k []uint64, c int) []uint64 {
	key := keyExpansion(k)

	xi := pt[0]
	xi1 := pt[1]

	var tmp, f uint64

	for i := range 72 {
		tmp = xi1
		if c == 1 {
			f = rotateLeft64(xi1, 1)
		} else {
			f = rotateLeft64(xi1, 1) & rotateLeft64(xi1, 8)
		}
		xi1 = xi ^ f ^ rotateLeft64(xi1, 2) ^ key[i]
		xi = tmp
	}

	return []uint64{xi, xi1}
}
