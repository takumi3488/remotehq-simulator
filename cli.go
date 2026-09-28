package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"remotehq-simulator/internal/paste"
	"remotehq-simulator/internal/rentals"
	"remotehq-simulator/internal/simulate"
)

func run() error {
	reader := bufio.NewReader(os.Stdin)
	cacheDir, err := cacheDirectory()
	if err != nil {
		return err
	}
	cachePath := filepath.Join(cacheDir, "rentals.html")

	snapshot, err := loadSnapshot(reader, cachePath)
	if err != nil {
		return err
	}
	printSnapshotSummary(snapshot)

	monthlyPoints, err := promptPositiveInt(reader, "レンタル月額 (pt): ")
	if err != nil {
		return err
	}
	months, err := promptPositiveInt(reader, "ポイント消化期間 (月): ")
	if err != nil {
		return err
	}

	localNow := time.Now()
	start := time.Date(localNow.Year(), localNow.Month(), localNow.Day(), 0, 0, 0, 0, time.UTC)
	result, err := simulate.Run(snapshot, monthlyPoints, months, start)
	if err != nil {
		return err
	}
	printResult(result, monthlyPoints, snapshot.IndividualPaymentEnabled)
	return nil
}

func cacheDirectory() (string, error) {
	if xdg := os.Getenv("XDG_CACHE_HOME"); xdg != "" && filepath.IsAbs(xdg) {
		return filepath.Join(xdg, "remotehq-simulator"), nil
	}
	if runtime.GOOS == "windows" {
		base, err := os.UserCacheDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(base, "remotehq-simulator"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".cache", "remotehq-simulator"), nil
}

func loadSnapshot(reader *bufio.Reader, cachePath string) (*rentals.Snapshot, error) {
	if info, err := os.Stat(cachePath); err == nil {
		useCache, err := askToUseCache(reader, cachePath, info.ModTime())
		if err != nil {
			return nil, err
		}
		if useCache {
			html, err := os.ReadFile(cachePath)
			if err != nil {
				return nil, err
			}
			snapshot, err := rentals.Parse(html)
			if err == nil {
				return snapshot, nil
			}
			fmt.Printf("キャッシュを解析できませんでした (%v)。新しいレスポンスを貼り付けてください。\n", err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	for {
		fmt.Print("ブラウザで https://app.hq-hq.co.jp/remote/mypage/rentals を開いてください。\n開発者ツールの「ネットワーク」タブで rentals のリクエストを選択してください (無ければページを再読み込み)。\n「レスポンス」をコピーしてこのシェルにペーストしてください (ペースト完了で自動的に読み込み。終わらない場合は Ctrl-D、中止は Ctrl-C)。\n")
		html, err := paste.ReadTerminal(os.Stdin, reader, os.Stdout)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("入力が終了しました: %w", err)
			}
			return nil, err
		}
		fmt.Printf("貼り付けを受信しました (%d bytes)。\n", len(html))
		snapshot, err := rentals.Parse(html)
		if err != nil {
			fmt.Printf("レスポンスを解析できませんでした (%v)。もう一度貼り付けてください。\n", err)
			continue
		}
		if err := saveCache(cachePath, html); err != nil {
			return nil, err
		}
		fmt.Printf("キャッシュに保存しました: %s\n", cachePath)
		return snapshot, nil
	}
}

func askToUseCache(reader *bufio.Reader, path string, modified time.Time) (bool, error) {
	prompt := fmt.Sprintf("キャッシュ %s (更新: %s) を使用しますか? [Y/n]: ", path, modified.Local().Format("2006-01-02 15:04"))
	for {
		line, err := readLine(reader, prompt)
		if err != nil {
			return false, err
		}
		switch strings.ToLower(strings.TrimSpace(line)) {
		case "", "y", "yes":
			return true, nil
		case "n", "no":
			return false, nil
		default:
			fmt.Println("y または n を入力してください。")
		}
	}
}

func promptPositiveInt(reader *bufio.Reader, prompt string) (int, error) {
	for {
		line, err := readLine(reader, prompt)
		if err != nil {
			return 0, err
		}
		value, err := strconv.Atoi(strings.TrimSpace(line))
		if err == nil && value > 0 {
			return value, nil
		}
		fmt.Println("正の整数を入力してください。")
	}
}

func readLine(reader *bufio.Reader, prompt string) (string, error) {
	fmt.Print(prompt)
	line, err := reader.ReadString('\n')
	if errors.Is(err, io.EOF) {
		return "", fmt.Errorf("入力が終了しました: %w", err)
	}
	return line, err
}

func saveCache(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func printSnapshotSummary(snapshot *rentals.Snapshot) {
	usingCost := 0
	for _, rental := range snapshot.Rentals {
		if rental.State == rentals.StateUsing {
			usingCost += rental.MonthlyCost
		}
	}
	fmt.Printf("残レンタル可能ポイント: %d pt/月\n貯まったポイント: %d pt\n利用中レンタル合計: %d pt/月\n", snapshot.ConsumableCapacity, snapshot.CarriedPoints, usingCost)
	if !snapshot.IndividualPaymentEnabled {
		fmt.Println("この組織では自己負担が無効です。")
	}
}
