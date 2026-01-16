package main

import (
	"fmt"
	"os"
	"sync"
	"time"
)

// それぞれのモデルの結果を格納
type ModelResult struct {
	Name     string
	Content  string
	Error    error
	Duration time.Duration
}

func main() {
	// 環境変数からAPIキーを取得
	openaiKey := os.Getenv("OPENAI_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")

	if openaiKey == "" || anthropicKey == "" {
		fmt.Println("Error: 環境変数が設定されていません。")
		return
	}

	// 引数のチェック
	if len(os.Args) < 2 {
		fmt.Println("go run . 'ここにプロンプトを入力'")
		return
	}
	prompt := os.Args[1]

    var wg sync.WaitGroup
	results := make(chan ModelResult, 2)
	wg.Add(2)

	// GPT-4oへのリクエスト
	go func() {
		defer wg.Done()
		start := time.Now()
		content, err := callOpenAI(openaiKey, prompt)
		results <- ModelResult{Name: "GPT-4o", Content: content, Error: err, Duration: time.Since(start)}
	}()

	// Claude3.5へのリクエスト
	go func() {
		defer wg.Done()
		start := time.Now()
		content, err := callClaude(anthropicKey, prompt)
		results <- ModelResult{Name: "Claude 3.5 Sonnet", Content: content, Error: err, Duration: time.Since(start)}
	}()

	wg.Wait()
	close(results)

	// 結果の表示
	for res := range results {
		fmt.Println("\n==========================================")
		fmt.Printf("🤖 %s (Time: %v)\n", res.Name, res.Duration)
		fmt.Println("------------------------------------------")

		if res.Error != nil {
			// エラーがある場合は赤文字っぽく表示して理由を知る
			fmt.Printf("❌ Error: %v\n", res.Error)
		} else if res.Content == "" {
			fmt.Println("⚠️ Warning: 回答が空です")
		} else {
			// ★ここに一番重要な「中身を表示する命令」を追加しました
			fmt.Println(res.Content)
		}
		fmt.Println("==========================================")
	}

}