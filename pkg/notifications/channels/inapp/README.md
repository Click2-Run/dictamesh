# In-App WebSocket Notification Channel

SSR-compliant, real-time WebSocket implementation for DictaMesh in-app notifications.

## Features

- **SSR Compatible**: Gracefully handles both WebSocket and regular HTTP requests
- **Real-time Delivery**: Instant notification delivery via WebSocket connections
- **Connection Management**: Robust hub-based connection pooling and lifecycle management
- **Authentication**: Flexible authentication with Bearer token support
- **Multi-connection Support**: Users can have multiple simultaneous connections (e.g., multiple tabs/devices)
- **Channel Subscriptions**: Subscribe to specific notification channels/topics
- **Acknowledgments**: Track notification delivery and read receipts
- **Health Monitoring**: Built-in health checks and Prometheus metrics
- **Graceful Shutdown**: Clean connection termination with proper cleanup

## Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ WebSocket
       ↓
┌──────────────┐
│   Handler    │ ← Authentication
└──────┬───────┘
       │
       ↓
┌──────────────┐
│     Hub      │ ← Connection Manager
└──────┬───────┘
       │
       ↓
┌──────────────┐
│ Connections  │ ← User Connections
└──────────────┘
```

## Usage

### Basic Server Setup

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/click2-run/dictamesh/pkg/notifications/channels/inapp"
    "go.uber.org/zap"
)

func main() {
    // Create logger
    logger, _ := zap.NewProduction()

    // Configure provider
    providerConfig := &inapp.Config{
        Enabled:           true,
        Transport:         "websocket",
        PersistenceDays:   30,
        MaxUnread:         100,
        WebSocketPath:     "/ws",
        WebSocketPingTime: 30 * time.Second,
    }

    // Create provider
    provider := inapp.NewProvider(providerConfig, logger)

    // Configure server
    serverConfig := inapp.DefaultServerConfig()
    serverConfig.Address = "0.0.0.0"
    serverConfig.Port = 8080
    serverConfig.WebSocketPath = "/ws"

    // Create server
    server := inapp.NewServer(provider, serverConfig, logger)

    // Start server
    ctx := context.Background()
    if err := server.Start(ctx); err != nil {
        log.Fatal(err)
    }

    // Keep running
    select {}
}
```

### Sending Notifications

```go
// Send a notification to a user
notification := &inapp.Notification{
    ID:        "notif-123",
    UserID:    "user-456",
    Subject:   "New Message",
    Body:      "You have a new message from John",
    Priority:  "normal",
    Category:  "message",
    Data: map[string]interface{}{
        "sender": "john@example.com",
        "message_id": "msg-789",
    },
    CreatedAt: time.Now(),
    ActionURL: "/messages/msg-789",
}

err := provider.Send(ctx, notification)
```

### Custom Authentication

```go
// Implement custom authentication
customAuth := func(r *http.Request) (string, error) {
    token := r.URL.Query().Get("token")

    // Validate JWT token
    claims, err := validateJWT(token)
    if err != nil {
        return "", err
    }

    return claims.UserID, nil
}

serverConfig.AuthFunc = customAuth
```

## Client Integration

### JavaScript/TypeScript Client

```javascript
class NotificationClient {
    constructor(url, token) {
        this.url = url;
        this.token = token;
        this.ws = null;
        this.reconnectDelay = 1000;
    }

    connect() {
        this.ws = new WebSocket(`${this.url}?token=${this.token}`);

        this.ws.onopen = () => {
            console.log('WebSocket connected');
            this.reconnectDelay = 1000;
        };

        this.ws.onmessage = (event) => {
            const message = JSON.parse(event.data);
            this.handleMessage(message);
        };

        this.ws.onerror = (error) => {
            console.error('WebSocket error:', error);
        };

        this.ws.onclose = () => {
            console.log('WebSocket closed, reconnecting...');
            setTimeout(() => this.connect(), this.reconnectDelay);
            this.reconnectDelay = Math.min(this.reconnectDelay * 2, 30000);
        };
    }

    handleMessage(message) {
        switch (message.type) {
            case 'welcome':
                console.log('Connected with ID:', message.data.connection_id);
                break;
            case 'notification':
                this.onNotification(message.data);
                this.acknowledge(message.data.notification_id);
                break;
            case 'pong':
                console.log('Pong received');
                break;
        }
    }

    acknowledge(notificationId) {
        this.send({
            type: 'ack',
            data: { notification_id: notificationId }
        });
    }

    markAsRead(notificationId) {
        this.send({
            type: 'read',
            data: { notification_id: notificationId }
        });
    }

    subscribe(channels) {
        this.send({
            type: 'subscribe',
            data: { channels }
        });
    }

    send(message) {
        if (this.ws && this.ws.readyState === WebSocket.OPEN) {
            this.ws.send(JSON.stringify(message));
        }
    }

    onNotification(notification) {
        // Override this method to handle notifications
        console.log('New notification:', notification);
    }
}

// Usage
const client = new NotificationClient('ws://localhost:8080/ws', 'user-token');
client.onNotification = (notif) => {
    console.log('Received:', notif.subject, notif.body);
    // Show notification to user
};
client.connect();
```

### React Hook Example

```typescript
import { useEffect, useState, useCallback } from 'react';

export function useNotifications(wsUrl: string, token: string) {
    const [notifications, setNotifications] = useState<any[]>([]);
    const [connected, setConnected] = useState(false);
    const [ws, setWs] = useState<WebSocket | null>(null);

    useEffect(() => {
        const socket = new WebSocket(`${wsUrl}?token=${token}`);

        socket.onopen = () => {
            setConnected(true);
            console.log('Connected to notification service');
        };

        socket.onmessage = (event) => {
            const message = JSON.parse(event.data);
            if (message.type === 'notification') {
                setNotifications(prev => [...prev, message.data]);
            }
        };

        socket.onclose = () => {
            setConnected(false);
            // Implement reconnection logic
        };

        setWs(socket);

        return () => {
            socket.close();
        };
    }, [wsUrl, token]);

    const acknowledge = useCallback((notificationId: string) => {
        ws?.send(JSON.stringify({
            type: 'ack',
            data: { notification_id: notificationId }
        }));
    }, [ws]);

    const markAsRead = useCallback((notificationId: string) => {
        ws?.send(JSON.stringify({
            type: 'read',
            data: { notification_id: notificationId }
        }));
    }, [ws]);

    return { notifications, connected, acknowledge, markAsRead };
}
```

## Message Protocol

### Client → Server Messages

#### Acknowledge
```json
{
    "type": "ack",
    "data": {
        "notification_id": "notif-123"
    }
}
```

#### Mark as Read
```json
{
    "type": "read",
    "data": {
        "notification_id": "notif-123"
    }
}
```

#### Subscribe to Channels
```json
{
    "type": "subscribe",
    "data": {
        "channels": ["alerts", "messages"]
    }
}
```

### Server → Client Messages

#### Welcome
```json
{
    "type": "welcome",
    "timestamp": "2025-01-08T10:00:00Z",
    "data": {
        "user_id": "user-456",
        "connection_id": "conn-abc123",
        "server_time": 1704711600,
        "version": "1.0"
    }
}
```

#### Notification
```json
{
    "type": "notification",
    "id": "notif-123",
    "timestamp": "2025-01-08T10:00:00Z",
    "data": {
        "notification_id": "notif-123",
        "subject": "New Message",
        "body": "You have a new message",
        "priority": "normal",
        "category": "message",
        "created_at": "2025-01-08T10:00:00Z",
        "action_url": "/messages/123"
    }
}
```

## Monitoring

### Health Check
```bash
curl http://localhost:8080/health
```

Response:
```json
{
    "status": "healthy",
    "active_connections": 42,
    "total_connections": 42,
    "connections_created": 156,
    "connections_dropped": 114,
    "total_messages": 1234,
    "total_broadcasts": 567
}
```

### Metrics (Prometheus)
```bash
curl http://localhost:8080/metrics
```

## SSR Compatibility

The handler automatically detects WebSocket upgrade requests and handles regular HTTP requests gracefully:

```bash
# WebSocket request (upgraded)
curl -i -H "Connection: upgrade" -H "Upgrade: websocket" http://localhost:8080/ws

# Regular HTTP request (returns friendly error)
curl http://localhost:8080/ws
# Response: {"error":"WebSocket connection required",...}
```

## Configuration

### Provider Configuration
- `Enabled`: Enable/disable the provider
- `Transport`: Transport mechanism (websocket/sse/longpoll)
- `PersistenceDays`: How long to persist notifications
- `MaxUnread`: Maximum unread notifications per user
- `WebSocketPath`: WebSocket endpoint path
- `WebSocketPingTime`: Ping interval for keepalive

### Server Configuration
- `Address`: Server bind address
- `Port`: Server port
- `WebSocketPath`: WebSocket endpoint
- `AuthFunc`: Authentication function
- `TLSEnabled`: Enable TLS/SSL
- `ReadTimeout`: HTTP read timeout
- `WriteTimeout`: HTTP write timeout
- `IdleTimeout`: Connection idle timeout

## Testing

```bash
# Run tests
go test ./...

# Run with coverage
go test -cover ./...

# Integration tests
go test -tags=integration ./...
```

## License

SPDX-License-Identifier: AGPL-3.0-or-later
Copyright (C) 2025 Controle Digital Ltda
