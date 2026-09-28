package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"remotehq-simulator/internal/simulate"
)

func printResult(result simulate.Result, monthlyPoints int, individualPaymentEnabled bool) {
	headers := []string{"月", "開始日", "残レンタル可能pt", "付与pt充当", "貯まったpt消化", "自己負担(円・税抜)", "貯まったpt残高"}
	rows := make([][]string, 1, len(result.Months)+1)
	rows[0] = headers
	for _, month := range result.Months {
		rows = append(rows, []string{
			strconv.Itoa(month.Number),
			month.Date.Format(time.DateOnly),
			strconv.Itoa(month.Available),
			strconv.Itoa(month.FromCapacity),
			strconv.Itoa(month.FromCarried),
			strconv.Itoa(month.IndividualPayment),
			strconv.Itoa(month.CarriedBalance),
		})
	}
	columnWidths := make([]int, len(headers))
	for _, row := range rows {
		for i, field := range row {
			if width := displayWidth(field); width > columnWidths[i] {
				columnWidths[i] = width
			}
		}
	}

	for rowIndex, row := range rows {
		for i, field := range row {
			if i > 0 {
				fmt.Print(" | ")
			}
			padding := columnWidths[i] - displayWidth(field)
			fmt.Print(strings.Repeat(" ", padding), field)
		}
		fmt.Println()
		if rowIndex == 0 {
			for i, width := range columnWidths {
				if i > 0 {
					fmt.Print("-+-")
				}
				fmt.Print(strings.Repeat("-", width))
			}
			fmt.Println()
		}
	}

	fmt.Printf("ポイントのみで注文可能な上限 (注文時点): %d pt/月\n", result.PointsOnlyLimit)
	if monthlyPoints <= result.PointsOnlyLimit {
		fmt.Println("注文時判定: ポイントのみで注文可能")
	} else {
		fmt.Println("注文時判定: ポイントのみでは注文不可 (自己負担の併用が必要)")
	}
	fmt.Printf("合計: 付与pt充当 %d pt / 貯まったpt消化 %d pt / 自己負担 %d 円 (税抜、別途消費税)\n", result.TotalFromCapacity, result.TotalFromCarried, result.TotalIndividualPayment)
	if len(result.Months) > 0 {
		fmt.Printf("終了時の貯まったポイント: %d pt\n", result.Months[len(result.Months)-1].CarriedBalance)
	}
	if result.TotalIndividualPayment > 0 && !individualPaymentEnabled {
		fmt.Println("警告: この組織では自己負担が無効のため、この注文はそのままではできません。")
	}
}

func displayWidth(s string) int {
	width := 0
	for _, r := range s {
		if r > 127 {
			width += 2
		} else {
			width++
		}
	}
	return width
}
