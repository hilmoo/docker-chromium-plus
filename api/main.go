package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

func screenshotHandler(w http.ResponseWriter, r *http.Request) {
	cmd := exec.Command("scrot", "-")
	cmd.Env = append(os.Environ(), "DISPLAY=:1")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		http.Error(w, "Failed to capture screenshot", 500)
		return
	}

	if err := cmd.Start(); err != nil {
		http.Error(w, "Failed to start scrot", 500)
		return
	}

	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")

	if _, err := io.Copy(w, stdout); err != nil {
		log.Println("stream error:", err)
	}

	cmd.Wait()
}

func main() {
	http.HandleFunc("/screenshot", screenshotHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	formattedPort := ":" + port
	fmt.Printf("Screenshot API running at http://localhost%s\n", formattedPort)

	log.Fatal(http.ListenAndServe(formattedPort, nil))
}
