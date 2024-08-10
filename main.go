package main

import (
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gliderlabs/ssh"
	gossh "golang.org/x/crypto/ssh"
	"golang.org/x/term"
)

var clients sync.Map

type HTTPHandler struct {
}

func (h *HTTPHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	fmt.Println(r.PathValue("id"))
	id := r.PathValue("id")
	ch, ok := clients.Load(id)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("client id not found"))

	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		log.Fatal(err)
	}
	defer r.Body.Close()
	ch.(chan string) <- string(b)
}

func startHTTPServer() error {
	httpPort := ":5000"
	router := http.NewServeMux()

	handler := &HTTPHandler{}
	router.HandleFunc("/{id}/*", handler.handleWebhook)
	return http.ListenAndServe(httpPort, router)

}

func startSSHServer() error {
	sshPort := ":2222"
	handler := NewSSHHandler()

	fwhandler := &ssh.ForwardedTCPHandler{}
	server := ssh.Server{
		Addr:    sshPort,
		Handler: handler.handleSSFSession,
		ServerConfigCallback: func(ctx ssh.Context) *gossh.ServerConfig {
			cfg := &gossh.ServerConfig{
				ServerVersion: "SSH-2.0-sendit",
			}
			cfg.Ciphers = []string{"chacha20-poly1305@openssh.com"}
			return cfg
		},
		PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
			return true
		},
		LocalPortForwardingCallback: ssh.LocalPortForwardingCallback(func(ctx ssh.Context, dhost string, dport uint32) bool {
			log.Println("Accepted foward", dhost, dport)
			return true

		}),
		ReversePortForwardingCallback: ssh.ReversePortForwardingCallback(func(ctx ssh.Context, host string, port uint32) bool {
			log.Println("Accepted foward", host, port, "granted")
			return true
		}),
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        fwhandler.HandleSSHRequest,
			"cancel-tcpip-forward": fwhandler.HandleSSHRequest,
		},
	}
	b, err := os.ReadFile("./keys/privatekey.pub")
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println("here : ", b)
	privateKey, err := gossh.ParsePrivateKey(b)
	if err != nil {
		log.Fatal("Failed to parse private key: ", err)
	}
	server.AddHostKey(privateKey)
	log.Printf("Starting SSH server on port %s", sshPort)
	return (server.ListenAndServe())

}

func main() {

	go startSSHServer()
	startHTTPServer()

}

type SSHHandler struct {
	channels map[string]chan string
}

func NewSSHHandler() *SSHHandler {
	return &SSHHandler{
		channels: make(map[string]chan string),
	}
}

func (h *SSHHandler) handleSSFSession(session ssh.Session) {

	term := term.NewTerminal(session, "")

	for {
		input, err := term.ReadLine()
		if err != nil {
			log.Fatal()
		}
		if len(input) == 0 {
			term.Write([]byte("Welcome to webhooker!\n\nenter webhook distination:"))

		}
		fmt.Println(input)

		if strings.Contains(input, "ssh -R") {
			for {
				time.Sleep(time.Second)
			}
		}

		generatedPort := randomPort()
		webhookURL := fmt.Sprintf("http://localhost:%d", generatedPort)
		command := fmt.Sprintf("\nGenerated webhook: %s\n\nComand to copy:\nssh -R 127.0.0.1:%d:%s localhost -p 2222\n", webhookURL, generatedPort, input)
		term.Write([]byte(command))
		return
	}

}

func randomPort() int {
	min := 49152
	max := 65535
	return min + rand.Intn(max-min+1)

}
