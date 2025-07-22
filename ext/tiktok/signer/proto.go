package signer

import (
	"encoding/binary"
	"fmt"
)

type ProtoError struct {
	Msg string
}

func (e ProtoError) Error() string {
	return e.Msg
}

type ProtoFieldType int

const (
	ProtoFieldTypeVarint     ProtoFieldType = 0
	ProtoFieldTypeInt64      ProtoFieldType = 1
	ProtoFieldTypeString     ProtoFieldType = 2
	ProtoFieldTypeGroupStart ProtoFieldType = 3
	ProtoFieldTypeGroupEnd   ProtoFieldType = 4
	ProtoFieldTypeInt32      ProtoFieldType = 5
	ProtoFieldTypeError1     ProtoFieldType = 6
	ProtoFieldTypeError2     ProtoFieldType = 7
)

func (t ProtoFieldType) String() string {
	names := map[ProtoFieldType]string{
		ProtoFieldTypeVarint:     "VARINT",
		ProtoFieldTypeInt64:      "INT64",
		ProtoFieldTypeString:     "STRING",
		ProtoFieldTypeGroupStart: "GROUPSTART",
		ProtoFieldTypeGroupEnd:   "GROUPEND",
		ProtoFieldTypeInt32:      "INT32",
		ProtoFieldTypeError1:     "ERROR1",
		ProtoFieldTypeError2:     "ERROR2",
	}

	if name, ok := names[t]; ok {
		return name
	}
	return fmt.Sprintf("UNKNOWN(%d)", t)
}

type ProtoField struct {
	Idx  int
	Type ProtoFieldType
	Val  any
}

func (f ProtoField) IsAsciiStr() bool {
	val, ok := f.Val.([]byte)
	if !ok {
		return false
	}
	for _, b := range val {
		if b < 0x20 || b > 0x7e {
			return false
		}
	}
	return true
}

func (f ProtoField) String() string {
	switch f.Type {
	case ProtoFieldTypeInt32, ProtoFieldTypeInt64, ProtoFieldTypeVarint:
		return fmt.Sprintf("%d(%s): %d", f.Idx, f.Type, f.Val)
	case ProtoFieldTypeString:
		val, ok := f.Val.([]byte)
		if ok && f.IsAsciiStr() {
			return fmt.Sprintf("%d(%s): \"%s\"", f.Idx, f.Type, string(val))
		}
		return fmt.Sprintf("%d(%s): h\"%x\"", f.Idx, f.Type, f.Val)
	case ProtoFieldTypeGroupStart, ProtoFieldTypeGroupEnd:
		return fmt.Sprintf("%d(%s): %v", f.Idx, f.Type, f.Val)
	case ProtoFieldTypeError1, ProtoFieldTypeError2:
		return fmt.Sprintf("%d(%s): %v", f.Idx, f.Type, f.Val)
	default:
		return fmt.Sprintf("%d(%s): %v", f.Idx, f.Type, f.Val)
	}
}

type ProtoBuf struct {
	Fields []ProtoField
}

func NewProtoBuf(data any) (*ProtoBuf, error) {
	pb := &ProtoBuf{
		Fields: []ProtoField{},
	}

	if data == nil {
		return pb, nil
	}

	switch d := data.(type) {
	case []byte:
		if len(d) > 0 {
			if err := pb.parseBuf(d); err != nil {
				return nil, err
			}
		}
	case map[string]any, map[int]any:
		if err := pb.parseDict(d); err != nil {
			return nil, err
		}
	default:
		return nil, ProtoError{Msg: fmt.Sprintf("unsupported type %T to protobuf", data)}
	}

	return pb, nil
}

type protoReader struct {
	data []byte
	pos  int
}

func (r *protoReader) isRemain(length int) bool {
	return r.pos+length <= len(r.data)
}

func (r *protoReader) read0() byte {
	if !r.isRemain(1) {
		panic("buffer overrun")
	}
	b := r.data[r.pos]
	r.pos++
	return b
}

func (r *protoReader) read(length int) []byte {
	if !r.isRemain(length) {
		panic("buffer overrun")
	}
	ret := r.data[r.pos : r.pos+length]
	r.pos += length
	return ret
}

func (r *protoReader) readInt32() uint32 {
	return binary.LittleEndian.Uint32(r.read(4))
}

func (r *protoReader) readInt64() uint64 {
	return binary.LittleEndian.Uint64(r.read(8))
}

func (r *protoReader) readVarint() uint64 {
	var vint uint64
	var n uint

	for {
		b := uint64(r.read0())
		vint |= (b & 0x7F) << (7 * n)
		if b < 0x80 {
			break
		}
		n++
	}

	return vint
}

func (r *protoReader) readString() []byte {
	length := r.readVarint()
	return r.read(int(length))
}

type protoWriter struct {
	data []byte
}

func (w *protoWriter) write0(b byte) {
	w.data = append(w.data, b)
}

func (w *protoWriter) write(bs []byte) {
	w.data = append(w.data, bs...)
}

func (w *protoWriter) writeInt32(i uint32) {
	bs := make([]byte, 4)
	binary.LittleEndian.PutUint32(bs, i)
	w.write(bs)
}

func (w *protoWriter) writeInt64(i uint64) {
	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, i)
	w.write(bs)
}

func (w *protoWriter) writeVarint(i uint64) {
	for i > 0x7F {
		w.write0(byte((i & 0x7F) | 0x80))
		i >>= 7
	}
	w.write0(byte(i & 0x7F))
}

func (w *protoWriter) writeString(bs []byte) {
	w.writeVarint(uint64(len(bs)))
	w.write(bs)
}

func (pb *ProtoBuf) parseBuf(data []byte) error {
	r := &protoReader{data: data, pos: 0}

	for r.isRemain(1) {
		var key uint64

		func() {
			defer func() {
				if r := recover(); r != nil {
					key = 0
				}
			}()
			key = r.readVarint()
		}()

		if key == 0 {
			break
		}

		fieldType := ProtoFieldType(key & 0x7)
		fieldIdx := int(key >> 3)

		if fieldIdx == 0 {
			break
		}

		var field ProtoField

		switch fieldType {
		case ProtoFieldTypeInt32:
			field = ProtoField{Idx: fieldIdx, Type: fieldType, Val: r.readInt32()}
		case ProtoFieldTypeInt64:
			field = ProtoField{Idx: fieldIdx, Type: fieldType, Val: r.readInt64()}
		case ProtoFieldTypeVarint:
			field = ProtoField{Idx: fieldIdx, Type: fieldType, Val: r.readVarint()}
		case ProtoFieldTypeString:
			field = ProtoField{Idx: fieldIdx, Type: fieldType, Val: r.readString()}
		case ProtoFieldTypeGroupStart, ProtoFieldTypeGroupEnd, ProtoFieldTypeError1, ProtoFieldTypeError2:
			// not implemented, just store nil or empty
			field = ProtoField{Idx: fieldIdx, Type: fieldType, Val: nil}
		default:
			return ProtoError{Msg: fmt.Sprintf("parse protobuf error, unexpected field type: %s", fieldType)}
		}

		pb.Fields = append(pb.Fields, field)
	}

	return nil
}

func (pb *ProtoBuf) ToBuf() ([]byte, error) {
	w := &protoWriter{}

	for _, field := range pb.Fields {
		key := uint64((field.Idx << 3) | int(field.Type&7))
		w.writeVarint(key)

		switch field.Type {
		case ProtoFieldTypeInt32:
			if val, ok := field.Val.(uint32); ok {
				w.writeInt32(val)
			} else if val, ok := field.Val.(int); ok {
				w.writeInt32(uint32(val))
			} else {
				return nil, ProtoError{Msg: fmt.Sprintf("invalid value type %T for INT32 field", field.Val)}
			}
		case ProtoFieldTypeInt64:
			if val, ok := field.Val.(uint64); ok {
				w.writeInt64(val)
			} else if val, ok := field.Val.(int); ok {
				w.writeInt64(uint64(val))
			} else {
				return nil, ProtoError{Msg: fmt.Sprintf("invalid value type %T for INT64 field", field.Val)}
			}
		case ProtoFieldTypeVarint:
			if val, ok := field.Val.(uint64); ok {
				w.writeVarint(val)
			} else if val, ok := field.Val.(int); ok {
				w.writeVarint(uint64(val))
			} else {
				return nil, ProtoError{Msg: fmt.Sprintf("invalid value type %T for VARINT field", field.Val)}
			}
		case ProtoFieldTypeString:
			if val, ok := field.Val.([]byte); ok {
				w.writeString(val)
			} else {
				return nil, ProtoError{Msg: fmt.Sprintf("invalid value type %T for STRING field", field.Val)}
			}
		case ProtoFieldTypeGroupStart, ProtoFieldTypeGroupEnd, ProtoFieldTypeError1, ProtoFieldTypeError2:
			// not implemented, skip writing value
		default:
			return nil, ProtoError{Msg: fmt.Sprintf("encode protobuf error, unexpected field type: %s", field.Type)}
		}
	}

	return w.data, nil
}

func (pb *ProtoBuf) Get(idx int) *ProtoField {
	for i := range pb.Fields {
		if pb.Fields[i].Idx == idx {
			return &pb.Fields[i]
		}
	}
	return nil
}

func (pb *ProtoBuf) GetInt(idx int) (int, error) {
	field := pb.Get(idx)
	if field == nil {
		return 0, nil
	}

	switch field.Type {
	case ProtoFieldTypeInt32, ProtoFieldTypeInt64, ProtoFieldTypeVarint:
		switch v := field.Val.(type) {
		case uint32:
			return int(v), nil
		case uint64:
			return int(v), nil
		case int:
			return v, nil
		}
	case ProtoFieldTypeString, ProtoFieldTypeGroupStart, ProtoFieldTypeGroupEnd, ProtoFieldTypeError1, ProtoFieldTypeError2:
		// not supported for GetInt
		return 0, ProtoError{Msg: fmt.Sprintf("GetInt(%d) -> %s (unsupported type)", idx, field.Type)}
	}

	return 0, ProtoError{Msg: fmt.Sprintf("GetInt(%d) -> %s", idx, field.Type)}
}

func (pb *ProtoBuf) GetBytes(idx int) ([]byte, error) {
	field := pb.Get(idx)
	if field == nil {
		return nil, nil
	}

	if field.Type == ProtoFieldTypeString {
		if val, ok := field.Val.([]byte); ok {
			return val, nil
		}
	}

	return nil, ProtoError{Msg: fmt.Sprintf("GetBytes(%d) -> %s", idx, field.Type)}
}

func (pb *ProtoBuf) GetUTF8(idx int) (string, error) {
	bytes, err := pb.GetBytes(idx)
	if err != nil || bytes == nil {
		return "", err
	}
	return string(bytes), nil
}

func (pb *ProtoBuf) GetProtoBuf(idx int) (*ProtoBuf, error) {
	bytes, err := pb.GetBytes(idx)
	if err != nil || bytes == nil {
		return nil, err
	}

	return NewProtoBuf(bytes)
}

func (pb *ProtoBuf) Put(field ProtoField) {
	pb.Fields = append(pb.Fields, field)
}

func (pb *ProtoBuf) PutInt32(idx int, val int) {
	pb.Put(ProtoField{Idx: idx, Type: ProtoFieldTypeInt32, Val: uint32(val)})
}

func (pb *ProtoBuf) PutInt64(idx int, val int) {
	pb.Put(ProtoField{Idx: idx, Type: ProtoFieldTypeInt64, Val: uint64(val)})
}

func (pb *ProtoBuf) PutVarint(idx int, val int) {
	pb.Put(ProtoField{Idx: idx, Type: ProtoFieldTypeVarint, Val: uint64(val)})
}

func (pb *ProtoBuf) PutBytes(idx int, val []byte) {
	pb.Put(ProtoField{Idx: idx, Type: ProtoFieldTypeString, Val: val})
}

func (pb *ProtoBuf) PutUTF8(idx int, val string) {
	pb.Put(ProtoField{Idx: idx, Type: ProtoFieldTypeString, Val: []byte(val)})
}

func (pb *ProtoBuf) PutProtoBuf(idx int, val *ProtoBuf) error {
	buf, err := val.ToBuf()
	if err != nil {
		return err
	}

	pb.Put(ProtoField{Idx: idx, Type: ProtoFieldTypeString, Val: buf})
	return nil
}

func (pb *ProtoBuf) parseDict(data any) error {
	switch d := data.(type) {
	case map[string]any:
		for key, val := range d {
			var idx int
			fmt.Sscanf(key, "%d", &idx)
			if err := pb.addDictValue(idx, val); err != nil {
				return err
			}
		}
	case map[int]any:
		for idx, val := range d {
			if err := pb.addDictValue(idx, val); err != nil {
				return err
			}
		}
	default:
		return ProtoError{Msg: fmt.Sprintf("unsupported dict type %T", data)}
	}

	return nil
}

func (pb *ProtoBuf) addDictValue(idx int, val any) error {
	switch v := val.(type) {
	case int:
		pb.PutVarint(idx, v)
	case string:
		pb.PutUTF8(idx, v)
	case []byte:
		pb.PutBytes(idx, v)
	case map[string]any, map[int]any:
		nestedPb, err := NewProtoBuf(v)
		if err != nil {
			return err
		}
		if err := pb.PutProtoBuf(idx, nestedPb); err != nil {
			return err
		}
	default:
		return ProtoError{Msg: fmt.Sprintf("unsupported value type %T for protobuf", val)}
	}

	return nil
}

func (pb *ProtoBuf) ToDict(out map[string]any) (map[string]any, error) {
	for key, val := range out {
		var idx int
		fmt.Sscanf(key, "%d", &idx)

		switch v := val.(type) {
		case int:
			intVal, err := pb.GetInt(idx)
			if err != nil {
				return nil, err
			}
			out[key] = intVal
		case string:
			strVal, err := pb.GetUTF8(idx)
			if err != nil {
				return nil, err
			}
			out[key] = strVal
		case []byte:
			bytesVal, err := pb.GetBytes(idx)
			if err != nil {
				return nil, err
			}
			out[key] = bytesVal
		case map[string]any:
			nestedPb, err := pb.GetProtoBuf(idx)
			if err != nil || nestedPb == nil {
				return nil, err
			}
			if _, err := nestedPb.ToDict(v); err != nil {
				return nil, err
			}
		default:
			return nil, ProtoError{Msg: fmt.Sprintf("unsupported value type %T for protobuf dict", val)}
		}
	}

	return out, nil
}
