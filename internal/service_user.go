package internal

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"test-api/internal/db"
	dbc "test-api/internal/pkg/psqlservice"
	"time"
)

type UserService struct {
    Users *db.Repository[*User]
    Roles *db.Repository[*Role]
}

// Constructor
func GetUserService() *UserService {
    pool := GetEnv().GetConnPool()
    return &UserService{
        Users: NewUserRepository(dbc.GetPsqlService[User](pool)),
        Roles: NewRoleRepository(dbc.GetPsqlService[Role](pool)),
    }
}

// User by `id`
func (us *UserService) Find(ctx context.Context, id int) (*User, bool) {
    u, ok := us.Users.Find(ctx, id)
    if ! ok {
        return u, false
    }
    if err := us.OrganizeUserEntity(ctx, u, false); err != nil {
        return u, false
    }
    return u, true
}
// list users
func (us *UserService) List(ctx context.Context, limit int, offset int) ([]*User, error) {
    uList, err := us.Users.FindAll(ctx, limit, offset, "id", db.ORDER_ASC)
    if err != nil {
        log.Println(err)
        return nil, ErrorFactory(ERR_GET_LIST, "User")
    }
    // get with roles
    for _, u := range uList {
        us.OrganizeUserEntity(ctx, u, false)
    }
    return uList, nil
}
// creates User entity
func (us *UserService) Create(
    ctx context.Context,
    fName string,
    lName string,
    email string,
    passphrase string,
    nickname string,
) (*User, error) {    
    // check (just for test)
    if nickname == "" {
        return &User{}, ErrorFactory(ERR_EMPTY_PARAM, "nickname")
    }
    // todo: check email
    validatedEmail 	:= email
    // todo: check passphrace (is it ok with our policy?)
    validatedPhrase	:= passphrase
    cTime := time.Now().Unix() // int64
    u := User{
        FirstName: 	fName,
        LastName: 	lName,
        Email: 		validatedEmail,
        PassPhrase: validatedPhrase,
        NickName: 	nickname,
        IsActive: 	true, // account can be active (it means not banned) but still not verified
        CreatedAt: 	cTime,
        UpdatedAt: 	cTime,
    }
    // DO NOT VERIFY AT THE MOMENT BUT FOR NOW
    u.IsVerified = true // todo: remove it and add verification feature
    // try create user
    err := us.Users.Save(ctx, &u)
    if err != nil {
        // we don't need revert at this moment
        return &User{}, ErrorFactory(ERR_ENTITY_SAVE, "User")
    }
    revert := false
    err = us.SetRoles(ctx, &u, "assign", []string{USER_ROLE_TRUSTED, USER_ROLE_COMMON})
    if err != nil {
        // we have to revet but...
        revert = true
        // first... log reason of rever 
        log.Println(err)
        log.Println(ErrorFactory(ERR_ASSIGN_ROLES, u.GetID()))
    }
    if revert {
        revertErr := us.revertOnCreateFailure(ctx, &u)
        if revertErr != nil {
            log.Println(revertErr)
            return &User{}, ErrorFactory(ERR_REVERT_AFTER_CREATE)
        }
    }
    return &u, nil
}
// base update of user's personal info
func (us *UserService) UpdatePersonalInfo(
    ctx context.Context,
    u *User,
    incomingData map[string]IncomingValue,
) (*User, error) {
    isModified := false
    // for `common` user
    for k, v := range incomingData {
        switch k {
        case "first_name":
            if ApplyStrV(&u.FirstName, v) { isModified = true }
        case "last_name":
            if ApplyStrV(&u.LastName, v) { isModified = true }
        case "nickname":
            if ApplyStrV(&u.NickName, v) { isModified = true }
        }
    }
    // for admin only
    if admin, ok := GetAuthService().GetLoggedUserPtr(ctx); ok {
        // check privileges (roles)
        if us.IsPrivileged(ctx, admin) {
            for k, v := range incomingData {
            switch k {
            case "email":
                if ApplyStrV(&u.Email, v) { isModified = true }
            case "phrase":
                if ApplyStrV(&u.PassPhrase, v) { isModified = true }
            }
            }
        }
    }
    // if not modified....
    if (! isModified) {
        // nothing to do
        return u, nil
    }
    // otherwise update
    return us.Update(ctx, u)
}
// `general` update on User entity
func (us *UserService) Update(ctx context.Context, u *User) (*User, error) {
    if u.ID < 1 {
        return u, ErrorFactory(ERR_ENTITY_UNSAVED, "User")        
    }    
    // small lifting before save
    u.UpdatedAt = time.Now().Unix() // fix update time (todo: only if changed)
    // persist due to strategy then save
    err := us.Users.Persist(u).Flush(ctx)
    if err != nil {
        log.Println(fmt.Errorf("Couldn't update this user %v. Reason:", u.ID))
        log.Println(err)
        return u, ErrorFactory(ERR_ENTITY_UDPATE, "User")
    }
    return u, nil
}
// removes User entity
func (us *UserService) Remove(ctx context.Context, u *User) error {
    uID := u.GetID()
    if uID == 0 {
        return ErrorFactory(ERR_ENTITY_UNSAVED, "User")        
    }
    pool := GetEnv().GetConnPool()
    err := dbc.RunInTx(ctx, pool, func(contextWithTx context.Context) error {        
        _, err := us.Users.Persist(u).Delete(contextWithTx)
        if err != nil {
            log.Println(fmt.Errorf("Unable to remove USER entity. Reason: %s", err))
            return ErrorFactory(ERR_ENTITY_REMOVE, "User")            
        }
        // should be done by CASCADE action ON DELETE, but justo for sure...
        _, err = dbc.DoExecRawQuery[*User2Role](
            contextWithTx,
            pool,
            "DELETE FROM user2roles WHERE user_id = $1",
            uID,
        )
        if err != nil {
            log.Println(fmt.Errorf("Unable to remove roles related to user [%d]. Reason: %s", uID, err))
            return ErrorFactory(ERR_REVOKE_ROLES, strconv.Itoa(uID))
        }
        return nil
    })
    return err
}
// assigns roles to user
func (us *UserService) AssignRoles(ctx context.Context, u *User, names []string) (error) {    
    args := make([]any, 0)
    q := "INSERT INTO public.user2roles (user_id,role_id,created_at,updated_at) VALUES ($1,$2,$3,$4)"

    roles, err := us.Roles.FindAll(ctx, 0, 0, "id", db.ORDER_ASC)
    if err != nil {
        return err
    }
    uid     := u.GetID()
    pool    := GetEnv().GetConnPool()
    cTime   := time.Now().Unix()
    err = dbc.RunInTx(ctx, pool, func(contextWithTx context.Context) error {
        for _, r := range roles {
            for _, passedName := range names {
                if passedName == r.Name {
                    args = []any{uid, r.GetID(), cTime, cTime}
                    _ , err := dbc.DoExecRawQuery[*User2Role](contextWithTx, pool, q, args...)
                    if err != nil {
                        return fmt.Errorf("Unable to assign specified roles.")
                    }
                }
            }
        }
        return nil
    })
    return err
}
// assigns / revokes roles to user 
func (us *UserService) SetRoles(ctx context.Context, u *User, action string, names []string) (error) {
    args    := make([]any, 0)
    qIns    := "INSERT INTO public.user2roles (user_id,role_id,created_at,updated_at) VALUES ($1,$2,$3,$4)"
    qDel    := "DELETE FROM public.user2roles WHERE user_id = $1 AND role_id = $2"
    query   := ""

    roles, err := us.Roles.FindAll(ctx, 0, 0, "id", db.ORDER_ASC)
    if err != nil {
        return err
    }
    uid     := u.GetID()
    pool    := GetEnv().GetConnPool()
    cTime   := time.Now().Unix()
    err = dbc.RunInTx(ctx, pool, func(contextWithTx context.Context) error {
        for _, r := range roles {
            for _, passedName := range names {
                if passedName == r.Name {
                    switch action {
                    case "assign":
                        query = qIns
                        args = []any{uid, r.GetID(), cTime, cTime}
                    case "revoke":
                        if (len(u.Roles) < 1) {
                            us.FetchRoles(ctx, u)
                        }
                        _, exists := u.Roles[r.Name]
                        if !exists {
                            // nothing to do so go to next role
                            continue
                        }
                        query = qDel
                        args = []any{uid, r.GetID()}
                    default:
                        return fmt.Errorf("Invalid action for `setRoles`.")
                    }
                    // execute query
                    _ , err := dbc.DoExecRawQuery[*User2Role](contextWithTx, pool, query, args...)
                    if err != nil {
                        return fmt.Errorf("Unable to assign specified roles.")
                    }
                }
            }
        }
        return nil
    })
    return err
}
// assign only (roles)
func (us *UserService) AssignRole(ctx context.Context, name string, id int, u *User) (error) {	
    var whereStr string
    var param any
    // for assign by ID
    if name != "" {
        whereStr, param = "WHERE name = $1", name
    // for assign by ROLE name
    } else if id <= 0 {
        whereStr, param = "WHERE id = $1", strconv.Itoa(id)
    // can't determine
    } else {
        return fmt.Errorf("Invalid params passed (id or role name is required).")
    }
    // get connection pool
    pool := GetEnv().GetConnPool()
    // get role
    PT, err := dbc.DoRawQueryGetStruct[Role](ctx, pool, "SELECT * FROM public.roles " + whereStr + " LIMIT 1", param)
    if err != nil || len(PT) != 1 {
        return fmt.Errorf("Unable to fetch role defined as: %v", param)
    }
    r 		:= PT[len(PT)-1]
    // prepare args
    cTime   := time.Now().Unix()
    args 	:= []any{u.ID, r.ID, cTime, cTime}
    // try assign
    _ , err = dbc.DoExecRawQuery[dbc.IEntity](
        ctx,
        pool,
        "INSERT INTO public.user2roles (user_id, role_id, created_at, updated_at) VALUES ($1,$2,$3,$4)",
        args...,
    )
    
    if err != nil {
        return fmt.Errorf("Unable to assign specified role to user.") // todo: params
    }
    return nil
}
// TODO: LAX
// gets User's roles from DB
func (*UserService) getUserRoles(ctx context.Context, u *User) ([]*Role, error) {
    // get connection pool
    pool := GetEnv().GetConnPool()
    id := u.ID
    if id <= 0 {
        return nil, ErrorFactory(ERR_ENTITY_UNSAVED, "User")        
    }
    sqlStr := `
    SELECT r.* FROM roles AS r
    JOIN user2roles AS u2r ON r.id = u2r.role_id
    WHERE u2r.user_id = $1;
    `
    roles, err := dbc.DoRawQueryGetStruct[Role](ctx, pool, sqlStr, id)
    if err != nil {
        return nil, ErrorFactory(ERR_FETCH_ROLES, strconv.Itoa(id))
    }
    return roles, nil
}
// gets user's roles from DB and assigns to user struct (Roles field)
func (us *UserService) FetchRoles(ctx context.Context, u *User) error {
    roles, err := us.getUserRoles(ctx, u)
    if err != nil {
        return err
    }
    rolesMap := make(map[string]*Role)
    for _ , role := range roles {
        rolesMap[role.Name] = role
    }
    u.Roles = rolesMap
    return nil
}
// returs user's roles as map (RoleName => Role)
func (us *UserService) GetRolesFromDb(ctx context.Context) (map[string]*Role, error) {
    roles, err := us.Roles.FindAll(ctx, 0, 0, "id", db.ORDER_ASC)
    if err != nil {
        return nil, fmt.Errorf("Unable to get roles from DB [%w]", err)
    }
    
    result := make(map[string]*Role, len(roles))
    for _,role := range roles {
        result[role.Name] = role
    }
    return result, nil
}
// reverts "create user" action
func (us *UserService) revertOnCreateFailure(ctx context.Context, u *User) error {
    // todo: consider transaction
    errMsg := "Unable to revert CREATE action! User data is inconsistent!"
    pool := GetEnv().GetConnPool()
    id := u.GetID()
    if id < 1	{
        // looks like user doesn't save so NO "purge" action is required
        return nil
    }    
    _, err  := us.Users.Persist(u).Delete(ctx)
    if err != nil {
        return fmt.Errorf(errMsg + " Unable to remove USER entity: [%v]", err)
    }
    // also...    
    _, err = dbc.DoExecRawQuery[*User2Role](ctx, pool, "DELETE FROM user2roles WHERE user_id = $1", id)
    if err != nil {
        return fmt.Errorf(errMsg + " Can't remove ROLE(s) assoc.: [%v]", err)
    }
    return nil
}
// region helpers
func (us *UserService) OrganizeUserEntity(ctx context.Context, u *User, withToken bool) error {
    if u.GetID() == 0  {
        // just leave user as it is
        return nil
    }
    if u.Roles == nil  {
        err := us.FetchRoles(ctx, u)
        if err != nil {
            // forward
            return err
        }
    }
    // ... w. token if required
    if withToken && u.TokenData == nil {        
        GetAuthService().FetchUserToken(ctx, u, true)
    }
    return nil
}
// checks if user has extra roles / privileges (admin/system)
func (us *UserService) IsPrivileged(ctx context.Context, u *User) bool {
    return us.checkPrivileges(ctx, u, USER_ROLE_PRIVILEGED)
}
// user is admin ?
func (us *UserService) IsAdmin(ctx context.Context, u *User) bool {
    return us.checkPrivileges(ctx, u, USER_ROLE_ADMIN)
}
// user is system (bot for external and automated internal operations)
func (us *UserService) IsSystem(ctx context.Context, u *User) bool {
    return us.checkPrivileges(ctx, u, USER_ROLE_SYSTEM)
}
// helper to determine user extra roles
func (us *UserService) checkPrivileges(ctx context.Context, u *User, expected string) bool {
    if u == nil {
        // no logged (or passed) user
        // it can happen during external operation (like ext. massive data import)
        // so it's not redundant check at all 
        return false
    }
    if u.GetID() == 0 {
        // unsaved user can't be an admin, system-bot (dummy) etc.
        return false
    }
    err := us.OrganizeUserEntity(ctx, u, false)
    if err != nil {
        // just return false (for security reasons)
        return false
    }
    for _, roleInfo := range u.Roles {
        // for admin 
        if expected == USER_ROLE_ADMIN && roleInfo.IsAdmin == true {
            return true
        }
        // for system
        if expected == USER_ROLE_SYSTEM && roleInfo.IsSystem == true {
            return true
        }
        // for one of them (privileged user)
        if expected == USER_ROLE_PRIVILEGED {
            return roleInfo.IsAdmin || roleInfo.IsSystem
        }
    }
    return false
}
// endregion

