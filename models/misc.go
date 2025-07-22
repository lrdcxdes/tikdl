package models

type SendMediaFormatsOptions struct {
	IsStored bool
	Caption  string
}

type Chunk struct {
	Data []byte
	Idx  int
}
