package mcp

import (
	"regexp"
	"sort"
	"strconv"
)

type ToolHelp struct {
	Method    string
	ParamsNum int
}

func ParseToolHelps(input string) ([]ToolHelp, error) {
	var results []ToolHelp

	methodBlockRe := regexp.MustCompile(`(?s)new\s+web3\._extend\.Method\s*\(\s*\{(.*?)\}\s*\)`)
	callRe := regexp.MustCompile(`call\s*:\s*['\"]([^'\"]+)['\"]`)
	paramsRe := regexp.MustCompile(`params\s*:\s*(\d+)`)

	propBlockRe := regexp.MustCompile(`(?s)new\s+web3\._extend\.Property\s*\(\s*\{(.*?)\}\s*\)`)
	getterRe := regexp.MustCompile(`getter\s*:\s*['\"]([^'\"]+)['\"]`)

	methodBlocks := methodBlockRe.FindAllStringSubmatch(input, -1)
	for _, mb := range methodBlocks {
		block := mb[1]
		callMatch := callRe.FindStringSubmatch(block)
		if len(callMatch) == 2 {
			paramsNum := 0
			if p := paramsRe.FindStringSubmatch(block); len(p) == 2 {
				if n, err := strconv.Atoi(p[1]); err == nil {
					paramsNum = n
				}
			}
			results = append(results, ToolHelp{Method: callMatch[1], ParamsNum: paramsNum})
		}
	}

	propBlocks := propBlockRe.FindAllStringSubmatch(input, -1)
	for _, pb := range propBlocks {
		block := pb[1]
		getterMatch := getterRe.FindStringSubmatch(block)
		if len(getterMatch) == 2 {
			results = append(results, ToolHelp{Method: getterMatch[1], ParamsNum: 0})
		}
	}

	return results, nil
}

func ValuesByNumericKeyOrder(arguments map[string]interface{}) []interface{} {
	if len(arguments) <= 0 {
		return nil
	}

	type indexedValue struct {
		index int
		value interface{}
	}

	digitRe := regexp.MustCompile(`\d+`)

	arr := make([]indexedValue, 0, len(arguments))
	for k, v := range arguments {
		matches := digitRe.FindAllString(k, -1)
		if len(matches) == 0 {
			continue
		}
		last := matches[len(matches)-1]
		idx, err := strconv.Atoi(last)
		if err != nil {
			continue
		}
		arr = append(arr, indexedValue{index: idx, value: v})
	}

	sort.Slice(arr, func(i, j int) bool { return arr[i].index < arr[j].index })

	res := make([]interface{}, 0, len(arr))
	for _, iv := range arr {
		res = append(res, iv.value)
	}
	return res
}

type ToolResult struct {
	raw []byte
}

func (t *ToolResult) UnmarshalJSON(data []byte) error {
	t.raw = data
	return nil
}
