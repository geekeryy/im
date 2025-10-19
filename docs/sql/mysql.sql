-- 会话表
create table sessions (
    id bigint auto_increment, -- 主键ID
    uuid varchar(255) not null, -- 会话UUID
    name varchar(255) not null default '', -- 会话名称
    avatar varchar(255) not null default '', -- 会话头像
    session_type int not null, -- 会话类型 1: 单聊 2: 群聊
    status int not null, -- 状态
    created_at datetime default current_timestamp not null, -- 创建时间
    updated_at datetime default current_timestamp on update current_timestamp not null, -- 更新时间
    primary key (id) -- 主键ID
);
create unique index idx_sessions_uuid on sessions (uuid);

-- 会话成员表
create table session_members (
    id bigint auto_increment, -- 主键ID
    session_uuid varchar(255) not null, -- 会话UUID
    user_uuid varchar(255) not null, -- 用户UUID
    created_at datetime default current_timestamp not null, -- 创建时间
    updated_at datetime default current_timestamp on update current_timestamp not null, -- 更新时间
    primary key (id) -- 主键ID
);
create unique index idx_session_members_user_uuid_session_uuid on session_members (user_uuid, session_uuid);
create index idx_session_members_session_uuid on session_members (session_uuid);


-- 消息表
create table messages (
    id bigint auto_increment, -- 主键ID
    uuid varchar(255) not null, -- 消息UUID
    session_uuid varchar(255) not null, -- 会话UUID
    sender_uuid varchar(255) not null, -- 发送者UUID
    seq_id bigint not null, -- 消息序列号ID
    message_type int not null, -- 消息类型 1: text
    status int not null, -- 状态 1: 已发送 2: 已接收 3: 已读
    content text not null, -- 消息内容
    created_at datetime default current_timestamp not null, -- 创建时间
    updated_at datetime default current_timestamp on update current_timestamp not null, -- 更新时间
    primary key(id) -- 主键ID
);
create unique index idx_messages_uuid on messages (uuid);

-- 用户表
create table user_base (
    id bigint auto_increment,
    uuid varchar(255) not null, -- 用户UUID
    name varchar(255) not null, -- 用户名
    avatar varchar(255) not null, -- 用户头像
    status int not null, -- 状态
    created_at datetime default current_timestamp not null, -- 创建时间
    updated_at datetime default current_timestamp on update current_timestamp not null, -- 更新时间
    primary key (id) -- 主键ID
);
create unique index idx_user_base_uuid on user_base (uuid);

-- 用户信息表
create table user_info (
    id bigint auto_increment,
    uuid varchar(255) not null, -- 用户UUID
    gender int not null, -- 1: male, 2: female, 3: other
    mobile varchar(255) not null, -- 手机号
    email varchar(255) not null, -- 邮箱
    created_at datetime default current_timestamp not null, -- 创建时间
    updated_at datetime default current_timestamp on update current_timestamp not null, -- 更新时间
    primary key (id) -- 主键ID
);
create unique index idx_user_info_uuid on user_info (uuid);

-- 用户身份表
create table user_identity (
    id bigint auto_increment,
    user_uuid varchar(255) not null, -- 用户UUID
    identity_type      int not null,                     -- 身份类型 1: 手机号 2: 邮箱 3: 用户名 4: wechat 5: google 6: facebook 7: github
    identifier         varchar(255) not null,            -- 标识符 手机号/邮箱/用户名/google_id/facebook_id/github_id
    credential         varchar(255) not null default '', -- 凭证 密码
    created_at datetime default current_timestamp not null, -- 创建时间
    updated_at datetime default current_timestamp on update current_timestamp not null, -- 更新时间
    primary key (id) -- 主键ID
);
create unique index idx_user_identity_user_uuid_identity_type on user_identity (user_uuid,identity_type);