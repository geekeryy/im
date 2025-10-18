package model

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserBaseModel = (*customUserBaseModel)(nil)

type (
	// UserBaseModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserBaseModel.
	UserBaseModel interface {
		userBaseModel
		withSession(session sqlx.Session) UserBaseModel
		FindByUuid(ctx context.Context, uuid string) (*UserBase, error)
		FindByUuids(ctx context.Context, uuids []string) ([]*UserBase, error)
	}

	customUserBaseModel struct {
		*defaultUserBaseModel
	}
)

// NewUserBaseModel returns a model for the database table.
func NewUserBaseModel(conn sqlx.SqlConn) UserBaseModel {
	return &customUserBaseModel{
		defaultUserBaseModel: newUserBaseModel(conn),
	}
}

func (m *customUserBaseModel) withSession(session sqlx.Session) UserBaseModel {
	return NewUserBaseModel(sqlx.NewSqlConnFromSession(session))
}

// 根据用户UUID查询用户基本信息
func (m *customUserBaseModel) FindByUuid(ctx context.Context, uuid string) (*UserBase, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE uuid = ?", m.table)
	var resp UserBase
	err := m.conn.QueryRowCtx(ctx, &resp, query, uuid)
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("find user base by uuid %s failed", uuid))
	}
	return &resp, nil
}

// 批量查询用户基本信息
func (m *customUserBaseModel) FindByUuids(ctx context.Context, uuids []string) ([]*UserBase, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE uuid in (?)", m.table)
	var resp []*UserBase
	err := m.conn.QueryRowCtx(ctx, &resp, query, strings.Join(uuids,","))
	if err != nil {
		return nil, errors.Join(err, fmt.Errorf("find user base by uuids %v failed", uuids))
	}
	return resp, nil
}

