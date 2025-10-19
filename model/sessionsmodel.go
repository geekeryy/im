package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SessionsModel = (*customSessionsModel)(nil)

type (
	// SessionsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSessionsModel.
	SessionsModel interface {
		sessionsModel
		withSession(session sqlx.Session) SessionsModel
		FindByUuid(ctx context.Context, uuid string) (*Sessions, error)
		CreateSession(ctx context.Context,tx sqlx.Session, session *Sessions) error
	}

	customSessionsModel struct {
		*defaultSessionsModel
	}
)

const (
	SessionTypeSingle = 1 // 单聊
	SessionTypeGroup = 2 // 群聊
)

const (
	SessionStatusActive = 1 // 活跃
	SessionStatusInactive = 2 // 不活跃
	SessionStatusDeleted = 3 // 删除
)

// NewSessionsModel returns a model for the database table.
func NewSessionsModel(conn sqlx.SqlConn) SessionsModel {
	return &customSessionsModel{
		defaultSessionsModel: newSessionsModel(conn),
	}
}

func (m *customSessionsModel) withSession(session sqlx.Session) SessionsModel {
	return NewSessionsModel(sqlx.NewSqlConnFromSession(session))
}

// 根据会话UUID查询会话详细信息
func (m *customSessionsModel) FindByUuid(ctx context.Context, uuid string) (*Sessions, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE uuid = ?", m.table)
	var resp Sessions
	err := m.conn.QueryRowCtx(ctx, &resp, query, uuid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Join(err, fmt.Errorf("find session by uuid %s failed", uuid))
	}
	return &resp, nil
}

// 创建会话
func (m *customSessionsModel) CreateSession(ctx context.Context,tx sqlx.Session, session *Sessions) error {
	var conn sqlx.Session
	if tx == nil {
		conn = m.conn
	} else {
		conn = tx
	}
	query := fmt.Sprintf("INSERT INTO %s (uuid, name, avatar, session_type, status) VALUES (?, ?, ?, ?, ?)", m.table)
	_, err := conn.ExecCtx(ctx, query, session.Uuid, session.Name, session.Avatar, session.SessionType, session.Status)
	if err != nil {
		return errors.Join(err, fmt.Errorf("create session failed"))
	}
	return nil
}
