package internal

import (	
    // "test-api/internal/pkg/psqlservice"
    db "test-api/internal/db"
    dbc "test-api/internal/pkg/psqlservice"

)

// ===============
// USER REPOSITORY
// ===============

// prepare interface IEntity and data model
func (u *User) SetID(id int) {	
    u.ID = id
}

func (u *User) GetStruct() any {
    return u
}


func (u *User) GetID() int {
    return u.ID
}

func (u *User) GetTableName() string {
    return "public.users"
}

func (u *User) GetColumns() []string {
    return []string{
        "first_name",
        "last_name",
        "email",
        "nickname",
        "passphrase",
        "last_seen",
        "is_verified",
        "is_active",
        "created_at",
        "updated_at",
    }
}

func (u *User) GetValues() []any {
    return []any{
        u.FirstName,
        u.LastName,
        u.Email,
        u.NickName,
        u.PassPhrase,
        u.LastSeen,
        u.IsVerified,
        u.IsActive,
        u.CreatedAt,
        u.UpdatedAt,
    }	
}

func (p *User) GetKeyValuePair() map[string]any {
    // TODO: write helper to build KeyValuePair
    return map[string]any {}
}

func NewUserRepository[PT dbc.IEntity] (driver dbc.IDatabaseDriver[PT]) *db.Repository[PT] {
    return db.NewRepository(driver)
}

// ===============
// ROLE REPOSITORY
// ==============

func (role *Role) SetID(id int) {	
    role.ID = id
}

func (role *Role) GetStruct() any {
    return role
}


func (role *Role) GetID() int {
    return role.ID
}

func (role *Role) GetTableName() string {
    return "public.roles"
}

func (role *Role) GetColumns() []string {
    return []string{
        "name",
        "description",
        "is_admin",
        "is_system",
        "can_login",
        "created_at",
        "updated_at",
    }
}

func (role *Role) GetValues() []any {
    return []any{
        role.Name,
        role.Desc,
        role.IsAdmin,
        role.IsSystem,
        role.CanLogin,		
        role.CreatedAt,
        role.UpdatedAt,
    }	
}

func (role *Role) GetKeyValuePair() map[string]any {
    return map[string]any {}
}

func NewRoleRepository[PT dbc.IEntity] (driver dbc.IDatabaseDriver[PT]) *db.Repository[PT] {
    return db.NewRepository(driver)
}

// ===============
// ROLE REPOSITORY
// ===============

func (t *UserToken) SetID(id int) {
    t.ID = id
}

func (t *UserToken) GetStruct() any {
    return t
}


func (t *UserToken) GetID() int {
    return t.ID
}

func (t *UserToken) GetTableName() string {
    return "public.token"
}

func (t *UserToken) GetColumns() []string {
    return []string{
        "user_id",
        "value",
        "valid_to",
        "created_at",
        "updated_at",
    }
}

func (t *UserToken) GetValues() []any {
    return []any{
        t.UserID,
        t.Value,
        t.ValidTo,	
        t.CreatedAt,
        t.UpdatedAt,
    }
}

func (t *UserToken) GetKeyValuePair() map[string]any {
    return map[string]any {}
}

func NewTokenRepository[PT dbc.IEntity] (driver dbc.IDatabaseDriver[PT]) *db.Repository[PT] {
    return db.NewRepository(driver)
}

// ===============
// USER 2 ROLE REPOSITORY
// ===============
func (u2r *User2Role) SetID(id int) {
    u2r.ID = id
}

func (u2r *User2Role) GetStruct() any {
    return u2r
}

func (u2r *User2Role) GetID() int {
    return u2r.ID
}

func (u2r *User2Role) GetTableName() string {
    return "public.user2roles"
}

func (u2r *User2Role) GetColumns() []string {
    return []string{
        "user_id",
        "role_id",
        "created_at",
        "updated_at",
    }
}

func (u2r *User2Role) GetValues() []any {
    return []any{
        u2r.UserID,
        u2r.RoleID,
        u2r.CreatedAt,
        u2r.UpdatedAt,
    }
}

func (u2r *User2Role) GetKeyValuePair() map[string]any {
    return map[string]any {}
}

func NewUser2RoleRepository[PT dbc.IEntity] (driver dbc.IDatabaseDriver[PT]) *db.Repository[PT] {
    return db.NewRepository(driver)
}