package model

import (
	"context"
	"database/sql"
	"fmt"

	"errors"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ SessionMembersModel = (*customSessionMembersModel)(nil)

type (
	// SessionMembersModel is an interface to be customized, add more methods here,
	// and implement the added methods in customSessionMembersModel.
	SessionMembersModel interface {
		sessionMembersModel
		withSession(session sqlx.Session) SessionMembersModel
		FindSessionsByUserUuid(ctx context.Context, userUuid string) ([]string, error)
		JoinSession(ctx context.Context, tx sqlx.Session, sessionUuid string, userUuid string) error
		FindAllMembersBySessionUuid(ctx context.Context, sessionUuid string) ([]string, error)
	}

	customSessionMembersModel struct {
		*defaultSessionMembersModel
	}
)

// NewSessionMembersModel returns a model for the database table.
func NewSessionMembersModel(conn sqlx.SqlConn) SessionMembersModel {
	return &customSessionMembersModel{
		defaultSessionMembersModel: newSessionMembersModel(conn),
	}
}

func (m *customSessionMembersModel) withSession(session sqlx.Session) SessionMembersModel {
	return NewSessionMembersModel(sqlx.NewSqlConnFromSession(session))
}

// 查找用户加入的会话列表
func (m *customSessionMembersModel) FindSessionsByUserUuid(ctx context.Context, userUuid string) ([]string, error) {
	resp := []string{}
	query := fmt.Sprintf("SELECT session_uuid FROM %s WHERE user_uuid = ?", m.table)
	err := m.conn.QueryRowsCtx(ctx, &resp, query, userUuid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Join(err, fmt.Errorf("find sessions by user uuid %s failed", userUuid))
	}
	return resp, nil
}

// 加入会话
func (m *customSessionMembersModel) JoinSession(ctx context.Context, tx sqlx.Session, sessionUuid string, userUuid string) error {
	var conn sqlx.Session
	if tx == nil {
		conn = m.conn
	} else {
		conn = tx
	}
	query := fmt.Sprintf("INSERT INTO %s (session_uuid, user_uuid) VALUES (?, ?)", m.table)
	_, err := conn.ExecCtx(ctx, query, sessionUuid, userUuid)
	if err != nil {
		return errors.Join(err, fmt.Errorf("join session failed"))
	}
	return nil
}

// 查找会话中的所有成员
func (m *customSessionMembersModel) FindAllMembersBySessionUuid(ctx context.Context, sessionUuid string) ([]string, error) {
	resp := []string{}
	query := fmt.Sprintf("SELECT user_uuid FROM %s WHERE session_uuid = ?", m.table)
	err := m.conn.QueryRowsCtx(ctx, &resp, query, sessionUuid)
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("find all members by session uuid %s failed", sessionUuid))
	}
	return resp, nil
}
