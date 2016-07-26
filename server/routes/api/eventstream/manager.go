package eventstream

import (
	"log"
	"sync"
	"time"

	"github.com/tchap/steemwatch/notifications/events"
	"github.com/tchap/steemwatch/server/context"
	"github.com/tchap/steemwatch/server/users"

	"github.com/labstack/echo"
	"github.com/labstack/echo/engine/fasthttp"
	"github.com/pkg/errors"
	"github.com/tchap/websocket"
	"gopkg.in/mgo.v2/bson"
)

type connectionRecord struct {
	conn *websocket.Conn
	lock *sync.Mutex
}

type Manager struct {
	connections map[string]*connectionRecord
	closed      bool
	lock        *sync.RWMutex
}

func NewManager() *Manager {
	return &Manager{
		connections: make(map[string]*connectionRecord),
		lock:        &sync.RWMutex{},
	}
}

func (manager *Manager) Bind(serverCtx *context.Context, group *echo.Group) {
	group.GET("/ws/", func(ctx echo.Context) error {
		user := ctx.Get("user").(*users.User)

		var upgrader websocket.FastHTTPUpgrader

		upgrader.Handler = func(conn *websocket.Conn) {
			defer conn.Close()
			manager.lock.Lock()

			if manager.closed {
				manager.lock.Unlock()
				return
			}

			// Close any existing connection for the user.
			// This is perhaps not idea, but it at least prevents leaking connections.
			record, ok := manager.connections[user.Id]
			if ok {
				record.conn.Close()
			}

			// Insert the new connection record into the map.
			manager.connections[user.Id] = &connectionRecord{conn, &sync.Mutex{}}
			log.Println(
				"WebSocket connection added. Number of connections:", len(manager.connections))
			manager.lock.Unlock()

			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					manager.lock.Lock()
					delete(manager.connections, user.Id)
					log.Println(
						"WebSocket connection removed. Number of connections:",
						len(manager.connections))
					manager.lock.Unlock()
					return
				}
			}
		}

		return fasthttp.WrapHandler(upgrader.UpgradeHandler)(ctx)
	})
}

func (manager *Manager) sendEvent(userId string, event interface{}) error {
	manager.lock.RLock()
	defer manager.lock.RUnlock()

	if manager.closed {
		return nil
	}

	record, ok := manager.connections[userId]
	if !ok {
		return nil
	}

	record.lock.Lock()
	defer record.lock.Unlock()

	if err := record.conn.SetWriteDeadline(time.Now().Add(10 * time.Second)); err != nil {
		return errors.Wrap(err, "failed to set write deadline")
	}
	return record.conn.WriteJSON(event)
}

func (manager *Manager) Close() error {
	manager.lock.Lock()
	defer manager.lock.Unlock()

	manager.closed = true

	for _, record := range manager.connections {
		record.conn.Close()
	}

	return nil
}

func (manager *Manager) DispatchAccountUpdatedEvent(
	userId string,
	_ bson.Raw,
	event *events.AccountUpdated,
) error {
	return manager.sendEvent(userId, formatAccountUpdated(event))
}

func (manager *Manager) DispatchTransferMadeEvent(
	userId string,
	_ bson.Raw,
	event *events.TransferMade,
) error {
	return manager.sendEvent(userId, formatTransferMade(event))
}

func (manager *Manager) DispatchUserMentionedEvent(
	userId string,
	_ bson.Raw,
	event *events.UserMentioned,
) error {
	return manager.sendEvent(userId, formatUserMentioned(event))
}

func (manager *Manager) DispatchStoryPublishedEvent(
	userId string,
	_ bson.Raw,
	event *events.StoryPublished,
) error {
	return manager.sendEvent(userId, formatStoryPublished(event))
}

func (manager *Manager) DispatchStoryVotedEvent(
	userId string,
	_ bson.Raw,
	event *events.StoryVoted,
) error {
	return manager.sendEvent(userId, formatStoryVoted(event))
}

func (manager *Manager) DispatchCommentPublishedEvent(
	userId string,
	_ bson.Raw,
	event *events.CommentPublished,
) error {
	return manager.sendEvent(userId, formatCommentPublished(event))
}

func (manager *Manager) DispatchCommentVotedEvent(
	userId string,
	_ bson.Raw,
	event *events.CommentVoted,
) error {
	return manager.sendEvent(userId, formatCommentVoted(event))
}
