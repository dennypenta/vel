---
title: Server-Sent Events (SSE)
description: Implement real-time streaming with Server-Sent Events in vel applications.
---

## Overview

Server-Sent Events (SSE) provide a simple way to stream real-time data from your vel server to clients. Unlike WebSockets, SSE is unidirectional (server-to-client only) and works over standard HTTP, making it perfect for use cases like:

- Real-time progress updates
- Live notifications
- Build/deployment status streaming
- System monitoring feeds
- Chat message feeds (receive-only)

This tutorial shows how to implement SSE endpoints in vel and consume them from TypeScript and Go clients.

### Key Concepts

- **Event Stream**: Continuous HTTP response with `text/event-stream` content type
- **Data Format**: Each message follows the format `data: {JSON}\n\n`
- **Connection Management**: Clients automatically reconnect on connection loss
- **Context Cancellation**: Server can detect client disconnection via context

:::note[Client Generation Limitation]
vel's automatic client generation doesn't currently support SSE endpoints. You'll need to implement SSE clients manually as shown in this tutorial.
:::

## Server Implementation

Implement SSE endpoints in vel by accessing the response writer directly and streaming data with proper headers.

### Basic SSE Handler

```go
type ProgressRequest struct {
    DeploymentID string `schema:"deploymentID"`
}

type ProgressMessage struct {
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`
    Payload   string    `json:"payload"`
    Final     bool      `json:"final"`
}

type ProgressResponse struct {
    Message ProgressMessage `json:"message"`
}

func GetBuildProgress(ctx context.Context, req ProgressRequest) (ProgressResponse, *vel.Error) {
    w := vel.WriterFromContext(ctx)

    // Set SSE headers
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*") // Adjust for your CORS policy

    // Check if streaming is supported
    flusher, ok := w.(http.Flusher)
    if !ok {
        return ProgressResponse{}, &vel.Error{
            Code:    "STREAMING_UNSUPPORTED",
            Message: "Server does not support streaming",
        }
    }

    // Get progress channel (implementation depends on your progress system)
    messages := getProgressChannel(ctx, req.DeploymentID)

    for {
        select {
        case message, ok := <-messages:
            if !ok {
                // Channel closed, end stream
                return ProgressResponse{}, nil
            }

            // Create response message
            response := ProgressResponse{Message: message}
            
            // Marshal to JSON
            data, err := json.Marshal(response)
            if err != nil {
                return ProgressResponse{}, &vel.Error{
                    Code:    "JSON_MARSHAL_ERROR",
                    Message: "Failed to marshal progress message",
                    Err:     err,
                }
            }

            // Send SSE formatted message
            fmt.Fprintf(w, "data: %s\n\n", data)
            flusher.Flush()

            // Close connection if this is the final message
            if message.Final {
                return ProgressResponse{}, nil
            }

        case <-ctx.Done():
            // Client disconnected or context cancelled
            return ProgressResponse{}, nil
        }
    }
}
```

### Advanced SSE Handler with Error Handling

```go
func GetBuildProgressAdvanced(ctx context.Context, req ProgressRequest) (ProgressResponse, *vel.Error) {
    w := vel.WriterFromContext(ctx)
    
    // Set SSE headers
    setupSSEHeaders(w)
    
    flusher, ok := w.(http.Flusher)
    if !ok {
        return ProgressResponse{}, &vel.Error{
            Code:    "STREAMING_UNSUPPORTED",
            Message: "Server does not support streaming",
        }
    }

    // Validate deployment exists
    if !deploymentExists(req.DeploymentID) {
        return ProgressResponse{}, &vel.Error{
            Code:    "DEPLOYMENT_NOT_FOUND",
            Message: "Deployment not found",
        }
    }

    // Send initial connection confirmation
    initialMessage := ProgressMessage{
        Timestamp: time.Now(),
        Level:     "INFO",
        Payload:   "Connected to build progress stream",
        Final:     false,
    }
    
    if err := sendSSEMessage(w, flusher, ProgressResponse{Message: initialMessage}); err != nil {
        return ProgressResponse{}, &vel.Error{
            Code:    "SEND_ERROR",
            Message: "Failed to send initial message",
            Err:     err,
        }
    }

    // Start streaming progress
    messages := getProgressChannel(ctx, req.DeploymentID)
    
    // Send heartbeat every 30 seconds to keep connection alive
    heartbeat := time.NewTicker(30 * time.Second)
    defer heartbeat.Stop()

    for {
        select {
        case message, ok := <-messages:
            if !ok {
                // Send final message and close
                finalMessage := ProgressMessage{
                    Timestamp: time.Now(),
                    Level:     "INFO",
                    Payload:   "Stream ended",
                    Final:     true,
                }
                sendSSEMessage(w, flusher, ProgressResponse{Message: finalMessage})
                return ProgressResponse{}, nil
            }

            if err := sendSSEMessage(w, flusher, ProgressResponse{Message: message}); err != nil {
                return ProgressResponse{}, &vel.Error{
                    Code:    "SEND_ERROR",
                    Message: "Failed to send progress message",
                    Err:     err,
                }
            }

            if message.Final {
                return ProgressResponse{}, nil
            }

        case <-heartbeat.C:
            // Send heartbeat to keep connection alive
            heartbeatMsg := ProgressMessage{
                Timestamp: time.Now(),
                Level:     "HEARTBEAT",
                Payload:   "Connection alive",
                Final:     false,
            }
            sendSSEMessage(w, flusher, ProgressResponse{Message: heartbeatMsg})

        case <-ctx.Done():
            return ProgressResponse{}, nil
        }
    }
}

func setupSSEHeaders(w http.ResponseWriter) {
    w.Header().Set("Content-Type", "text/event-stream")
    w.Header().Set("Cache-Control", "no-cache")
    w.Header().Set("Connection", "keep-alive")
    w.Header().Set("Access-Control-Allow-Origin", "*")
    w.Header().Set("Access-Control-Allow-Headers", "Cache-Control")
}

func sendSSEMessage(w http.ResponseWriter, flusher http.Flusher, response ProgressResponse) error {
    data, err := json.Marshal(response)
    if err != nil {
        return err
    }
    
    _, err = fmt.Fprintf(w, "data: %s\n\n", data)
    if err != nil {
        return err
    }
    
    flusher.Flush()
    return nil
}
```

### Register SSE Handler

```go
func main() {
    router := vel.NewRouter()
    
    // Register SSE endpoint as GET handler
    vel.RegisterGet(router, "getBuildProgress", GetBuildProgress)
    
    http.ListenAndServe(":8080", router.Mux())
}
```

## Client Implementation

Since vel's client generation doesn't support SSE, implement clients manually using the appropriate SSE libraries for each platform.

### TypeScript Client

```typescript
interface ProgressMessage {
  timestamp: string;
  level: string;
  payload: string;
  final: boolean;
}

interface GetBuildProgressResponse {
  message: ProgressMessage;
}

class SSEClient {
  private baseUrl: string;

  constructor(baseUrl: string) {
    this.baseUrl = baseUrl;
  }

  listenProgress(
    deploymentID: string, 
    callback: (data: GetBuildProgressResponse) => void,
    onError?: (error: Event) => void,
    onClose?: () => void
  ): () => void {
    const url = this.buildUrl('getBuildProgress', { deploymentID });
    
    const eventSource = new EventSource(url, { 
      withCredentials: true 
    });

    eventSource.addEventListener('message', (event) => {
      try {
        const data: GetBuildProgressResponse = JSON.parse(event.data);
        callback(data);

        if (data.message.final) {
          eventSource.close();
          console.log('SSE stream ended');
          onClose?.();
        }
      } catch (error) {
        console.error('Failed to parse SSE message:', error);
        onError?.(event);
      }
    });

    eventSource.addEventListener('error', (event) => {
      console.error('SSE connection error:', event);
      onError?.(event);
    });

    eventSource.addEventListener('open', () => {
      console.log('SSE connection opened');
    });

    // Return cleanup function
    return () => {
      eventSource.close();
    };
  }

  private buildUrl(endpoint: string, params: Record<string, string>): string {
    const url = new URL(`${this.baseUrl}/${endpoint}`);
    Object.entries(params).forEach(([key, value]) => {
      url.searchParams.append(key, value);
    });
    return url.toString();
  }
}

// Usage example
const client = new SSEClient('https://api.example.com');

const cleanup = client.listenProgress(
  'deployment-123',
  (data) => {
    console.log(`[${data.message.level}] ${data.message.payload}`);
    
    if (data.message.final) {
      console.log('Build completed!');
    }
  },
  (error) => {
    console.error('Connection error:', error);
  },
  () => {
    console.log('Stream closed');
  }
);

// Clean up when component unmounts or when needed
// cleanup();
```

### React Hook for SSE

```typescript
import { useEffect, useCallback, useRef } from 'react';

interface UseSSEOptions {
  onMessage: (data: GetBuildProgressResponse) => void;
  onError?: (error: Event) => void;
  onClose?: () => void;
  enabled?: boolean;
}

export function useSSE(deploymentID: string, options: UseSSEOptions) {
  const { onMessage, onError, onClose, enabled = true } = options;
  const eventSourceRef = useRef<EventSource | null>(null);
  const cleanupRef = useRef<(() => void) | null>(null);

  const connect = useCallback(() => {
    if (!enabled || !deploymentID) return;

    const client = new SSEClient(process.env.REACT_APP_API_URL!);
    
    const cleanup = client.listenProgress(
      deploymentID,
      onMessage,
      onError,
      onClose
    );

    cleanupRef.current = cleanup;
  }, [deploymentID, enabled, onMessage, onError, onClose]);

  const disconnect = useCallback(() => {
    if (cleanupRef.current) {
      cleanupRef.current();
      cleanupRef.current = null;
    }
  }, []);

  useEffect(() => {
    connect();
    return disconnect;
  }, [connect, disconnect]);

  return { disconnect };
}

// Usage in component
function BuildProgress({ deploymentID }: { deploymentID: string }) {
  const [messages, setMessages] = useState<ProgressMessage[]>([]);
  const [isComplete, setIsComplete] = useState(false);

  const { disconnect } = useSSE(deploymentID, {
    onMessage: (data) => {
      setMessages(prev => [...prev, data.message]);
      if (data.message.final) {
        setIsComplete(true);
      }
    },
    onError: (error) => {
      console.error('SSE error:', error);
    },
    onClose: () => {
      console.log('Build progress stream ended');
    },
    enabled: !isComplete
  });

  return (
    <div>
      <h3>Build Progress</h3>
      {messages.map((msg, index) => (
        <div key={index} className={`message message-${msg.level.toLowerCase()}`}>
          [{new Date(msg.timestamp).toLocaleTimeString()}] {msg.payload}
        </div>
      ))}
      {isComplete && <button onClick={disconnect}>Close Stream</button>}
    </div>
  );
}
```

### Go Client

```go
package main

import (
    "bufio"
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "strings"
    "time"
)

type ProgressMessage struct {
    Timestamp time.Time `json:"timestamp"`
    Level     string    `json:"level"`
    Payload   string    `json:"payload"`
    Final     bool      `json:"final"`
}

type GetBuildProgressResponse struct {
    Message ProgressMessage `json:"message"`
}

type SSEClient struct {
    baseURL string
    client  *http.Client
}

func NewSSEClient(baseURL string) *SSEClient {
    return &SSEClient{
        baseURL: baseURL,
        client: &http.Client{
            Timeout: 0, // No timeout for streaming connections
        },
    }
}

func (c *SSEClient) ListenProgress(
    ctx context.Context,
    deploymentID string,
    callback func(GetBuildProgressResponse),
) error {
    url := fmt.Sprintf("%s/getBuildProgress?deploymentID=%s", c.baseURL, deploymentID)
    
    req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
    if err != nil {
        return fmt.Errorf("creating request: %w", err)
    }
    
    // Set headers for SSE
    req.Header.Set("Accept", "text/event-stream")
    req.Header.Set("Cache-Control", "no-cache")
    
    resp, err := c.client.Do(req)
    if err != nil {
        return fmt.Errorf("making request: %w", err)
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
    }
    
    scanner := bufio.NewScanner(resp.Body)
    
    for scanner.Scan() {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        
        line := scanner.Text()
        if line == "" {
            continue
        }
        
        // Parse SSE message
        if strings.HasPrefix(line, "data: ") {
            data := strings.TrimPrefix(line, "data: ")
            
            var response GetBuildProgressResponse
            if err := json.Unmarshal([]byte(data), &response); err != nil {
                fmt.Printf("Error parsing SSE message: %v\n", err)
                continue
            }
            
            callback(response)
            
            if response.Message.Final {
                return nil
            }
        }
    }
    
    return scanner.Err()
}

// Usage example
func main() {
    client := NewSSEClient("http://localhost:8080")
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()
    
    err := client.ListenProgress(ctx, "deployment-123", func(response GetBuildProgressResponse) {
        msg := response.Message
        fmt.Printf("[%s] %s: %s\n", 
            msg.Timestamp.Format(time.RFC3339), 
            msg.Level, 
            msg.Payload)
        
        if msg.Final {
            fmt.Println("Build completed!")
        }
    })
    
    if err != nil {
        fmt.Printf("SSE client error: %v\n", err)
    }
}
```

### Go Client with Reconnection

```go
type SSEClientWithReconnect struct {
    *SSEClient
    maxRetries int
    retryDelay time.Duration
}

func NewSSEClientWithReconnect(baseURL string, maxRetries int) *SSEClientWithReconnect {
    return &SSEClientWithReconnect{
        SSEClient:  NewSSEClient(baseURL),
        maxRetries: maxRetries,
        retryDelay: time.Second * 2,
    }
}

func (c *SSEClientWithReconnect) ListenProgressWithReconnect(
    ctx context.Context,
    deploymentID string,
    callback func(GetBuildProgressResponse),
) error {
    var lastError error
    
    for attempt := 0; attempt <= c.maxRetries; attempt++ {
        if attempt > 0 {
            fmt.Printf("Reconnection attempt %d/%d after %v\n", 
                attempt, c.maxRetries, c.retryDelay)
            
            select {
            case <-ctx.Done():
                return ctx.Err()
            case <-time.After(c.retryDelay):
            }
        }
        
        err := c.ListenProgress(ctx, deploymentID, callback)
        if err == nil {
            return nil // Stream completed successfully
        }
        
        lastError = err
        
        // Don't retry on context cancellation
        if ctx.Err() != nil {
            return ctx.Err()
        }
        
        fmt.Printf("SSE connection failed: %v\n", err)
    }
    
    return fmt.Errorf("failed after %d attempts: %w", c.maxRetries, lastError)
}
```

## Best Practices

### Error Handling

```go
func GetBuildProgressRobust(ctx context.Context, req ProgressRequest) (ProgressResponse, *vel.Error) {
    w := vel.WriterFromContext(ctx)
    setupSSEHeaders(w)
    
    flusher, ok := w.(http.Flusher)
    if !ok {
        return ProgressResponse{}, &vel.Error{
            Code: "STREAMING_UNSUPPORTED",
            Message: "Server does not support streaming",
        }
    }

    // Send error as SSE message instead of returning error
    sendError := func(code, message string) {
        errorMsg := ProgressMessage{
            Timestamp: time.Now(),
            Level:     "ERROR",
            Payload:   message,
            Final:     true,
        }
        sendSSEMessage(w, flusher, ProgressResponse{Message: errorMsg})
    }

    // Validate input
    if req.DeploymentID == "" {
        sendError("MISSING_DEPLOYMENT_ID", "Deployment ID is required")
        return ProgressResponse{}, nil
    }

    if !deploymentExists(req.DeploymentID) {
        sendError("DEPLOYMENT_NOT_FOUND", "Deployment not found")
        return ProgressResponse{}, nil
    }

    // Continue with normal streaming...
    messages := getProgressChannel(ctx, req.DeploymentID)
    // ... rest of implementation
}
```

### Connection Management

```go
// Track active SSE connections
type SSEManager struct {
    connections map[string]chan ProgressMessage
    mu          sync.RWMutex
}

func (m *SSEManager) AddConnection(deploymentID string) chan ProgressMessage {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    ch := make(chan ProgressMessage, 100) // Buffered channel
    m.connections[deploymentID] = ch
    return ch
}

func (m *SSEManager) RemoveConnection(deploymentID string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if ch, exists := m.connections[deploymentID]; exists {
        close(ch)
        delete(m.connections, deploymentID)
    }
}

func (m *SSEManager) BroadcastProgress(deploymentID string, message ProgressMessage) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    if ch, exists := m.connections[deploymentID]; exists {
        select {
        case ch <- message:
        default:
            // Channel buffer full, skip message or handle overflow
            fmt.Printf("SSE channel buffer full for deployment %s\n", deploymentID)
        }
    }
}
```

### Performance Considerations

1. **Buffer Management**: Use buffered channels to prevent blocking
2. **Connection Limits**: Implement connection pooling and limits
3. **Memory Usage**: Clean up closed connections promptly
4. **Heartbeats**: Send periodic heartbeats to detect disconnections
5. **Graceful Shutdown**: Handle server shutdown gracefully

```go
func (h *Handler) Shutdown(ctx context.Context) error {
    // Close all active SSE connections
    h.sseManager.CloseAll()
    
    // Wait for connections to close or timeout
    select {
    case <-h.allConnectionsClosed:
        return nil
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

Server-Sent Events provide an excellent way to stream real-time data from vel applications. While client generation isn't supported yet, the manual implementation patterns shown above provide robust, production-ready SSE capabilities for both browser and server applications.
