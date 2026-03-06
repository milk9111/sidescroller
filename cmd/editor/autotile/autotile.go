package autotile

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

const (
	MaskNorth uint8 = 1 << iota
	MaskNorthEast
	MaskEast
	MaskSouthEast
	MaskSouth
	MaskSouthWest
	MaskWest
	MaskNorthWest
)

var (
	defaultMaskOrderOnce sync.Once
	defaultMaskOrder     []uint8
	defaultMaskOffsets   map[uint8]int
)

var auto47MaskOrder = []uint8{
	28, 124, 112, 16, 247, 223, 125, 31, 255, 241, 17, 253, 127, 95, 7, 199, 193, 1, 117, 87, 245, 4, 68, 64, 0, 213, 93, 215, 23, 209, 116, 92, 20, 84, 80, 29, 113, 197, 71, 21, 85, 81, 221, 119, 5, 69, 65,
}

func Canonicalize(mask uint8) uint8 {
	if mask&MaskNorth == 0 || mask&MaskWest == 0 {
		mask &^= MaskNorthWest
	}
	if mask&MaskNorth == 0 || mask&MaskEast == 0 {
		mask &^= MaskNorthEast
	}
	if mask&MaskSouth == 0 || mask&MaskEast == 0 {
		mask &^= MaskSouthEast
	}
	if mask&MaskSouth == 0 || mask&MaskWest == 0 {
		mask &^= MaskSouthWest
	}
	return mask
}

func BuildMask(north, east, south, west, northWest, northEast, southEast, southWest bool) uint8 {
	var mask uint8
	if north {
		mask |= MaskNorth
	}
	if east {
		mask |= MaskEast
	}
	if south {
		mask |= MaskSouth
	}
	if west {
		mask |= MaskWest
	}
	if northEast {
		mask |= MaskNorthEast
	}
	if southEast {
		mask |= MaskSouthEast
	}
	if southWest {
		mask |= MaskSouthWest
	}
	if northWest {
		mask |= MaskNorthWest
	}
	return Canonicalize(mask)
}

func DefaultMaskOrder() []uint8 {
	ensureDefaultOrder()
	return append([]uint8(nil), defaultMaskOrder...)
}

func DefaultOffset(mask uint8) (int, bool) {
	ensureDefaultOrder()
	offset, ok := defaultMaskOffsets[Canonicalize(mask)]
	return offset, ok
}

func ResolveOffset(mask uint8, remap []int) (int, bool) {
	offset, ok := DefaultOffset(mask)
	if !ok {
		return 0, false
	}
	if len(remap) != len(DefaultMaskOrder()) {
		return offset, true
	}
	if remap[offset] < 0 {
		return offset, true
	}
	return remap[offset], true
}

func LoadRemap(path string) ([]int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read autotile remap %q: %w", path, err)
	}
	var remap []int
	if err := json.Unmarshal(data, &remap); err != nil {
		return nil, fmt.Errorf("decode autotile remap %q: %w", path, err)
	}
	if len(remap) != len(DefaultMaskOrder()) {
		return nil, fmt.Errorf("autotile remap %q must contain %d entries, got %d", path, len(DefaultMaskOrder()), len(remap))
	}
	return remap, nil
}

func ensureDefaultOrder() {
	defaultMaskOrderOnce.Do(func() {
		defaultMaskOrder = append([]uint8(nil), auto47MaskOrder...)
		defaultMaskOffsets = make(map[uint8]int, len(defaultMaskOrder))
		for index, mask := range defaultMaskOrder {
			defaultMaskOffsets[Canonicalize(mask)] = index
		}
	})
}
