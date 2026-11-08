package benchmarks

import (
	"errors"
	"fmt"
)

var ErrLegacyMissing = errors.New("legacy replay file is missing")

type BinaryResult struct {
	Key   string
	Bytes int64
}

func (files *Files) ConvertToBinary(framesKey string, framesFormat string) (BinaryResult, error) {
	legacy, found := files.Read(framesKey)
	if !found {
		return BinaryResult{}, ErrLegacyMissing
	}
	original, err := DecodeFrames(legacy, framesFormat)
	if err != nil {
		return BinaryResult{}, fmt.Errorf("could not decode %s: %w", framesKey, err)
	}
	encoded, err := EncodeBinaryFrames(original)
	if err != nil {
		return BinaryResult{}, err
	}
	decoded, err := DecodeBinaryFrames(encoded)
	if err != nil {
		return BinaryResult{}, fmt.Errorf("could not read back %s: %w", framesKey, err)
	}
	if err := FramesMatch(original, decoded); err != nil {
		return BinaryResult{}, fmt.Errorf("%s did not convert exactly: %w", framesKey, err)
	}
	binaryKey := BinaryKeyFor(framesKey)
	if err := files.WriteRelative(binaryKey, encoded); err != nil {
		return BinaryResult{}, err
	}
	return BinaryResult{Key: binaryKey, Bytes: int64(len(encoded))}, nil
}

func (files *Files) VerifyBinary(framesKey string, framesFormat string, binaryKey string) error {
	legacy, found := files.Read(framesKey)
	if !found {
		return ErrLegacyMissing
	}
	original, err := DecodeFrames(legacy, framesFormat)
	if err != nil {
		return err
	}
	stored, found := files.Read(binaryKey)
	if !found {
		return fmt.Errorf("%s is missing", binaryKey)
	}
	decoded, err := DecodeBinaryFrames(stored)
	if err != nil {
		return fmt.Errorf("%s: %w", binaryKey, err)
	}
	return FramesMatch(original, decoded)
}
