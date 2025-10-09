package utils

import (
	"math"
)

// FloatEquals 比较两个浮点数是否相等（考虑精度误差）
func FloatEquals(a, b, tolerance float64) bool {
	return math.Abs(a-b) <= tolerance
}

// RoundToDecimal 将浮点数四舍五入到指定小数位数
func RoundToDecimal(value float64, places int) float64 {
	multiplier := math.Pow(10, float64(places))
	return math.Round(value*multiplier) / multiplier
}

// Pow 计算 x 的 y 次幂
func Pow(x, y float64) float64 {
	return math.Pow(x, y)
}
