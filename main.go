package main

import (
	"bufio"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	gomail "gopkg.in/gomail.v2"
)

func getStatus(urls []string) <-chan string {
	// チャネルを生成
	statusChan := make(chan string)
	for _, url := range urls {
		go func(url string) {
			res, err := http.Get(url)
			if err != nil {
				log.Fatal(err)
			}
			defer res.Body.Close()
			statusChan <- res.Status + " => " + url // レスポンスステータスとURLをチャネルに入れる
		}(url)
	}
	return statusChan // チャネルを返す
}

func fromFile(filePath string) []string {
	// ファイルを開く
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "File %s could not read: %v\n", filePath, err)
		os.Exit(1)
	}

	defer f.Close()

	// 読み込む
	lines := make([]string, 0, 100) // capacityを100に指定してメモリを確保
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		// 追加
		lines = append(lines, scanner.Text())
	}
	if serr := scanner.Err(); serr != nil {
		fmt.Fprintf(os.Stderr, "File %s scan error: %v\n", filePath, err)
	}

	return lines
}

func sendMail(from, to, cc, subject, body string) {
	m := gomail.NewMessage()
	m.SetHeader("From", from)
	m.SetHeader("To", to)
	m.SetHeader("Cc", cc)
	m.SetHeader("Subject", subject)
	m.SetBody("text/plain", body)
	//m.Attach("test.jpg")

	d := gomail.NewDialer("HOSTNAME", 25, "My_MALL", "PASSWORD") // HostName, Port, MailAddress, Password
	if err := d.DialAndSend(m); err != nil {
		panic(err)
	}
}

func main() {
	// Mail情報
	const from = "My_MALL"
	const to = "TO"
	const cc = "Cc"
	subject := "Unconnected Url(s)"
	body := "Error:\n"

	// タイムアウトは1秒に設定
	seconds := 1
	timeout := time.After(time.Duration(seconds) * time.Second)

	urls := fromFile("urls.txt") // テキストファイルに記載したurlを配列に入れる

	// URLバリデーションをする
	errorUrls := []string{} // invalidなurlをためこむ配列
	for _, site := range urls {
		_, err := url.ParseRequestURI(site)
		// URLがinvalidだった場合
		if err != nil {
			errorUrls = append(errorUrls, site) // errorUrls配列にurlをためこむ
		}
	}

	lURL := len(errorUrls) // errorUrls配列の要素数

	// invalidが1つでもあった場合
	if lURL >= 1 {
		fmt.Println("Invalid URL:")
		// invalidだったURLの配列リストを端末に出力
		for _, u := range errorUrls {
			fmt.Println(u)
		}
		os.Exit(0) // 処理を終了する
	}

	statusChan := getStatus(urls) // 並列処理にわたす
	unconnectedUrls := []string{} // つながらなかったURLが入る配列

LOOP:
	for {
		select {
		case status := <-statusChan:
			fmt.Println(status) // 処理したURLのステータスを端末に出力
			// statusに200 OKがなければ、-1が入る
			statusFlag := strings.Index(status, "200 OK")
			if statusFlag == -1 {
				unconnectedUrls = append(unconnectedUrls, status) // 配列にプッシュ
			}

		case <-timeout:
			// 設定したタイムアウト時間後にfor/selectを抜ける
			break LOOP
		}
	}

	lStatus := len(unconnectedUrls) // unconnectedUrls配列の要素数

	// つながらないURLが1つでもあった場合
	if lStatus >= 1 {
		// つながらなかったURLの配列リストをbody変数に改行付きで追加していく
		for _, s := range unconnectedUrls {
			body += s + "\n"
		}
		// エラーメール送信
		sendMail(from, to, cc, subject, body)
	}

	fmt.Println("Done!")
}
