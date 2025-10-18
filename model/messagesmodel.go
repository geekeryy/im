package model

import (
	"context"
	"errors"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ MessagesModel = (*customMessagesModel)(nil)

type (
	// MessagesModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMessagesModel.
	MessagesModel interface {
		messagesModel
		withSession(session sqlx.Session) MessagesModel
		FindLatestMessageBySessionUuid(ctx context.Context, sessionUuid string) (*Messages, error)
		FindMessagesBySeqidGreaterThan(ctx context.Context, sessionUuid string, startSeqid int64) ([]*Messages, error)
	}

	customMessagesModel struct {
		*defaultMessagesModel
	}
)

// NewMessagesModel returns a model for the database table.
func NewMessagesModel(conn sqlx.SqlConn) MessagesModel {
	return &customMessagesModel{
		defaultMessagesModel: newMessagesModel(conn),
	}
}

func (m *customMessagesModel) withSession(session sqlx.Session) MessagesModel {
	return NewMessagesModel(sqlx.NewSqlConnFromSession(session))
}

// 查询会话的最新一条消息
func (m *customMessagesModel) FindLatestMessageBySessionUuid(ctx context.Context, sessionUuid string) (*Messages, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE session_uuid = ? ORDER BY updated_at DESC LIMIT 1", m.table)
	var resp Messages
	err := m.conn.QueryRowCtx(ctx, &resp, query, sessionUuid)
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("find latest message by session uuid %s failed", sessionUuid))
	}
	return &resp, nil
}

// 查询大于指定序列号的消息列表
func (m *customMessagesModel) FindMessagesBySeqidGreaterThan(ctx context.Context, sessionUuid string, startSeqid int64) ([]*Messages, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE session_uuid = ? AND seq_id > ? ORDER BY seq_id ASC", m.table)
	var resp []*Messages
	err := m.conn.QueryRowsCtx(ctx, &resp, query, sessionUuid, startSeqid)
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("find messages by seqid %d failed", startSeqid))
	}
	return resp, nil
}
