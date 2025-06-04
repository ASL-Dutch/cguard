package utils

import (
	"fmt"
	"strconv"

	"github.com/xuri/excelize/v2"
)

// GetCellStringValue 获取Excel单元格的字符串值，自动处理公式和普通单元格
func GetCellStringValue(f *excelize.File, sheet string, col string, row int) (string, error) {
	cell := col + strconv.Itoa(row)
	// 先判断是否为公式
	formula, err := f.GetCellFormula(sheet, cell)
	if err == nil && formula != "" {
		// 是公式，计算其值
		val, err := f.CalcCellValue(sheet, cell)
		fmt.Println("calc cell value:", val)
		return val, err
	}
	// 否则直接取值
	val, err := f.GetCellValue(sheet, cell)
	return val, err
}

// GetCellFloatValue 获取Excel单元格的float64值，自动处理公式和普通单元格
func GetCellFloatValue(f *excelize.File, sheet string, col string, row int) (float64, error) {
	valStr, err := GetCellStringValue(f, sheet, col, row)
	if err != nil {
		return 0, err
	}
	if valStr == "" {
		return 0, nil
	}
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, err
	}
	return val, nil
}

// GetCellIntValue 获取Excel单元格的int值，自动处理公式和普通单元格
func GetCellIntValue(f *excelize.File, sheet string, col string, row int) (int, error) {
	valStr, err := GetCellStringValue(f, sheet, col, row)
	if err != nil {
		return 0, err
	}
	if valStr == "" {
		return 0, nil
	}
	val, err := strconv.Atoi(valStr)
	if err != nil {
		return 0, err
	}
	return val, nil
}

// SetCellValue 设置Excel单元格的值，支持多种数据类型
func SetCellValue(f *excelize.File, sheet string, col string, row int, value interface{}) error {
	cell := col + strconv.Itoa(row)
	return f.SetCellValue(sheet, cell, value)
}

// IsCellValueNumber 判断Excel单元格的值是否为数字
func IsCellValueNumber(f *excelize.File, sheet string, col string, row int) bool {
	val, err := GetCellStringValue(f, sheet, col, row)
	if err != nil {
		return false
	}

	_, err = strconv.ParseFloat(val, 64)
	return err == nil
}
