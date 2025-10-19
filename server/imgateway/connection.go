package imgateway

import (
	"net"
	"sync"

	"github.com/google/uuid"
)

type Session struct {
	user_uuids []string
}

type Connection struct {
	user_uuid string
	conn      net.Conn
}

type ConnManager struct {
	locker        sync.RWMutex
	sessions      map[string]*Session
	connections   map[string]*Connection
	user_conn_map map[string]string
}

func NewConnManager() *ConnManager {
	return &ConnManager{
		sessions:      make(map[string]*Session),
		connections:   make(map[string]*Connection),
		user_conn_map: make(map[string]string),
	}
}

func (c *ConnManager) AddConnection(user_uuid string, conn net.Conn) string {
	conn_uuid := uuid.New().String()
	c.locker.Lock()
	defer c.locker.Unlock()
	c.connections[conn_uuid] = &Connection{
		user_uuid: user_uuid,
		conn:      conn,
	}
	c.user_conn_map[user_uuid] = conn_uuid
	return conn_uuid
}
func (c *ConnManager) GetConnection(conn_uuid string) *Connection {
	c.locker.RLock()
	defer c.locker.RUnlock()
	return c.connections[conn_uuid]
}

func (c *ConnManager) AddSession(session_uuid string, user_uuids []string) *Session {
	c.locker.Lock()
	defer c.locker.Unlock()
	session := &Session{
		user_uuids: user_uuids,
	}
	c.sessions[session_uuid] = session
	return session
}

func (c *ConnManager) GetSession(session_uuid string) *Session {
	c.locker.RLock()
	defer c.locker.RUnlock()
	return c.sessions[session_uuid]
}

func (c *ConnManager) GetUserConnUUID(user_uuid string) string {
	c.locker.RLock()
	defer c.locker.RUnlock()
	return c.user_conn_map[user_uuid]
}
