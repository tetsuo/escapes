package escapes

// SAUCE record constants.
const (
	sauceRecordSize = 128
	sauceID         = "SAUCE"
	sauceIDLen      = 5
)

// SAUCEInfo holds relevant fields extracted from a SAUCE record.
type SAUCEInfo struct {
	Width  uint16
	Height uint16
}

// ParseSAUCE checks for a SAUCE record at the end of input.
// Returns the info and the slice of input with the SAUCE record (and any
// preceding SUB/EOF byte) stripped.
func ParseSAUCE(input []byte) (SAUCEInfo, []byte) {
	if len(input) < sauceRecordSize {
		return SAUCEInfo{}, input
	}
	rec := input[len(input)-sauceRecordSize:]
	if string(rec[:sauceIDLen]) != sauceID {
		return SAUCEInfo{}, input
	}
	info := SAUCEInfo{
		// TInfo1 is at byte offset 96, TInfo2 at 98, little-endian uint16
		Width:  uint16(rec[96]) | uint16(rec[97])<<8,
		Height: uint16(rec[98]) | uint16(rec[99])<<8,
	}
	trimmed := input[:len(input)-sauceRecordSize]
	// Strip trailing SUB (0x1A) / null EOF markers
	for len(trimmed) > 0 && (trimmed[len(trimmed)-1] == 0x1A || trimmed[len(trimmed)-1] == 0x00) {
		trimmed = trimmed[:len(trimmed)-1]
	}
	return info, trimmed
}
