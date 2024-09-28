package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/joho/godotenv"
)

var (
	orModel    string
	orKey      string
	orEndpoint string
)

func init() {
	err := godotenv.Load()
	if err != nil {
		log.Println("Error loading .env file")
	}

	orModel = getEnv("OR_MODEL", "")
	orKey = getEnv("OR_KEY", "")
	orEndpoint = getEnv("OR_ENDPOINT", "")

	if orModel == "" || orKey == "" || orEndpoint == "" {
		log.Fatal("Missing required environment variables")
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

func handleRequest(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	client := &http.Client{}
	req, err := http.NewRequest("POST", orEndpoint, strings.NewReader(string(body)))
	if err != nil {
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		return
	}

	req.Header = r.Header
	req.Header.Set("Authorization", "Bearer "+orKey)
	req.Header.Set("HTTP-Referer", "https://github.com/PentBeear/Rythm-OpenRouter-backend")

	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error forwarding request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, "Error reading response body", http.StatusInternalServerError)
		return
	}

	for k, v := range resp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(resp.StatusCode)
	w.Write(respBody)
}

func main() {
	checkAndFixHosts()

	http.HandleFunc("/", handleRequest)

	fmt.Println("Server is running on port 80")
	log.Fatal(http.ListenAndServe(":80", nil))
}