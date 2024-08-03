package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gliderlabs/ssh"
	"github.com/teris-io/shortid"
	gossh "golang.org/x/crypto/ssh"
)

func main() {
	sshPort := ":2222"

	respCh := make(chan string)
	go func() {
		time.Sleep(time.Second * 3)
		id, _ := shortid.Generate()
		respCh <- "http://webhooker.com/" + id

		time.Sleep(time.Second * 5)
		for {
			time.Sleep(time.Second * 2)
			respCh <- "received data from hook"
		}
	}()
	handler := &SHHHandler{
		respCh: respCh,
	}

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
	log.Fatal(server.ListenAndServe())

}

type SHHHandler struct {
	respCh chan string
}

func (h *SHHHandler) handleSSFSession(session ssh.Session) {
	forwardURL := session.RawCommand()
	_ = forwardURL
	resp := <-h.respCh
	fmt.Println(resp)
	if forwardURL == "" {
		fmt.Println("No command received.")
		return
	}
	session.Write([]byte(resp + "\n"))

	for data := range h.respCh {
		session.Write([]byte(data + "\n"))
	}

	//fmt.Fprintf(session, "You sent: %s\n", forwardURL) // writes back tot he client

}
