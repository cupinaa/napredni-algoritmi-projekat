package wal

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

const (
	segmentPrefix    = "wal_"
	segmentExtension = ".log"
)

func segmentFileName(number uint64) string {
	return fmt.Sprintf("%s%04d%s", segmentPrefix, number, segmentExtension)
}

func parseSegmentNumber(fileName string) (uint64, bool) {
	digits := strings.TrimPrefix(fileName, segmentPrefix)
	digits = strings.TrimSuffix(digits, segmentExtension)

	number, err := strconv.ParseUint(digits, 10, 64)
	if err != nil {
		return 0, false
	}
	if number == 0 {
		return 0, false
	}
	if segmentFileName(number) != fileName {
		return 0, false
	}
	return number, true
}

func sortedSegmentNumbers(fileNames []string) ([]uint64, error) {
	var numbers []uint64
	for _, fileName := range fileNames {
		number, ok := parseSegmentNumber(fileName)
		if ok {
			numbers = append(numbers, number)
		}
	}
	slices.Sort(numbers)

	for index := 1; index < len(numbers); index++ {
		if numbers[index] != numbers[index-1]+1 {
			return nil, fmt.Errorf("wal: segment %s is missing", segmentFileName(numbers[index-1]+1))
		}
	}
	return numbers, nil
}
