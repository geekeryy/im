package model

import (
	"context"
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
	}

	customSessionsModel struct {
		*defaultSessionsModel
	}
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
		return nil, errors.Join(err, fmt.Errorf("find session by uuid %s failed", uuid))
	}
	return &resp, nil
}
