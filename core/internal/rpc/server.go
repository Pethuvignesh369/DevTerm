package rpc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
)

// Request represents a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response represents a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RpcError   `json:"error,omitempty"`
}

// RpcError represents a JSON-RPC 2.0 error object.
type RpcError struct {
	Code    string      `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Notifier allows managers to push notifications to the frontend.
type Notifier struct {
	mu     *sync.Mutex
	writer io.Writer
}

var globalNotifier *Notifier

// GetNotifier returns the global notifier instance.
func GetNotifier() *Notifier {
	return globalNotifier
}

// Notify sends a JSON-RPC notification (no id) to the frontend.
func (n *Notifier) Notify(method string, params interface{}) {
	n.mu.Lock()
	defer n.mu.Unlock()

	msg := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
	}
	body, err := json.Marshal(msg)
	if err != nil {
		log.Printf("failed to marshal notification: %v", err)
		return
	}
	frame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(body), body)
	if _, err := io.WriteString(n.writer, frame); err != nil {
		log.Printf("failed to write notification: %v", err)
	}
}

// Serve runs the JSON-RPC server reading from stdin and writing to stdout.
// It uses Content-Length framing (LSP-style).
func Serve(stdin io.Reader, stdout io.Writer, dispatcher *Dispatcher) error {
	reader := bufio.NewReader(stdin)
	var writeMu sync.Mutex
	// Responses and notifications share one stream and must therefore share
	// one lock; otherwise their Content-Length frames can interleave.
	globalNotifier = &Notifier{writer: stdout, mu: &writeMu}

	for {
		// Read Content-Length header
		headerLine, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("reading header: %w", err)
		}
		headerLine = strings.TrimSpace(headerLine)
		if !strings.HasPrefix(headerLine, "Content-Length: ") {
			continue
		}
		lengthStr := strings.TrimPrefix(headerLine, "Content-Length: ")
		contentLength, err := strconv.Atoi(lengthStr)
		if err != nil {
			log.Printf("invalid Content-Length: %s", lengthStr)
			continue
		}

		// Read the empty separator line
		if _, err := reader.ReadString('\n'); err != nil {
			return fmt.Errorf("reading separator: %w", err)
		}

		// Read the body
		body := make([]byte, contentLength)
		if _, err := io.ReadFull(reader, body); err != nil {
			return fmt.Errorf("reading body: %w", err)
		}

		// Parse the request
		var req Request
		if err := json.Unmarshal(body, &req); err != nil {
			log.Printf("invalid JSON-RPC request: %v", err)
			continue
		}

		// Handle request asynchronously
		go func(req Request) {
			result, callErr := dispatcher.Dispatch(req.Method, req.Params)

			var resp Response
			resp.JSONRPC = "2.0"
			resp.ID = req.ID

			if callErr != nil {
				resp.Error = &RpcError{
					Code:    "INTERNAL",
					Message: callErr.Error(),
				}
			} else {
				resp.Result = result
			}

			respBody, err := json.Marshal(resp)
			if err != nil {
				log.Printf("failed to marshal response: %v", err)
				return
			}

			frame := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(respBody), respBody)

			writeMu.Lock()
			if _, err := io.WriteString(stdout, frame); err != nil {
				log.Printf("failed to write response: %v", err)
			}
			writeMu.Unlock()
		}(req)
	}
}
