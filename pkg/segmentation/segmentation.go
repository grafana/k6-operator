package segmentation

import (
	"errors"
	"fmt"
	"strings"
)

const (
	beginning = "0"
	end       = "1"
)

// NewCommandFragments builds command fragments for starting k6 with execution segments.
func NewCommandFragments(index int, total int) ([]string, error) {

	if index > total {
		return nil, errors.New("node index exceeds configured parallelism")
	}

	parts := []string{beginning}

	for i := 1; i < total; i++ {
		parts = append(parts, fmt.Sprintf("%d/%d", i, total))
	}

	parts = append(parts, end)

	getSegmentPart := func(index int, total int) string {
		if index == 0 {
			return "0"
		}
		if index == total {
			return "1"
		}
		return fmt.Sprintf("%d/%d", index, total)
	}

	segment := fmt.Sprintf("%s:%s", getSegmentPart(index-1, total), getSegmentPart(index, total))
	sequence := strings.Join(parts[:], ",")

	return []string{
		fmt.Sprintf("--execution-segment=%s", segment),
		fmt.Sprintf("--execution-segment-sequence=%s", sequence),
	}, nil
}
