package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// 응답 본문을 구성하는 JSON 페이로드 구조체
type responsePayload struct {
	Data    string `json:"data"`
	Version string `json:"version"`
}

// 루트 경로 핸들러: GET 요청만 허용하고 JSON 환영 메시지를 반환
func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Printf("failed to read hostname: %v", err)
		hostname = "unknown"
	}

	payload := responsePayload{
		Data:    "Welcome! " + hostname,
		Version: "v3.2",
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to write response: %v", err)
	}
}

func main() {
	// mux를 생성해 핸들러를 등록
	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler)

	// 서버 타임아웃을 지정해 느린 클라이언트로부터 보호
	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	log.Printf("starting http-go server (pid=%d) on %s", os.Getpid(), server.Addr)

	// 서버를 별도 고루틴에서 실행하고 오류를 감시
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server error: %v", err)
		}
	}()

	// SIGINT, SIGTERM 신호를 받아 그레이스풀 셧다운 진행
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	log.Println("shutdown signal received")

	// 셧다운 타임아웃을 적용해 연결 종료를 기다림
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}

	log.Println("server stopped")
}

