package mods

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"

	"github.com/sandertv/gophertunnel/minecraft/nbt"
)

func parseNBTSchematic(data []byte, ext string) (*Schematic, error) {
	// Try gzip decompress first
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err == nil {
		decompressed, err2 := io.ReadAll(reader)
		if err2 == nil {
			data = decompressed
		}
		reader.Close()
	}

	switch ext {
	case ".schematic":
		return parseClassicSchematic(data)
	case ".schem":
		return parseSpongeSchematic(data)
	case ".litematic":
		return parseLitematic(data)
	case ".nbt":
		return parseClassicSchematic(data)
	}
	return nil, fmt.Errorf("unsupported format: %s", ext)
}

// Classic .schematic format (MCEdit/WorldEdit)
func parseClassicSchematic(data []byte) (*Schematic, error) {
	var raw map[string]any
	if err := nbt.UnmarshalEncoding(data, &raw, nbt.BigEndian); err != nil {
		return nil, fmt.Errorf("nbt decode: %w", err)
	}

	width := int(toInt(raw["Width"]))
	height := int(toInt(raw["Height"]))
	length := int(toInt(raw["Length"]))

	blocks, ok := raw["Blocks"].([]byte)
	if !ok {
		return nil, fmt.Errorf("missing Blocks field")
	}

	blockData, _ := raw["Data"].([]byte)

	// Block ID to name mapping (simplified)
	sc := &Schematic{}
	for i, blockID := range blocks {
		if blockID == 0 {
			continue // skip air
		}
		y := i / (width * length)
		z := (i % (width * length)) / width
		x := (i % (width * length)) % width

		if y >= height || z >= length || x >= width {
			continue
		}

		var meta byte
		if blockData != nil && i < len(blockData) {
			meta = blockData[i]
		}

		blockName := legacyIDToName(int(blockID), int(meta))
		if blockName == "" || blockName == "air" {
			continue
		}

		sc.Blocks = append(sc.Blocks, SchematicBlock{
			X:     x,
			Y:     y,
			Z:     z,
			Block: blockName,
		})
	}
	return sc, nil
}

// Sponge .schem format
func parseSpongeSchematic(data []byte) (*Schematic, error) {
	var raw map[string]any
	if err := nbt.UnmarshalEncoding(data, &raw, nbt.BigEndian); err != nil {
		return nil, fmt.Errorf("nbt decode: %w", err)
	}

	width := int(toInt(raw["Width"]))
	height := int(toInt(raw["Height"]))
	length := int(toInt(raw["Length"]))

	// Get palette
	paletteRaw, ok := raw["Palette"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing Palette")
	}
	palette := make(map[int]string)
	for name, id := range paletteRaw {
		palette[int(toInt(id))] = name
	}

	blockData, ok := raw["BlockData"].([]byte)
	if !ok {
		return nil, fmt.Errorf("missing BlockData")
	}

	sc := &Schematic{}
	for i, blockID := range blockData {
		name, ok := palette[int(blockID)]
		if !ok || strings.Contains(name, "air") {
			continue
		}
		y := i / (width * length)
		z := (i % (width * length)) / width
		x := (i % (width * length)) % width
		if y >= height || z >= length || x >= width {
			continue
		}
		// Strip blockstate properties
		clean := strings.Split(name, "[")[0]
		clean = strings.TrimPrefix(clean, "minecraft:")
		sc.Blocks = append(sc.Blocks, SchematicBlock{X: x, Y: y, Z: z, Block: clean})
	}
	return sc, nil
}

// Litematic format
func parseLitematic(data []byte) (*Schematic, error) {
	var raw map[string]any
	if err := nbt.UnmarshalEncoding(data, &raw, nbt.BigEndian); err != nil {
		return nil, fmt.Errorf("nbt decode: %w", err)
	}

	regions, ok := raw["Regions"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("missing Regions")
	}

	sc := &Schematic{}
	for _, regionAny := range regions {
		region, ok := regionAny.(map[string]any)
		if !ok {
			continue
		}

		// Get block state palette
		paletteList, ok := region["BlockStatePalette"].([]any)
		if !ok {
			continue
		}
		palette := make([]string, len(paletteList))
		for i, entry := range paletteList {
			m, ok := entry.(map[string]any)
			if !ok {
				continue
			}
			name, _ := m["Name"].(string)
			name = strings.TrimPrefix(name, "minecraft:")
			name = strings.Split(name, "[")[0]
			palette[i] = name
		}

		// Get size
		sizeMap, _ := region["Size"].(map[string]any)
		if sizeMap == nil {
			continue
		}
		sX := int(toInt(sizeMap["x"]))
		sY := int(toInt(sizeMap["y"]))
		sZ := int(toInt(sizeMap["z"]))
		if sX < 0 { sX = -sX }
		if sY < 0 { sY = -sY }
		if sZ < 0 { sZ = -sZ }

		// Block states are packed into int64 array
		blockStates, ok := region["BlockStates"].([]int64)
		if !ok {
			continue
		}

		bits := 4
		for bits < len(palette) {
			bits++
		}
		mask := int64((1 << bits) - 1)
		total := sX * sY * sZ

		for i := 0; i < total; i++ {
			longIdx := (i * bits) / 64
			bitIdx := uint((i * bits) % 64)
			if longIdx >= len(blockStates) {
				break
			}
			var stateID int64
			if bitIdx+uint(bits) <= 64 {
				stateID = (blockStates[longIdx] >> bitIdx) & mask
			} else {
				stateID = blockStates[longIdx] >> bitIdx
				if longIdx+1 < len(blockStates) {
					remaining := uint(bits) - (64 - bitIdx)
					stateID |= (blockStates[longIdx+1] & int64((1<<remaining)-1)) << (64 - bitIdx)
				}
			}

			if int(stateID) >= len(palette) {
				continue
			}
			name := palette[stateID]
			if name == "" || name == "air" {
				continue
			}

			y := i / (sX * sZ)
			z := (i % (sX * sZ)) / sX
			x := (i % (sX * sZ)) % sX

			sc.Blocks = append(sc.Blocks, SchematicBlock{X: x, Y: y, Z: z, Block: name})
		}
	}
	return sc, nil
}

func toInt(v any) int64 {
	switch n := v.(type) {
	case int8:
		return int64(n)
	case int16:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case uint8:
		return int64(n)
	case uint16:
		return int64(n)
	case uint32:
		return int64(n)
	case uint64:
		return int64(n)
	}
	return 0
}

// Legacy Java block ID to Bedrock name (subset of common blocks)
func legacyIDToName(id, meta int) string {
	switch id {
	case 0: return "air"
	case 1: return "stone"
	case 2: return "grass"
	case 3: return "dirt"
	case 4: return "cobblestone"
	case 5: return "planks"
	case 7: return "bedrock"
	case 8: return "water"
	case 10: return "lava"
	case 12: return "sand"
	case 13: return "gravel"
	case 14: return "gold_ore"
	case 15: return "iron_ore"
	case 16: return "coal_ore"
	case 17: return "log"
	case 18: return "leaves"
	case 20: return "glass"
	case 24: return "sandstone"
	case 31: return "tallgrass"
	case 35: return "wool"
	case 41: return "gold_block"
	case 42: return "iron_block"
	case 43: return "double_stone_slab"
	case 44: return "stone_slab"
	case 45: return "brick_block"
	case 46: return "tnt"
	case 47: return "bookshelf"
	case 48: return "mossy_cobblestone"
	case 49: return "obsidian"
	case 53: return "oak_stairs"
	case 54: return "chest"
	case 56: return "diamond_ore"
	case 57: return "diamond_block"
	case 58: return "crafting_table"
	case 60: return "farmland"
	case 61: return "furnace"
	case 67: return "stone_stairs"
	case 73: return "redstone_ore"
	case 79: return "ice"
	case 80: return "snow"
	case 81: return "cactus"
	case 82: return "clay"
	case 87: return "netherrack"
	case 88: return "soul_sand"
	case 89: return "glowstone"
	case 98: return "stonebrick"
	case 102: return "glass_pane"
	case 103: return "melon_block"
	case 112: return "nether_brick"
	case 121: return "end_stone"
	case 125: return "double_wooden_slab"
	case 126: return "wooden_slab"
	case 128: return "sandstone_stairs"
	case 129: return "emerald_ore"
	case 133: return "emerald_block"
	case 134: return "spruce_stairs"
	case 135: return "birch_stairs"
	case 136: return "jungle_stairs"
	case 152: return "redstone_block"
	case 153: return "quartz_ore"
	case 155: return "quartz_block"
	case 156: return "quartz_stairs"
	case 159: return "stained_hardened_clay"
	case 160: return "stained_glass_pane"
	case 161: return "leaves2"
	case 162: return "log2"
	case 163: return "acacia_stairs"
	case 164: return "dark_oak_stairs"
	case 168: return "prismarine"
	case 169: return "sea_lantern"
	case 170: return "hay_block"
	case 172: return "hardened_clay"
	case 173: return "coal_block"
	case 174: return "packed_ice"
	case 179: return "red_sandstone"
	case 180: return "red_sandstone_stairs"
	case 201: return "purpur_block"
	case 203: return "purpur_stairs"
	case 206: return "end_bricks"
	case 208: return "grass_path"
	case 213: return "magma"
	case 214: return "nether_wart_block"
	case 215: return "red_nether_brick"
	case 216: return "bone_block"
	case 220: return "white_glazed_terracotta"
	case 251: return "concrete"
	case 252: return "concrete_powder"
	}
	return fmt.Sprintf("stone") // default unknown blocks to stone
}
