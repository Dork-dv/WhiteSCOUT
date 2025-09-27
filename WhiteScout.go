package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

type Crawler struct {
	BaseURL     *url.URL
	Visited     map[string]bool
	Mutex       sync.Mutex
	Results     []string
	ThreadLimit chan struct{}
	DelayMs     int
	MaxDepth    int
	FilterMode  int // 1 = interessante, 2 = padrão, 3 = tudo
}

var totalRequests int
var requestsMutex sync.Mutex

var endpointRegex = regexp.MustCompile(`["'](\/[^\s"'<>]+|https?://[^\s"'<>]+)["']`)
var ignoreExt = []string{".css", ".js", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".ttf", ".pdf", ".zip", ".rar"}
var interestingKeywords = []string{"admin", "login", "api", "dashboard", "wp-", "auth", "user", "account", "manage", "upload", "signin", "signup"}

func main() {
	rand.Seed(time.Now().UnixNano())
	clearScreen()
	banner()

	target := readInput("Digite a URL alvo: ")

	threadsInput := readInput("Digite o limite de threads (padrão 10): ")
	delayInput := readInput("Digite o delay entre requisições em ms (padrão 200): ")

	fmt.Println("\nNíveis disponíveis:")
	fmt.Println("1) Rasa — endpoints interessantes — profundidade curta")
	fmt.Println("2) Padrão — ignora arquivos estáticos")
	fmt.Println("3) Profunda — coleta TODOS os endpoints")
	levelInput := readInput("Escolha o nível (1/2/3) (padrão 2): ")

	threads, delay, level := 10, 200, 2
	if threadsInput != "" {
		fmt.Sscanf(threadsInput, "%d", &threads)
	}
	if delayInput != "" {
		fmt.Sscanf(delayInput, "%d", &delay)
	}
	if levelInput != "" {
		fmt.Sscanf(levelInput, "%d", &level)
		if level < 1 || level > 3 {
			level = 2
		}
	}

	var maxDepth int
	switch level {
	case 1:
		maxDepth = 1
	case 2:
		maxDepth = 2
	case 3:
		maxDepth = 3
	default:
		maxDepth = 2
	}

	crawler, err := NewCrawler(target, threads, delay, maxDepth, level)
	if err != nil {
		fmt.Println("[ERRO] URL inválida:", err)
		return
	}

	start := time.Now()
	go showLiveProgress(start)
	results := crawler.Start()

	displayResults(results)
	fmt.Printf("\nTotal de endpoints coletados: %d\n", len(results))
	fmt.Printf("Tempo: %.2fs\n", time.Since(start).Seconds())
	askToExport(results)
}

// ---------------- Utils ----------------
func readInput(prompt string) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Print(prompt)
	text, _ := reader.ReadString('\n')
	return strings.TrimSpace(text)
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func banner() {
	fmt.Println(`
⣿⣿⣿⣿⣯⠉⠄⠄⠄⠄⠄⠄⡄⠄⠄⠄⠄⠄⠄⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⡟⠁⠄⠄⠄⠄⠄⢀⢀⠃⠄⠄⠄⠄⠄⠄⠘⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⡇⠄⠄⣾⣳⠄⠄⢀⣄⣦⣶⣴⠂⢒⠄⠄⠄⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⣿⡄⠄⠈⠚⡆⠄⢸⣿⣿⣿⣯⠋⡏⠄⠄⢸⣿⣿⣿⠿⠛⠛⠿⣿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⣿⠟⣂⣀⣀⣀⡀⠠⠻⣷⣎⡼⠞⠓⠦⣤⣛⣋⣭⣴⣾⣿⣿⣷⣌⠻⣿⣿⣿⣿⣿
⣿⣿⣿⠋⣼⣿⣿⣿⣿⣿⣷⣦⣍⣙⠻⠳⠄⠄⠈⠙⠿⢿⣿⣿⣿⣿⣿⡟⣰⣿⣿⣿⣿⣿
⣿⣿⡟⢸⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣶⣄⣀⠄⠄⢀⣤⣤⣭⡛⠛⣩⣴⣿⣿⣿⣿⣿⣿
⣿⣿⣷⠸⠿⠛⠉⠙⠛⠿⢿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠟⠛⠷⠦⣹⣿⣿⣿⣿⣿
⣿⣿⣿⣧⠄⠄⠄⢀⣴⣷⣶⣦⣬⣭⣉⣙⣛⠛⠿⠿⠿⠟⠁⡀⠄⠄⠄⢁⣿⣿⣿⣿⣿⣿
⣿⣿⣿⣿⡅⠄⢀⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣦⣍⠲⣶⣤⣄⡀⠄⣴⣿⣿⣿⣿⣿⣿⣿
⣿⣿⣿⣿⣷⠄⣾⡏⢿⣿⣿⣿⣿⣿⣿⣿⠿⠿⠄⠹⣷⡌⢿⣿⣿⣷⣦⡙⢿⣿⣿⣿⣿⣿
⣿⣿⣿⣿⣿⣷⡌⢷⡘⣿⣿⣿⣿⣿⣿⣧⣀⣀⡀⠄⠈⠹⡈⣿⣿⣿⣿⣿⣦⡙⣿⣿⣿⣿
⣿⣿⣿⣿⣿⣿⣿⣎⢷⡘⢿⣿⣿⣿⣿⣿⣿⣿⠃⠄⣼⣶⡇⣿⣿⣿⣿⣿⣿⠓⠜⣿⣿⣿
⣿⣿⣿⣿⣿⣿⣿⣿⣎⢻⣦⡙⠿⣿⣿⣿⣿⣿⣿⣿⣿⠟⠄⣿⣿⣿⣿⣿⣿⣄⡀⢸⣿⣿
⣿⣿⣿⣿⣿⣿⣿⡿⢃⢼⣿⣿⣷⣤⣍⣉⣙⣛⣛⣉⣥⡄⠄⢿⣿⣿⣿⣿⡿⠟⣥⣿⣿⣿
⣿⣿⣿⣿⣿⡿⢋⣴⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣟⣿⣿⢁⣷⣤⣍⣉⣉⣭⣴⣾⣿⣿⣿⣿
=    WhiteScout - DorkDv - Go    =
`)
}

func randomUserAgent() string {
	agents := []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 Chrome/114.0.0.0 Safari/537.36",
		"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/100.0.0.0 Safari/537.36",
	}
	return agents[rand.Intn(len(agents))]
}

func incrementRequests() {
	requestsMutex.Lock()
	totalRequests++
	requestsMutex.Unlock()
}

func getTotalRequests() int {
	requestsMutex.Lock()
	defer requestsMutex.Unlock()
	return totalRequests
}

// ---------------- Crawler ----------------
func NewCrawler(target string, threads, delay, maxDepth, filterMode int) (*Crawler, error) {
	u, err := url.Parse(target)
	if err != nil {
		return nil, err
	}
	return &Crawler{
		BaseURL:     u,
		Visited:     make(map[string]bool),
		ThreadLimit: make(chan struct{}, threads),
		DelayMs:     delay,
		MaxDepth:    maxDepth,
		FilterMode:  filterMode,
	}, nil
}

func (c *Crawler) normalize(link string) string {
	u, err := url.Parse(link)
	if err != nil {
		return ""
	}
	if u.IsAbs() {
		if u.Host == c.BaseURL.Host {
			return u.String()
		}
		return ""
	}
	resolved := c.BaseURL.ResolveReference(u)
	if resolved.Host == c.BaseURL.Host {
		return resolved.String()
	}
	return ""
}

func (c *Crawler) Start() []string {
	var wg sync.WaitGroup
	c.Mutex.Lock()
	startURL := c.BaseURL.String()
	c.Visited[startURL] = true
	c.Results = append(c.Results, startURL)
	c.Mutex.Unlock()

	wg.Add(1)
	go c.fetchAndParse(startURL, &wg, 0)
	wg.Wait()
	return c.Results
}

func (c *Crawler) fetchAndParse(target string, wg *sync.WaitGroup, depth int) {
	defer wg.Done()
	if c.MaxDepth > 0 && depth > c.MaxDepth {
		return
	}

	c.ThreadLimit <- struct{}{}
	defer func() { <-c.ThreadLimit }()
	time.Sleep(time.Duration(rand.Intn(c.DelayMs)+1) * time.Millisecond)

	client := &http.Client{Timeout: 12 * time.Second}
	req, _ := http.NewRequest("GET", target, nil)
	req.Header.Set("User-Agent", randomUserAgent())
	incrementRequests()

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	tokenizer := html.NewTokenizer(bytes.NewReader(bodyBytes))
	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}
		t := tokenizer.Token()
		if t.Type == html.StartTagToken {
			for _, attr := range t.Attr {
				if attr.Key == "href" || attr.Key == "src" || attr.Key == "action" {
					link := c.normalize(attr.Val)
					if link != "" {
						c.addLink(link, wg, depth+1)
					}
				}
			}
		}
	}

	content := string(bodyBytes)
	foundLinks := extractEndpoints(content, c.BaseURL)
	for _, link := range foundLinks {
		c.addLink(link, wg, depth+1)
	}
}

func (c *Crawler) addLink(link string, wg *sync.WaitGroup, depth int) {
	link = c.normalize(link)
	if link == "" {
		return
	}

	switch c.FilterMode {
	case 1:
		if !isInteresting(link) {
			return
		}
	case 2:
		if isIgnored(link) {
			return
		}
	case 3:
	default:
		if isIgnored(link) {
			return
		}
	}

	c.Mutex.Lock()
	if c.Visited[link] {
		c.Mutex.Unlock()
		return
	}
	c.Visited[link] = true
	c.Mutex.Unlock()

	if !isValidEndpoint(link) {
		return
	}

	c.Mutex.Lock()
	c.Results = append(c.Results, link)
	c.Mutex.Unlock()

	wg.Add(1)
	go c.fetchAndParse(link, wg, depth)
}

// ---------------- Validação ----------------
func isValidEndpoint(link string) bool {
	client := &http.Client{Timeout: 8 * time.Second}

	req, _ := http.NewRequest("HEAD", link, nil)
	req.Header.Set("User-Agent", randomUserAgent())
	incrementRequests()
	resp, err := client.Do(req)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode < 400 {
			return true
		}
		return false
	}

	if strings.Contains(err.Error(), "received DATA on a HEAD request") || strings.Contains(err.Error(), "malformed HTTP response") {
		req2, _ := http.NewRequest("GET", link, nil)
		req2.Header.Set("User-Agent", randomUserAgent())
		req2.Header.Set("Range", "bytes=0-0")
		incrementRequests()
		resp2, err2 := client.Do(req2)
		if err2 != nil {
			return false
		}
		defer resp2.Body.Close()
		if resp2.StatusCode < 400 {
			return true
		}
		return false
	}
	return false
}

// ---------------- Filtragem ----------------
func isIgnored(link string) bool {
	u, err := url.Parse(link)
	if err == nil {
		path := u.Path
		for _, ext := range ignoreExt {
			if strings.HasSuffix(strings.ToLower(path), ext) {
				return true
			}
		}
	}
	for _, ext := range ignoreExt {
		if strings.HasSuffix(strings.ToLower(link), ext) {
			return true
		}
	}
	return false
}

func isInteresting(link string) bool {
	u, err := url.Parse(link)
	path := link
	if err == nil {
		path = u.Path
	}
	lower := strings.ToLower(path)
	for _, k := range interestingKeywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

// ---------------- Extração ----------------
func extractEndpoints(content string, baseURL *url.URL) []string {
	var links []string
	matches := endpointRegex.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		u, err := baseURL.Parse(m[1])
		if err == nil {
			links = append(links, u.String())
		}
	}
	return links
}

// ---------------- Exibição ----------------
func displayResults(results []string) {
	fmt.Println("\n======= Endpoints Coletados =======")
	for i, url := range results {
		fmt.Printf("%d - %s\n", i+1, url)
	}
}

// ---------------- Export ----------------
func askToExport(results []string) {
	input := readInput("\nDeseja exportar os endpoints para endpoints.txt? (S/N): ")
	if strings.ToLower(input) == "s" {
		file, err := os.Create("endpoints.txt")
		if err != nil {
			fmt.Println("Erro ao criar arquivo:", err)
			return
		}
		defer file.Close()
		writer := bufio.NewWriter(file)
		for _, u := range results {
			writer.WriteString(u + "\n")
		}
		writer.Flush()
		fmt.Println("[SUCESSO] Resultados exportados para endpoints.txt")
	} else {
		fmt.Println("Exportação ignorada.")
	}
}

// ---------------- Progress ----------------
func showLiveProgress(start time.Time) {
	for {
		time.Sleep(1 * time.Second)
		elapsed := time.Since(start).Seconds()
		rps := float64(getTotalRequests()) / elapsed
		fmt.Printf("\r[Progresso] Total Requests: %d | RPS: %.2f", getTotalRequests(), rps)
		if elapsed > 3600 {
			break
		}
	}
}
