package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/dop251/goja"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

var (
	orModel      string
	orKey        string
	orEndpoint   string
	middleware   string
	systemPrompt string
	jsRuntime    *goja.Runtime
	noHosts      bool
	routerFile   string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	orModel = getEnv("OR_MODEL", "")
	orKey = getEnv("OR_KEY", "")
	orEndpoint = getEnv("OR_ENDPOINT", "")

	flag.StringVar(&middleware, "middleware", "", "Path to JavaScript middleware file")
	flag.StringVar(&systemPrompt, "system", "", "Path to system prompt file")
	flag.BoolVar(&noHosts, "nohosts", false, "Skip /etc/hosts check and modification")
	flag.StringVar(&routerFile, "router", "", "Path to JavaScript router file")
	flag.Parse()

	if middleware != "" {
		jsRuntime = goja.New()
		js, err := ioutil.ReadFile(middleware)
		if err != nil {
			log.Fatalf("Error reading middleware file: %v", err)
		}
		_, err = jsRuntime.RunString(string(js))
		if err != nil {
			log.Fatalf("Error running middleware: %v", err)
		}
	}

	if routerFile != "" {
		if jsRuntime == nil {
			jsRuntime = goja.New()
		}
		js, err := ioutil.ReadFile(routerFile)
		if err != nil {
			log.Fatalf("Error reading router file: %v", err)
		}
		_, err = jsRuntime.RunString(string(js))
		if err != nil {
			log.Fatalf("Error running router script: %v", err)
		}
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func checkAndFixHosts() {
	hostsFile := "/etc/hosts"
	if os.Getenv("GOOS") == "windows" {
		hostsFile = "C:\\Windows\\System32\\drivers\\etc\\hosts"
	}

	content, err := ioutil.ReadFile(hostsFile)
	if err != nil {
		log.Printf("Error reading hosts file: %v", err)
		return
	}

	lines := strings.Split(string(content), "\n")
	anthropicFound := false
	openaiFound := false

	for _, line := range lines {
		if strings.Contains(line, "api.anthropic.com") {
			anthropicFound = true
		}
		if strings.Contains(line, "api.openai.com") {
			openaiFound = true
		}
	}

	if !anthropicFound || !openaiFound {
		fmt.Println("The hosts file needs to be updated. Do you want to fix it? (y/n)")
		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "y" || response == "yes" {
			err := fixHostsFile(hostsFile)
			if err != nil {
				log.Printf("Error fixing hosts file: %v", err)
			} else {
				fmt.Println("Hosts file updated successfully.")
			}
		}
	}
}

func fixHostsFile(hostsFile string) error {
	content := "127.0.0.1 api.anthropic.com\n127.0.0.1 api.openai.com\n"

	var cmd *exec.Cmd
	if os.Getenv("GOOS") == "windows" {
		cmd = exec.Command("powershell", "-Command", fmt.Sprintf("Add-Content -Path '%s' -Value '%s' -Force", hostsFile, content))
	} else {
		cmd = exec.Command("sudo", "sh", "-c", fmt.Sprintf("echo '%s' >> %s", content, hostsFile))
	}

	return cmd.Run()
}

func processRequest(body []byte, isAnthropicAPI bool) ([]byte, string, string, string, error) {
	var requestBody map[string]interface{}
	err := json.Unmarshal(body, &requestBody)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("error unmarshaling request body: %v", err)
	}

	if systemPrompt != "" {
		systemContent, err := ioutil.ReadFile(systemPrompt)
		if err != nil {
			return nil, "", "", "", fmt.Errorf("error reading system prompt file: %v", err)
		}
		if isAnthropicAPI {
			prompt := fmt.Sprintf("%s\n\nHuman: %s\n\nAssistant:", string(systemContent), requestBody["prompt"])
			requestBody["prompt"] = prompt
		} else {
			messages, ok := requestBody["messages"].([]interface{})
			if !ok {
				return nil, "", "", "", fmt.Errorf("invalid or missing 'messages' field in request")
			}
			systemMessage := map[string]interface{}{
				"role":    "system",
				"content": string(systemContent),
			}
			requestBody["messages"] = append([]interface{}{systemMessage}, messages...)
		}
	}

	if middleware != "" {
		processFunc, ok := goja.AssertFunction(jsRuntime.Get("process"))
		if !ok {
			return nil, "", "", "", fmt.Errorf("middleware does not contain a 'process' function")
		}

		result, err := processFunc(goja.Undefined(), jsRuntime.ToValue(requestBody))
		if err != nil {
			return nil, "", "", "", fmt.Errorf("error executing middleware: %v", err)
		}

		err = jsRuntime.ExportTo(result, &requestBody)
		if err != nil {
			return nil, "", "", "", fmt.Errorf("error exporting middleware result: %v", err)
		}
	}

	model := orModel
	url := orEndpoint
	bearer := orKey

	if routerFile != "" {
		routeFunc, ok := goja.AssertFunction(jsRuntime.Get("route"))
		if !ok {
			return nil, "", "", "", fmt.Errorf("router does not contain a 'route' function")
		}

		result, err := routeFunc(goja.Undefined(), jsRuntime.ToValue(requestBody))
		if err != nil {
			return nil, "", "", "", fmt.Errorf("error executing router: %v", err)
		}

		routeResult := result.Export().(map[string]interface{})
		model = routeResult["model"].(string)
		url = routeResult["url"].(string)
		bearer = routeResult["bearer"].(string)
	}

	requestBody["model"] = model

	processedBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, "", "", "", fmt.Errorf("error marshaling processed request: %v", err)
	}

	return processedBody, model, url, bearer, nil
}

func handleRequest(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	isAnthropicAPI := strings.Contains(r.URL.Path, "/v1/complete")

	processedBody, _, url, bearer, err := processRequest(body, isAnthropicAPI)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error processing request: %v", err), http.StatusInternalServerError)
		return
	}

	client := &http.Client{}
	req, err := http.NewRequest("POST", url, strings.NewReader(string(processedBody)))
	if err != nil {
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		return
	}

	req.Header = r.Header
	req.Header.Set("Authorization", "Bearer "+bearer)
	req.Header.Set("HTTP-Referer", "https://github.com/PentBeear/Rythm-OpenRouter-backend")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error forwarding request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)

	if r.URL.Query().Get("stream") == "true" {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
			return
		}

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Printf("Error reading stream: %v", err)
				break
			}

			_, err = w.Write(line)
			if err != nil {
				log.Printf("Error writing to response: %v", err)
				break
			}
			flusher.Flush()
		}
	} else {
		io.Copy(w, resp.Body)
	}
}
func logRequest(r *http.Request, code int, duration time.Duration) {
	log.Printf("%s %s %s %d %v", r.RemoteAddr, r.Method, r.URL.Path, code, duration)
}

// loggingMiddleware wraps an http.HandlerFunc and logs request details
func loggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Create a custom ResponseWriter to capture the status code
		lrw := &loggingResponseWriter{w, 200}

		// Call the next handler
		next.ServeHTTP(lrw, r)

		// Log the request details
		duration := time.Since(start)
		logRequest(r, lrw.statusCode, duration)
	}
}

// loggingResponseWriter is a custom ResponseWriter that captures the status code
type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
func main() {
	if !noHosts {
		checkAndFixHosts()
	}

	// Create a new ServeMux for our handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/chat/completions", loggingMiddleware(handleRequest)) // OpenAI endpoint
	mux.HandleFunc("/v1/complete", loggingMiddleware(handleRequest))         // Anthropic endpoint

	// Create an error group to manage both servers
	var g errgroup.Group

	// Start HTTP server
	g.Go(func() error {
		httpServer := &http.Server{
			Addr:    ":80",
			Handler: mux,
		}
		fmt.Println("HTTP Server is running on port 80")
		return httpServer.ListenAndServe()
	})

	// Start HTTPS server
	g.Go(func() error {
		// Load SSL certificate and key
		cert, err := tls.LoadX509KeyPair("server.crt", "server.key")
		if err != nil {
			log.Fatalf("Error loading SSL certificate and key: %v", err)
		}

		// Configure the TLS server
		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{cert},
		}

		httpsServer := &http.Server{
			Addr:      ":443",
			Handler:   mux,
			TLSConfig: tlsConfig,
		}

		fmt.Println("HTTPS Server is running on port 443")
		return httpsServer.ListenAndServeTLS("", "")
	})

	// Wait for both servers and log any errors
	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}