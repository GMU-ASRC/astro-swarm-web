package validate

import (
	"fmt"
)

const (
	maxScriptsPerProgram = 64
	maxBlocksPerEntry    = 4000
	maxBlockDepth        = 16
	maxParamsPerBlock    = 12
	maxSimulatorName     = 40
	maxTextParamLength   = 60
)

type blockBudget struct {
	remaining int
}

func cleanScripts(value any, budget *blockBudget) ([]any, error) {
	scripts, isList := value.([]any)
	if !isList || len(scripts) > maxScriptsPerProgram {
		return nil, fmt.Errorf("a program must be a list of up to %d scripts", maxScriptsPerProgram)
	}
	cleaned := []any{}
	for _, item := range scripts {
		script, isObject := item.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("each script must be an object")
		}
		x, err := roundedNumber(objectValue(script, "x", 0.0), "script x", 1)
		if err != nil {
			return nil, err
		}
		y, err := roundedNumber(objectValue(script, "y", 0.0), "script y", 1)
		if err != nil {
			return nil, err
		}
		blocks, err := cleanBlocks(objectValue(script, "blocks", []any{}), budget, 0)
		if err != nil {
			return nil, err
		}
		cleaned = append(cleaned, map[string]any{"x": x, "y": y, "blocks": blocks})
	}
	return cleaned, nil
}

func cleanBlocks(value any, budget *blockBudget, depth int) ([]any, error) {
	if depth > maxBlockDepth {
		return nil, fmt.Errorf("blocks are nested more than %d levels deep", maxBlockDepth)
	}
	blocks, isList := value.([]any)
	if !isList {
		return nil, fmt.Errorf("blocks must be a list")
	}
	cleaned := []any{}
	for _, item := range blocks {
		block, isObject := item.(map[string]any)
		if !isObject {
			return nil, fmt.Errorf("each block must be an object")
		}
		budget.remaining--
		if budget.remaining < 0 {
			return nil, fmt.Errorf("an entry may hold at most %d blocks", maxBlocksPerEntry)
		}
		blockType, err := simulatorText(block["type"], "block type", maxSimulatorName)
		if err != nil {
			return nil, err
		}
		params, err := cleanParams(objectValue(block, "params", map[string]any{}))
		if err != nil {
			return nil, err
		}
		children, err := cleanBlocks(objectValue(block, "children", []any{}), budget, depth+1)
		if err != nil {
			return nil, err
		}
		cleaned = append(cleaned, map[string]any{"type": blockType, "params": params, "children": children})
	}
	return cleaned, nil
}

func cleanParams(value any) (map[string]any, error) {
	params, isObject := value.(map[string]any)
	if !isObject || len(params) > maxParamsPerBlock {
		return nil, fmt.Errorf("block params must be an object of up to %d values", maxParamsPerBlock)
	}
	cleaned := map[string]any{}
	for key, param := range params {
		name, err := simulatorText(key, "param name", maxSimulatorName)
		if err != nil {
			return nil, err
		}
		switch typed := param.(type) {
		case bool:
			cleaned[name] = typed
		case nil, string, []any, map[string]any:
			text, err := simulatorText(param, name, maxTextParamLength)
			if err != nil {
				return nil, err
			}
			cleaned[name] = text
		default:
			number, err := simulatorNumber(param, name)
			if err != nil {
				return nil, err
			}
			cleaned[name] = number
		}
	}
	return cleaned, nil
}
