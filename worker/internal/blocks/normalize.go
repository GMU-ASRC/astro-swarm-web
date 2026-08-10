package blocks

import "strings"

func NormalizeToScripts(data any) []Script {
	switch typed := data.(type) {
	case map[string]any:
		if scripts, present := typed["scripts"]; present {
			return rebuildScripts(toSlice(scripts))
		}
		if flat, present := typed["blocks"]; present {
			return splitIntoStacks(NormalizeBlocks(toSlice(flat)))
		}
		return nil
	case []any:
		if len(typed) == 0 {
			return nil
		}
		if first := toMap(typed[0]); first != nil {
			if _, present := first["blocks"]; present {
				return rebuildScripts(typed)
			}
		}
		return splitIntoStacks(NormalizeBlocks(typed))
	}
	return nil
}

func NormalizeBlocks(list []any) []Block {
	if len(list) == 0 {
		return nil
	}
	if first := toMap(list[0]); first != nil {
		if _, present := first["children"]; present {
			return rebuildNested(list)
		}
	}

	result := make([]Block, 0, len(list))
	currentIndex := -1
	for _, item := range list {
		entry := toMap(item)
		if entry == nil {
			continue
		}
		blockType := stringOf(entry["type"])
		node := Block{Type: blockType, Params: copyParams(entry["params"])}
		switch {
		case strings.HasPrefix(blockType, "when_"):
			result = append(result, node)
			currentIndex = len(result) - 1
		case strings.HasPrefix(blockType, "do_") && currentIndex >= 0:
			result[currentIndex].Children = append(result[currentIndex].Children, node)
		default:
			currentIndex = -1
			result = append(result, node)
		}
	}
	return result
}

func rebuildNested(list []any) []Block {
	result := make([]Block, 0, len(list))
	for _, item := range list {
		entry := toMap(item)
		if entry == nil {
			continue
		}
		result = append(result, Block{
			Type:     stringOf(entry["type"]),
			Params:   copyParams(entry["params"]),
			Children: rebuildNested(toSlice(entry["children"])),
		})
	}
	return result
}

func rebuildScripts(list []any) []Script {
	result := make([]Script, 0, len(list))
	fallbackY := 40.0
	for _, item := range list {
		entry := toMap(item)
		if entry == nil {
			continue
		}
		result = append(result, Script{
			X:      floatOf(entry["x"], 40.0),
			Y:      floatOf(entry["y"], fallbackY),
			Blocks: rebuildNested(toSlice(entry["blocks"])),
		})
		fallbackY += 150.0
	}
	return result
}

func splitIntoStacks(nested []Block) []Script {
	var configs []Block
	var heads []Block
	for _, node := range nested {
		if strings.HasPrefix(node.Type, "set_") {
			configs = append(configs, node)
		} else {
			heads = append(heads, node)
		}
	}

	var ordered []Block
	if len(configs) > 0 {
		ordered = append(ordered, Block{
			Type:     "when_start",
			Params:   map[string]any{},
			Children: configs,
		})
	}
	ordered = append(ordered, heads...)

	result := make([]Script, 0, len(ordered))
	for index, node := range ordered {
		column := index % 2
		row := index / 2
		result = append(result, Script{
			X:      40.0 + float64(column)*400.0,
			Y:      30.0 + float64(row)*280.0,
			Blocks: []Block{node},
		})
	}
	return result
}
