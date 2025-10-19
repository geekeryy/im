package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserInfoModel = (*customUserInfoModel)(nil)

type (
	// UserInfoModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserInfoModel.
	UserInfoModel interface {
		userInfoModel
		withSession(session sqlx.Session) UserInfoModel
		FindByUuid(ctx context.Context, userUuid string) (*UserInfo, error)
	}

	customUserInfoModel struct {
		*defaultUserInfoModel
	}
)

// NewUserInfoModel returns a model for the database table.
func NewUserInfoModel(conn sqlx.SqlConn) UserInfoModel {
	return &customUserInfoModel{
		defaultUserInfoModel: newUserInfoModel(conn),
	}
}

func (m *customUserInfoModel) withSession(session sqlx.Session) UserInfoModel {
	return NewUserInfoModel(sqlx.NewSqlConnFromSession(session))
}

// 根据用户UUID查询用户信息
func (m *customUserInfoModel) FindByUuid(ctx context.Context, userUuid string) (*UserInfo, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE uuid = ?", m.table)
	var resp UserInfo
	err := m.conn.QueryRowCtx(ctx, &resp, query, userUuid)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &resp, nil
}
