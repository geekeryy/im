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
