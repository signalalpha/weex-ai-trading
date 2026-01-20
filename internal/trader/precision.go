package trader

import (
	"fmt"
	"math"
	"strings"
)

// SymbolPrecision 交易对精度配置
type SymbolPrecision struct {
	PriceStep float64 // 价格步长
	SizeStep  float64 // 数量步长
	MinSize   float64 // 最小数量
}

// 交易对的精度配置（根据实际API测试结果整理）
var symbolPrecisionMap = map[string]SymbolPrecision{
	"cmt_btcusdt":  {PriceStep: 0.1, SizeStep: 0.001, MinSize: 0.001},
	"cmt_ethusdt":  {PriceStep: 0.01, SizeStep: 0.001, MinSize: 0.001},
	"cmt_solusdt":  {PriceStep: 0.01, SizeStep: 0.1, MinSize: 0.1},
	"cmt_dogeusdt": {PriceStep: 0.00001, SizeStep: 100, MinSize: 100},
	"cmt_xrpusdt":  {PriceStep: 0.0001, SizeStep: 10, MinSize: 10},
	"cmt_adausdt":  {PriceStep: 0.0001, SizeStep: 10, MinSize: 10},
	"cmt_bnbusdt":  {PriceStep: 0.01, SizeStep: 0.1, MinSize: 0.1},
	"cmt_ltcusdt":  {PriceStep: 0.01, SizeStep: 0.1, MinSize: 0.1},
}

// RoundToStep 将值四舍五入到指定步长
func RoundToStep(value, step float64) float64 {
	if step <= 0 {
		return value
	}
	return math.Round(value/step) * step
}

// AdjustPriceToPrecision 根据交易对的精度调整价格
func AdjustPriceToPrecision(price float64, symbol string) float64 {
	precision, ok := symbolPrecisionMap[symbol]
	if !ok {
		// 默认使用 0.01 步长
		return RoundToStep(price, 0.01)
	}
	return RoundToStep(price, precision.PriceStep)
}

// AdjustSizeToPrecision 根据交易对的精度调整数量，并确保不小于最小值
func AdjustSizeToPrecision(size float64, symbol string) float64 {
	precision, ok := symbolPrecisionMap[symbol]
	if !ok {
		// 默认使用 0.001 步长和最小值
		adjusted := RoundToStep(size, 0.001)
		if adjusted < 0.001 {
			adjusted = 0.001
		}
		return adjusted
	}

	// 先调整到步长
	adjusted := RoundToStep(size, precision.SizeStep)

	// 确保不小于最小值
	if adjusted < precision.MinSize {
		adjusted = precision.MinSize
	}

	return adjusted
}

// FormatPriceString 根据交易对的精度格式化价格字符串
func FormatPriceString(price float64, symbol string) string {
	precision, ok := symbolPrecisionMap[symbol]
	if !ok {
		// 默认保留2位小数
		return formatFloat(price, 0.01)
	}

	return formatFloat(price, precision.PriceStep)
}

// formatFloat 根据步长格式化浮点数为字符串
func formatFloat(value, step float64) string {
	var decimals int

	if step >= 1 {
		// 步长 >= 1，使用整数
		decimals = 0
	} else if step >= 0.1 {
		// 步长 >= 0.1，保留1位小数
		decimals = 1
	} else if step >= 0.01 {
		// 步长 >= 0.01，保留2位小数
		decimals = 2
	} else if step >= 0.001 {
		// 步长 >= 0.001，保留3位小数
		decimals = 3
	} else if step >= 0.0001 {
		// 步长 >= 0.0001，保留4位小数
		decimals = 4
	} else if step >= 0.00001 {
		// 步长 >= 0.00001，保留5位小数
		decimals = 5
	} else {
		// 步长更小，计算需要的小数位数
		decimals = countDecimals(step)
	}

	format := fmt.Sprintf("%%.%df", decimals)
	str := fmt.Sprintf(format, value)

	// 去掉末尾的0（但保留必要的小数点）
	return strings.TrimRight(strings.TrimRight(str, "0"), ".")
}

// countDecimals 计算步长需要的小数位数
func countDecimals(step float64) int {
	if step >= 1 {
		return 0
	}
	count := 0
	multiplier := 1.0
	for step*multiplier < 1 {
		count++
		multiplier *= 10
		if count > 10 {
			break
		}
	}
	return count
}
