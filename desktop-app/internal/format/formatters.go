package format

import (
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"unsafe"
)

// StringToHex converts string to hex
func StringToHex(s string) string {
	result := ""
	for i := 0; i < len(s); i++ {
		result += fmt.Sprintf("%02X", s[i])
	}
	return result
}

// Int32ToHexLE converts int32 to hex (little-endian)
func Int32ToHexLE(val int32) string {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(val))
	return fmt.Sprintf("%02X%02X%02X%02X", buf[0], buf[1], buf[2], buf[3])
}

// Parse hex dump output and extract raw bytes
func ParseHexDump(dump string) []byte {
	var result []byte
	lines := strings.Split(dump, "\n")

	for _, line := range lines {
		// Look for lines with hex dump format: "00000000: 01 02 03 04 ..."
		if !strings.Contains(line, ":") {
			continue
		}

		// Split on colon
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}

		// Extract hex bytes (before the ASCII section)
		hexPart := parts[1]
		// Remove ASCII part (after double space)
		if idx := strings.Index(hexPart, "  "); idx != -1 {
			hexPart = hexPart[:idx]
		}

		// Parse hex bytes
		hexBytes := strings.Fields(hexPart)
		for _, hexByte := range hexBytes {
			if len(hexByte) == 2 {
				val, err := strconv.ParseUint(hexByte, 16, 8)
				if err == nil {
					result = append(result, byte(val))
				}
			}
		}
	}

	return result
}

// FormatMemoryDump reformats the hex dump based on the selected display mode
func FormatMemoryDump(rawDump string, displayMode string) string {
	// If not in bytes mode, we need to parse and reformat
	if displayMode == "Bytes (Hex)" {
		return rawDump // Return original format
	}

	// Parse the hex dump to extract bytes
	bytes := ParseHexDump(rawDump)
	if len(bytes) == 0 {
		return rawDump // Return original if parsing failed
	}

	// Extract starting address from the dump
	startAddr := ExtractStartAddress(rawDump)

	// Format based on display mode
	switch displayMode {
	case "16-bit Words":
		return FormatAs16BitWords(bytes, startAddr)
	case "32-bit Words":
		return FormatAs32BitWords(bytes, startAddr)
	case "Float (32-bit)":
		return FormatAsFloats(bytes, startAddr)
	default:
		return rawDump
	}
}

// extractStartAddress extracts the starting address from the hex dump
func ExtractStartAddress(dump string) uint32 {
	lines := strings.Split(dump, "\n")
	for _, line := range lines {
		// Look for "Address: 0x12345678"
		if strings.Contains(line, "Address:") {
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "Address:" && i+1 < len(fields) {
					addr := strings.TrimSuffix(fields[i+1], ",")
					var val uint64
					fmt.Sscanf(addr, "0x%x", &val)
					return uint32(val)
				}
			}
		}
	}
	return 0
}

// formatAs16BitWords formats bytes as 16-bit words (little-endian)
func FormatAs16BitWords(bytes []byte, startAddr uint32) string {
	var sb strings.Builder
	sb.WriteString("=== 16-bit Word View (Little-Endian) ===\n\n")
	sb.WriteString("Address:  Hex Bytes  Value      Decimal\n")
	sb.WriteString("--------  ---------  ------     --------\n")

	for i := 0; i+1 < len(bytes); i += 2 {
		addr := startAddr + uint32(i)
		val := binary.LittleEndian.Uint16(bytes[i : i+2])
		sb.WriteString(fmt.Sprintf("%08x: %02x %02x     0x%04x     %d\n",
			addr, bytes[i], bytes[i+1], val, int16(val)))
	}

	// Handle odd byte at end if present
	if len(bytes)%2 == 1 {
		i := len(bytes) - 1
		addr := startAddr + uint32(i)
		sb.WriteString(fmt.Sprintf("%08x: %02x        0x%02x       %d (partial)\n",
			addr, bytes[i], bytes[i], int8(bytes[i])))
	}

	sb.WriteString("\n===END===\n")
	return sb.String()
}

// formatAs32BitWords formats bytes as 32-bit words (little-endian)
func FormatAs32BitWords(bytes []byte, startAddr uint32) string {
	var sb strings.Builder
	sb.WriteString("=== 32-bit Word View (Little-Endian) ===\n\n")
	sb.WriteString("Address:  Hex Bytes        Value       Decimal (signed)  Decimal (unsigned)\n")
	sb.WriteString("--------  ---------------  ----------  ----------------  ------------------\n")

	for i := 0; i+3 < len(bytes); i += 4 {
		addr := startAddr + uint32(i)
		val := binary.LittleEndian.Uint32(bytes[i : i+4])
		sb.WriteString(fmt.Sprintf("%08x: %02x %02x %02x %02x  0x%08x  %-16d  %d\n",
			addr, bytes[i], bytes[i+1], bytes[i+2], bytes[i+3],
			val, int32(val), val))
	}

	// Handle remaining bytes if not aligned to 4
	remainder := len(bytes) % 4
	if remainder > 0 {
		i := len(bytes) - remainder
		addr := startAddr + uint32(i)
		sb.WriteString(fmt.Sprintf("%08x: ", addr))
		for j := 0; j < remainder; j++ {
			sb.WriteString(fmt.Sprintf("%02x ", bytes[i+j]))
		}
		sb.WriteString("(partial word)\n")
	}

	sb.WriteString("\n===END===\n")
	return sb.String()
}

// formatAsFloats formats bytes as 32-bit floats (little-endian)
func FormatAsFloats(bytes []byte, startAddr uint32) string {
	var sb strings.Builder
	sb.WriteString("=== Float View (32-bit, Little-Endian) ===\n\n")
	sb.WriteString("Address:  Hex Bytes        Float Value\n")
	sb.WriteString("--------  ---------------  -----------\n")

	for i := 0; i+3 < len(bytes); i += 4 {
		addr := startAddr + uint32(i)
		bits := binary.LittleEndian.Uint32(bytes[i : i+4])
		floatVal := *(*float32)(unsafe.Pointer(&bits))
		sb.WriteString(fmt.Sprintf("%08x: %02x %02x %02x %02x  %f\n",
			addr, bytes[i], bytes[i+1], bytes[i+2], bytes[i+3], floatVal))
	}

	// Handle remaining bytes if not aligned to 4
	remainder := len(bytes) % 4
	if remainder > 0 {
		i := len(bytes) - remainder
		addr := startAddr + uint32(i)
		sb.WriteString(fmt.Sprintf("%08x: ", addr))
		for j := 0; j < remainder; j++ {
			sb.WriteString(fmt.Sprintf("%02x ", bytes[i+j]))
		}
		sb.WriteString("(partial float)\n")
	}

	sb.WriteString("\n===END===\n")
	return sb.String()
}
