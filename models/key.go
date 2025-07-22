package models

type DecryptionKey struct {
	Key           []byte `json:"key"`            // encoded key for AES decryption
	IV            []byte `json:"iv"`             // initialization vector for AES decryption
	Method        string `json:"method"`         // e.g., "AES-128-CBC"
	MediaSequence int    `json:"media_sequence"` // sequence number for HLS segments
}
