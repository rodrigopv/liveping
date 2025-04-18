package ping

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	probing "github.com/prometheus-community/pro-bing"
	"github.com/gorilla/websocket"
)

// WebSocket Upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Hub manages WebSocket connections
type Hub struct {
	clients     map[*websocket.Conn]bool
	broadcast   chan []byte
	register    chan *websocket.Conn
	unregister  chan *websocket.Conn
	mu          sync.Mutex
	pinger      *probing.Pinger
	statsTicker *time.Ticker
	statsDone   chan struct{}
}

func newHub() *Hub {
	return &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
		statsDone:  make(chan struct{}),
	}
}

// Run the Hub in a separate goroutine
func (h *Hub) run() {
	for {
		select {
		case conn := <-h.register:
			h.mu.Lock()
			h.clients[conn] = true
			log.Printf("Client registered: %s", conn.RemoteAddr())
			h.mu.Unlock()
		case conn := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[conn]; ok {
				delete(h.clients, conn)
				conn.Close()
				log.Printf("Client unregistered: %s", conn.RemoteAddr())
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.Lock()
			for conn := range h.clients {
				err := conn.WriteMessage(websocket.TextMessage, message)
				if err != nil {
					log.Printf("Error writing to client %s: %v. Unregistering.", conn.RemoteAddr(), err)
					go func(c *websocket.Conn) {
						h.unregister <- c
					}(conn)
				}
			}
			h.mu.Unlock()
		}
	}
}

// PingResult structure for JSON marshalling
type PingResult struct {
	LatencyMs *float64 `json:"latency_ms"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
	Seq       int       `json:"seq"`
	LostCount int       `json:"lost_count"`
}

// PingStats structure for statistics
type PingStats struct {
	PacketsSent    int     `json:"packets_sent"`
	PacketsRecv    int     `json:"packets_recv"`
	PacketLoss     float64 `json:"packet_loss"`
	MinRtt         float64 `json:"min_rtt"`
	MaxRtt         float64 `json:"max_rtt"`
	AvgRtt         float64 `json:"avg_rtt"`
	StdDevRtt      float64 `json:"std_dev_rtt"`
	Timestamp      time.Time `json:"timestamp"`
}

// RunServer starts the ping server with the given configuration
func RunServer(config Config) error {
	log.Printf("Starting Live Ping Tool")
	log.Printf("Target Host: %s", config.TargetHost)
	log.Printf("Ping Interval: %s", config.PingInterval)
	log.Printf("Web Server Listening on: %s", config.ListenAddr)
	if config.Verbose {
		log.Printf("Verbose mode enabled")
	}

	hub := newHub()
	go hub.run()

	// Start the pinger
	go runPinger(config.TargetHost, config.PingInterval, hub, config.Verbose)

	// Setup HTTP Server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		serveHTML(w, r, config.TargetHost)
	})
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})

	// Start HTTP server in a goroutine
	go func() {
		log.Printf("Web server started.")
		if err := http.ListenAndServe(config.ListenAddr, nil); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	// Graceful shutdown handling
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs // Wait for termination signal

	log.Println("Shutting down...")
	return nil
}

// Serves the index.html file
func serveHTML(w http.ResponseWriter, r *http.Request, target string) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	
	htmlStr := strings.ReplaceAll(indexHTML, "#{target}", target)
	
	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte(htmlStr))
}

// Handles WebSocket connections
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade WebSocket connection: %v", err)
		return
	}

	hub.register <- conn

	go func() {
		defer func() {
			hub.unregister <- conn
		}()
		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("Client %s read error: %v", conn.RemoteAddr(), err)
				} else {
					log.Printf("Client %s disconnected.", conn.RemoteAddr())
				}
				break
			}

			if string(message) == "RESTART" {
				hub.mu.Lock()
				if hub.pinger != nil {
					hub.pinger.Stop()
					pinger, err := probing.NewPinger(hub.pinger.Addr())
					if err != nil {
						log.Printf("Failed to create new pinger: %v", err)
						hub.mu.Unlock()
						continue
					}
					pinger.Interval = hub.pinger.Interval
					pinger.Count = -1
					hub.pinger = pinger
					go runPinger(hub.pinger.Addr(), hub.pinger.Interval, hub, false)
				}
				hub.mu.Unlock()
			}
		}
	}()
}

// Runs the pinger and sends results to the hub
func runPinger(target string, interval time.Duration, hub *Hub, verbose bool) {
	log.Printf("Starting pinger for %s...", target)

	hub.mu.Lock()
	if hub.pinger != nil {
		hub.pinger.Stop()
	}
	if hub.statsTicker != nil {
		hub.statsTicker.Stop()
		close(hub.statsDone)
		hub.statsDone = make(chan struct{})
	}
	hub.mu.Unlock()

	pinger, err := probing.NewPinger(target)
	if err != nil {
		log.Printf("Failed to create pinger: %v", err)
		return
	}

	pinger.Interval = interval
	pinger.Count = -1

	hub.mu.Lock()
	hub.pinger = pinger
	hub.statsTicker = time.NewTicker(1 * time.Second)
	hub.mu.Unlock()

	var lastSeq int = -1

	pinger.OnRecv = func(pkt *probing.Packet) {
		latency := float64(pkt.Rtt.Microseconds()) / 1000.0
		lostCount := 0
		
		if lastSeq != -1 {
			lostCount = pkt.Seq - lastSeq - 1
		}
		lastSeq = pkt.Seq

		result := PingResult{
			LatencyMs: &latency,
			Timestamp: time.Now(),
			Seq:       pkt.Seq,
			LostCount: lostCount,
		}

		if lostCount > 0 {
			log.Printf("Detected %d lost packets between seq %d and %d\n", lostCount, lastSeq-lostCount, pkt.Seq)
			// Send a special message for each lost packet
			for i := 0; i < lostCount; i++ {
				lostSeq := lastSeq - lostCount + i
				lostMsg := fmt.Sprintf("LOST_PACKET:%d", lostSeq)
				hub.broadcast <- []byte(lostMsg)
			}
		}

		// Only log ping responses in verbose mode
		if verbose {
			log.Printf("%d bytes from %s: icmp_seq=%d time=%.3f ms\n",
				pkt.Nbytes, pkt.IPAddr, pkt.Seq, latency)
		}

		jsonData, err := json.Marshal(result)
		if err != nil {
			log.Printf("Error marshalling ping result: %v", err)
			return
		}
		hub.broadcast <- jsonData
	}

	pinger.OnDuplicateRecv = func(pkt *probing.Packet) {
		latency := float64(pkt.Rtt.Microseconds()) / 1000.0
		result := PingResult{
			LatencyMs: &latency,
			Timestamp: time.Now(),
			Error:     "duplicate",
			Seq:       pkt.Seq,
		}
		
		// Only log duplicate packet in verbose mode
		if verbose {
			log.Printf("Duplicate packet received: %d bytes from %s: icmp_seq=%d time=%.3f ms (DUP!)\n",
				pkt.Nbytes, pkt.IPAddr, pkt.Seq, latency)
		}

		jsonData, err := json.Marshal(result)
		if err != nil {
			log.Printf("Error marshalling duplicate result: %v", err)
			return
		}
		hub.broadcast <- jsonData
	}

	pinger.OnFinish = func(stats *probing.Statistics) {
		log.Printf("\nPing statistics for %s:\n", stats.Addr)
		log.Printf("  %d packets transmitted, %d packets received, %v%% packet loss\n",
			stats.PacketsSent, stats.PacketsRecv, stats.PacketLoss)
		log.Printf("round-trip min/avg/max/stddev = %v/%v/%v/%v\n",
			stats.MinRtt, stats.AvgRtt, stats.MaxRtt, stats.StdDevRtt)
	}

	go func() {
		for {
			select {
			case <-hub.statsTicker.C:
				stats := pinger.Statistics()
				pingStats := PingStats{
					PacketsSent: stats.PacketsSent,
					PacketsRecv: stats.PacketsRecv,
					PacketLoss:  stats.PacketLoss,
					MinRtt:      float64(stats.MinRtt.Microseconds()) / 1000.0,
					MaxRtt:      float64(stats.MaxRtt.Microseconds()) / 1000.0,
					AvgRtt:      float64(stats.AvgRtt.Microseconds()) / 1000.0,
					StdDevRtt:   float64(stats.StdDevRtt.Microseconds()) / 1000.0,
					Timestamp:   time.Now(),
				}
				
				jsonData, err := json.Marshal(pingStats)
				if err != nil {
					log.Printf("Error marshalling stats: %v", err)
					continue
				}
				
				statsMsg := append([]byte("STATS:"), jsonData...)
				hub.broadcast <- statsMsg
			case <-hub.statsDone:
				return
			}
		}
	}()

	log.Printf("Running pinger... (Press Ctrl+C to stop)")
	err = pinger.Run()
	if err != nil {
		log.Printf("Pinger failed: %v", err)
	}
} 