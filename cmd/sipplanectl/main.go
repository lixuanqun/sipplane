package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

var authToken string

func main() {
	server := flag.String("server", envOr("SIPPLANE_CONTROL", "http://127.0.0.1:8090"), "control plane base URL")
	token := flag.String("token", envOr("SIPPLANE_CONTROL_TOKEN", ""), "Bearer token (SIPPLANE_CONTROL_TOKEN)")
	flag.Parse()
	authToken = *token
	args := flag.Args()
	if len(args) < 1 {
		usage()
		os.Exit(2)
	}
	switch args[0] {
	case "apply":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: sipplanectl apply <file-or-dir>")
			os.Exit(2)
		}
		body, err := readYAML(args[1])
		if err != nil {
			fatal(err)
		}
		post(*server+"/v1/apply", body)
	case "dry-run":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: sipplanectl dry-run <file-or-dir>")
			os.Exit(2)
		}
		body, err := readYAML(args[1])
		if err != nil {
			fatal(err)
		}
		post(*server+"/v1/dry-run", body)
	case "revision":
		get(*server + "/v1/revision")
	case "snapshot":
		get(*server + "/v1/snapshot")
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `sipplanectl — sipplane control-plane client

Usage:
  sipplanectl apply <file-or-dir>
  sipplanectl dry-run <file-or-dir>
  sipplanectl revision
  sipplanectl snapshot

Flags:
  --server URL   control plane base URL
  --token TOKEN  Bearer token for /v1/*

Env:
  SIPPLANE_CONTROL        base URL (default http://127.0.0.1:8090)
  SIPPLANE_CONTROL_TOKEN  Bearer token
`)
}

func readYAML(path string) ([]byte, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return os.ReadFile(path)
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(e.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(e.Name()), "bootstrap") &&
			(strings.HasSuffix(strings.ToLower(e.Name()), ".yaml") || strings.HasSuffix(strings.ToLower(e.Name()), ".yml")) {
			continue
		}
		data, err := os.ReadFile(filepath.Join(path, e.Name()))
		if err != nil {
			return nil, err
		}
		buf.Write(data)
		if !bytes.HasSuffix(data, []byte("\n")) {
			buf.WriteByte('\n')
		}
		buf.WriteString("---\n")
	}
	return buf.Bytes(), nil
}

func setAuth(req *http.Request) {
	if authToken != "" {
		req.Header.Set("Authorization", "Bearer "+authToken)
	}
}

func post(url string, body []byte) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		fatal(err)
	}
	req.Header.Set("Content-Type", "application/yaml")
	req.Header.Set("X-Actor", "sipplanectl")
	setAuth(req)
	res, err := client.Do(req)
	if err != nil {
		fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	fmt.Print(string(out))
	if res.StatusCode >= 300 {
		os.Exit(1)
	}
}

func get(url string) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		fatal(err)
	}
	setAuth(req)
	res, err := client.Do(req)
	if err != nil {
		fatal(err)
	}
	defer res.Body.Close()
	out, _ := io.ReadAll(res.Body)
	fmt.Print(string(out))
	if res.StatusCode >= 300 {
		os.Exit(1)
	}
}

func envOr(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
