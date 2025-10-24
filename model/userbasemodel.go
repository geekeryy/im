package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"im/pkg/xstrings"

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
		RegisterUserBase(ctx context.Context, tx sqlx.Session, data *UserBase) error
		FindAll(ctx context.Context) ([]*UserBase, error)
	}

	customUserBaseModel struct {
		*defaultUserBaseModel
	}
)

// 用户状态
const (
	UserStatusActive   = 1 // 活跃
	UserStatusInactive = 2 // 不活跃
	UserStatusDeleted  = 3 // 删除
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
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Join(err, fmt.Errorf("find user base by uuid %s failed", uuid))
	}
	return &resp, nil
}

// 批量查询用户基本信息
func (m *customUserBaseModel) FindByUuids(ctx context.Context, uuids []string) ([]*UserBase, error) {
	queryStr, args := xstrings.BuildInQuery(uuids)
	query := fmt.Sprintf("SELECT * FROM %s WHERE uuid in (%s)", m.table, queryStr)
	var resp []*UserBase

	err := m.conn.QueryRowsCtx(ctx, &resp, query, args...)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Join(err, fmt.Errorf("find user base by uuids %v failed", uuids))
	}
	return resp, nil
}

// 注册用户基本信息
func (m *customUserBaseModel) RegisterUserBase(ctx context.Context, tx sqlx.Session, data *UserBase) error {
	query := fmt.Sprintf("insert into %s (uuid, name, avatar, status) values (?, ?, ?, ?)", m.table)
	_, err := m.conn.ExecCtx(ctx, query, data.Uuid, data.Name, data.Avatar, data.Status)
	return err
}

// 查询所有用户基本信息
func (m *customUserBaseModel) FindAll(ctx context.Context) ([]*UserBase, error) {
	query := fmt.Sprintf("SELECT * FROM %s", m.table)
	var resp []*UserBase
	err := m.conn.QueryRowsCtx(ctx, &resp, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Join(err, fmt.Errorf("find all user base failed"))
	}
	return resp, nil
}
