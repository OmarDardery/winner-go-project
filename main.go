package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

var clients = make(map[*websocket.Conn]string) // store connections with role (user/admin)
var mu sync.Mutex

func handleWebSocket(c *gin.Context) {
	c.Writer.Header().Set("Connection", "Upgrade")
	c.Writer.Header().Set("Upgrade", "websocket")
	role := c.Query("role") // admin or user
	if role == "" {
		role = "user" // default to user if not specified
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		fmt.Println("Upgrade error:", err)
		return
	}
	defer conn.Close()

	mu.Lock()
	clients[conn] = role
	mu.Unlock()

	fmt.Printf("✅ %s connected\n", role)

	for {
		messageType, msg, err := conn.ReadMessage()
		if err != nil {
			fmt.Printf("❌ %s disconnected\n", role)
			break
		}

		fmt.Printf("📩 Message from %s: %s\n", role, msg)

		if role == "admin" {
			// Pick one random user
			mu.Lock()
			var userConns []*websocket.Conn
			for c, r := range clients {
				if r == "user" {
					userConns = append(userConns, c)
				}
			}
			if len(userConns) > 0 {
				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				targetIndex := r.Intn(len(userConns))
				for i := 0; i < len(userConns); i++ {
					if i == targetIndex {
						userConns[i].WriteMessage(messageType, []byte("Winner"))
					} else {
						userConns[i].WriteMessage(messageType, []byte("loser"))
					}
				}
			} else {
				fmt.Println("⚠️ No users connected")
			}
			mu.Unlock()
		}
	}

	mu.Lock()
	delete(clients, conn)
	mu.Unlock()
}

func main() {
	r := gin.Default()
	r.Static("/static", "./static")
	r.LoadHTMLGlob("templates/*.html")

	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	r.GET("/admin-zain", func(c *gin.Context) {
		c.HTML(http.StatusOK, "admin.html", nil)
	})

	r.GET("/ws", handleWebSocket)

	fmt.Println("🚀 Server running on http://localhost:8080")
	r.Run(":8080")
}
