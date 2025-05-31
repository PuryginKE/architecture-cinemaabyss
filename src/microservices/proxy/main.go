package main

import (
    "context"
    "fmt"
    "io"
    "log"
    "math/rand"
    "net/http"
    "os"
    "strconv"
    "time"
		"strings"

    "github.com/gin-gonic/gin"
)

func main() {
    rand.Seed(time.Now().UnixNano())

    port := os.Getenv("PORT")
    if port == "" {
        port = "8000"
    }

    monolithURL := os.Getenv("MONOLITH_URL")
    if monolithURL == "" {
        monolithURL = "http://localhost:8080"
    }

    moviesServiceURL := os.Getenv("MOVIES_SERVICE_URL")
    if moviesServiceURL == "" {
        moviesServiceURL = "http://localhost:8081"
    }

    gradualMigration := os.Getenv("GRADUAL_MIGRATION") == "true"
    moviesMigrationPercent, err := strconv.Atoi(os.Getenv("MOVIES_MIGRATION_PERCENT"))
    if err != nil || moviesMigrationPercent < 0 {
        moviesMigrationPercent = 50
    }

    r := gin.Default()

    r.GET("/health", func(c *gin.Context) {
        c.JSON(http.StatusOK, gin.H{"status": "OK"})
    })

    r.Any("/api/movies/*path", func(c *gin.Context) {
        if gradualMigration && rand.Intn(100) <= moviesMigrationPercent {
            forwardRequest(c, moviesServiceURL)
        } else {
            forwardRequest(c, monolithURL)
        }
    })

    r.Any("/api/users/*path", func(c *gin.Context) {
        forwardRequest(c, monolithURL)
    })

    log.Printf("Starting proxy service on port %s\n", port)
    r.Run(fmt.Sprintf(":%s", port))
}

func forwardRequest(c *gin.Context, targetURL string) {
		path := c.Request.URL.Path

		if strings.HasSuffix(path, "/") && len(path) > 1 {
			path = path[:len(path)-1]
		}

		fmt.Println("TEST", targetURL, path)

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    client := &http.Client{}
    httpRequest, err := http.NewRequestWithContext(ctx, c.Request.Method, targetURL+path, c.Request.Body)

		fmt.Println("TEST2", err, c.Request.Method, targetURL+path)

    if err != nil {
        log.Printf("Error creating HTTP request: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
        return
    }

    httpRequest.Header = c.Request.Header
    resp, err := client.Do(httpRequest)

		fmt.Println("TEST3", resp, err)
    if err != nil {
        log.Printf("Error forwarding request: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
        return
    }
    defer resp.Body.Close()

    body, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Printf("Error reading response body: %v", err)
        c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal Server Error"})
        return
    }

		fmt.Println("TEST4", resp, err)

    c.Writer.WriteHeader(resp.StatusCode)
    c.Writer.Header().Set("Content-Type", resp.Header.Get("Content-Type"))
    c.Writer.Write(body)
}