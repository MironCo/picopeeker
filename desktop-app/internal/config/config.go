package config

// PicoModel represents the selected Pico model
type PicoModel int

const (
	Pico1 PicoModel = iota
	Pico2
)

// MemoryRegions holds the memory map for a Pico model
type MemoryRegions struct {
	SRAMSize     string // Human-readable size
	FlashSize    string // Human-readable size
	SRAMSizeHex  uint32 // Size in bytes
	FlashSizeHex uint32 // Size in bytes
}

// GetMemoryRegions returns the memory configuration for the selected Pico model
func GetMemoryRegions(model PicoModel) MemoryRegions {
	switch model {
	case Pico1:
		return MemoryRegions{
			SRAMSize:     "264KB",
			FlashSize:    "2MB",
			SRAMSizeHex:  0x42000,  // 264KB
			FlashSizeHex: 0x200000, // 2MB
		}
	case Pico2:
		return MemoryRegions{
			SRAMSize:     "520KB",
			FlashSize:    "4MB",
			SRAMSizeHex:  0x82000,  // 520KB
			FlashSizeHex: 0x400000, // 4MB
		}
	default:
		return GetMemoryRegions(Pico2) // Default to Pico 2
	}
}

// GetModelFromString converts a string to PicoModel
func GetModelFromString(model string) PicoModel {
	switch model {
	case "Pico 1 (RP2040)":
		return Pico1
	case "Pico 2 (RP2350)":
		return Pico2
	default:
		return Pico2
	}
}

// GetModelString returns the display string for a model
func GetModelString(model PicoModel) string {
	switch model {
	case Pico1:
		return "Pico 1 (RP2040)"
	case Pico2:
		return "Pico 2 (RP2350)"
	default:
		return "Pico 2 (RP2350)"
	}
}
