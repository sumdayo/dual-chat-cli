package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
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
	// 環境変数
	openaiKey := os.Getenv("OPENAI_API_KEY")
	anthropicKey := os.Getenv("ANTHROPIC_API_KEY")

	if openaiKey == "" || anthropicKey == "" {
		fmt.Println("Error: 環境変数が設定されていません。")
		return
	}

	dirPath := flag.String("d", "", "読み込むコンテキストフォルダのパス (例: -d ./docs)")
	flag.Parse()

	// プロンプトの取得
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: go run . [-d folder_path] '質問内容'")
		return
	}
	userQuery := args[0]

	var contextText string
	if *dirPath != "" {
		fmt.Printf("フォルダ '%s' を読み込んでいます...\n", *dirPath)
		var err error
		contextText, err = readFilesInDir(*dirPath)
		if err != nil {
			fmt.Printf("Error reading directory: %v\n", err)
			return
		}
		fmt.Printf("読み込み完了 (%d 文字)\n", len(contextText))
	}

	// AIに送る最終的なプロンプトを作成
	finalPrompt := userQuery
	if contextText != "" {
		finalPrompt = fmt.Sprintf("以下の【参考資料】を前提知識として、ユーザーの質問に答えてください。\n\n【参考資料】\n%s\n\n【ユーザーの質問】\n%s", contextText, userQuery)
	}

	// AIの呼び出し
	var wg sync.WaitGroup
	results := make(chan ModelResult, 2)
	wg.Add(2)

	// GPT-4o
	go func() {
		defer wg.Done()
		start := time.Now()
		content, err := callOpenAI(openaiKey, finalPrompt)
		results <- ModelResult{Name: "GPT-4o", Content: content, Error: err, Duration: time.Since(start)}
	}()

	// Claude3.5
	go func() {
		defer wg.Done()
		start := time.Now()
		content, err := callClaude(anthropicKey, finalPrompt)
		results <- ModelResult{Name: "Claude 3.5/4.5", Content: content, Error: err, Duration: time.Since(start)}
	}()

	wg.Wait()
	close(results)

	// 結果表示
	for res := range results {
		fmt.Println("\n==========================================")
		fmt.Printf("🤖 %s (Time: %v)\n", res.Name, res.Duration)
		fmt.Println("------------------------------------------")

		if res.Error != nil {
			fmt.Printf("❌ Error: %v\n", res.Error)
		} else if res.Content == "" {
			fmt.Println("⚠️ Warning: 回答が空です")
		} else {
			fmt.Println(res.Content)
		}
		fmt.Println("==========================================")
	}
}

// 指定されたフォルダ内のテキストファイルを再帰的に読み込む
func readFilesInDir(dir string) (string, error) {
	var sb strings.Builder

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if strings.HasPrefix(info.Name(), ".") && info.Name() != "." {
				return filepath.SkipDir
			}
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		if ext == ".txt" || ext == ".md" || ext == ".go" || ext == ".json" || ext == ".py" {
			content, err := ioutil.ReadFile(path)
			if err != nil {
				return nil // 読み込みエラーは無視して進む
			}
			sb.WriteString(fmt.Sprintf("\n--- File: %s ---\n", path))
			sb.WriteString(string(content))
			sb.WriteString("\n")
		}
		return nil
	})

	return sb.String(), err
}