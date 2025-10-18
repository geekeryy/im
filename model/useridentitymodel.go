package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ UserIdentityModel = (*customUserIdentityModel)(nil)

type (
	// UserIdentityModel is an interface to be customized, add more methods here,
	// and implement the added methods in customUserIdentityModel.
	UserIdentityModel interface {
		userIdentityModel
		withSession(session sqlx.Session) UserIdentityModel
		FindByIdentifierAndIdentityType(ctx context.Context, identifier string, identityType int64) (*UserIdentity, error)
		RegisterUserIdentity(ctx context.Context, tx sqlx.Session, identity *UserIdentity) error
	}

	customUserIdentityModel struct {
		*defaultUserIdentityModel
	}
)

// 身份类型
const (
	IdentityTypePhone    = 1
	IdentityTypeEmail    = 2
	IdentityTypePassword = 3
	IdentityTypeWechat   = 4
	IdentityTypeGoogle   = 5
	IdentityTypeFacebook = 6
	IdentityTypeGithub   = 7
)

// NewUserIdentityModel returns a model for the database table.
func NewUserIdentityModel(conn sqlx.SqlConn) UserIdentityModel {
	return &customUserIdentityModel{
		defaultUserIdentityModel: newUserIdentityModel(conn),
	}
}

func (m *customUserIdentityModel) withSession(session sqlx.Session) UserIdentityModel {
	return NewUserIdentityModel(sqlx.NewSqlConnFromSession(session))
}

// 根据身份标识符和身份类型查询用户身份
func (m *customUserIdentityModel) FindByIdentifierAndIdentityType(ctx context.Context, identifier string, identityType int64) (*UserIdentity, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE identifier = ? AND identity_type = ?", m.table)
	var resp UserIdentity
	err := m.conn.QueryRowCtx(ctx, &resp, query, identifier, identityType)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, errors.Join(err, fmt.Errorf("find user identity by identifier %s and identity type %d failed", identifier, identityType))
	}
	return &resp, nil
}

// 注册用户身份
func (m *customUserIdentityModel) RegisterUserIdentity(ctx context.Context, tx sqlx.Session, identity *UserIdentity) error {
	var conn sqlx.Session
	if tx == nil {
		conn = m.conn
	} else {
		conn = tx
	}
	_, err := conn.ExecCtx(ctx, "insert into user_identity (user_uuid, identity_type, identifier, credential) values (?, ?, ?, ?)", identity.UserUuid, identity.IdentityType, identity.Identifier, identity.Credential)
	return err
}
