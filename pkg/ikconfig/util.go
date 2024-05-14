package ikconfig

import (
	"fmt"
	"os"
)

type MagicNotFound interface {
	Error() string
}

type MagicNotFoundErr struct {
	msg   string
	magic []byte
}

func (e *MagicNotFoundErr) Error() string {
	return fmt.Sprintf("magic string %v not found", e.magic)
}

// Find a magic string in the provided file
// Return the offset of the magic string, err if not found or unknown
func SearchBytes(file string, b []byte) (uint, error) {
	dat, err := os.ReadFile(file)
	if err != nil {
		return 0, fmt.Errorf("error reading file: %w", err)
	}

	for i := 0; i < len(dat); i++ {
		if dat[i] == b[0] {
			found := true
			for j := 1; j < len(b); j++ {
				if dat[i+j] != b[j] {
					found = false
					break
				}
			}
			if found {
				return uint(i), nil
			}
		}
	}
	return 0, &MagicNotFoundErr{msg: "magic string not found", magic: b}
}
