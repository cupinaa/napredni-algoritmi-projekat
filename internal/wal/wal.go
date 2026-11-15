package wal

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cupinaa/napredni-algoritmi-projekat/internal/record"
)

type WAL interface {
	Append(records ...record.Record) (segmentIndex uint64, err error)
	// Read visits records from oldest to newest and returns an error on a CRC mismatch.
	Read(visit func(r record.Record) error) error
	// DeleteSegmentsUpTo deletes segments <= lowWaterMark, but never the last one.
	// Returns an error if lowWaterMark is greater than the last segment.
	DeleteSegmentsUpTo(lowWaterMark uint64) error
}

type SegmentedWAL struct {
	directory string
}

func (wal *SegmentedWAL) DeleteSegmentsUpTo(lowWaterMark uint64) error {
	entries, err := os.ReadDir(wal.directory)
	if err != nil {
		return fmt.Errorf("wal: reading directory: %w", err)
	}

	fileNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		fileNames = append(fileNames, entry.Name())
	}

	numbers, err := sortedSegmentNumbers(fileNames)
	if err != nil {
		return err
	}
	if len(numbers) == 0 {
		return nil
	}

	lastSegment := numbers[len(numbers)-1]
	if lowWaterMark > lastSegment {
		return fmt.Errorf("wal: low-water mark %d is greater than the last segment %d", lowWaterMark, lastSegment)
	}

	for _, number := range numbers {
		if number > lowWaterMark || number == lastSegment {
			break
		}
		path := filepath.Join(wal.directory, segmentFileName(number))
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("wal: deleting segment %s: %w", segmentFileName(number), err)
		}
	}
	return nil
}
