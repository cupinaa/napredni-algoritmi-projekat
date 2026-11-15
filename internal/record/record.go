package record

import (
	"encoding/binary"
	"errors"
)

type RecordType uint8

const (
	TypeRegular        RecordType = 0
	TypeMergeOperation RecordType = 1
	TypeBlobReference  RecordType = 2
)

type Record struct {
	Key       string
	Value     []byte
	Timestamp int64
	Tombstone bool
	Type      RecordType
	ExpiresAt int64
}

const (
	timestampSize   = 8
	tombstoneSize   = 1
	typeSize        = 1
	expiresAtSize   = 8
	keyLengthSize   = 8
	valueLenghtSize = 8

	HeaderSize = timestampSize + tombstoneSize + typeSize + expiresAtSize + keyLengthSize + valueLenghtSize
)

var ErrTooShort = errors.New("record : not enough bytes to decode")

func (r Record) Encode() []byte {
	keyLength := len(r.Key)
	valueLength := len(r.Value)
	bytes := make([]byte, HeaderSize+keyLength+valueLength)

	position := 0

	binary.LittleEndian.PutUint64(bytes[position:], uint64(r.Timestamp))
	position += timestampSize

	if r.Tombstone {
		bytes[position] = 1
	}
	position += tombstoneSize

	bytes[position] = byte(r.Type)
	position += typeSize

	binary.LittleEndian.PutUint64(bytes[position:], uint64(r.ExpiresAt))
	position += expiresAtSize

	binary.LittleEndian.PutUint64(bytes[position:], uint64(keyLength))
	position += keyLengthSize

	binary.LittleEndian.PutUint64(bytes[position:], uint64(valueLength))
	position += valueLenghtSize

	copy(bytes[position:], r.Key)
	position += keyLength

	copy(bytes[position:], r.Value)

	return bytes
}

func Decode(bytes []byte) (Record, error) {
	if len(bytes) < HeaderSize {
		return Record{}, ErrTooShort
	}

	var r Record
	position := 0

	r.Timestamp = int64(binary.LittleEndian.Uint64(bytes[position:]))
	position += timestampSize

	r.Tombstone = bytes[position] == 1
	position += tombstoneSize

	r.Type = RecordType(bytes[position])
	position += typeSize

	r.ExpiresAt = int64(binary.LittleEndian.Uint64(bytes[position:]))
	position += expiresAtSize

	keyLength := binary.LittleEndian.Uint64(bytes[position:])
	position += keyLengthSize

	valueLenght := binary.LittleEndian.Uint64(bytes[position:])
	position += valueLenghtSize

	if uint64(len(bytes)-position) < keyLength+valueLenght {
		return Record{}, ErrTooShort
	}

	keyEnd := position + int(keyLength)
	r.Key = string(bytes[position:keyEnd])

	valueEnd := keyEnd + int(valueLenght)
	r.Value = append([]byte(nil), bytes[keyEnd:valueEnd]...)

	return r, nil
}

func (r Record) Size() int {
	return HeaderSize + len(r.Key) + len(r.Value)
}
