package internal

import (
    "context"
    "crypto/rand"
    "encoding/hex"
    "fmt"
    "log"
    "strconv"
    "test-api/internal/db"
    dbc "test-api/internal/pkg/psqlservice"
    "time"
)

const DEF_SESSION_TOKEN_LEN = 32

type AuthService struct {
    Tokens 			*db.Repository[*UserToken]
    Users			*db.Repository[*User]
    TokenLifeTime	uint16
    tokenLen        uint8
}

// builds service
func GetAuthService() *AuthService {
    env                 := GetEnv()    
    return &AuthService{
        Tokens:			NewTokenRepository(dbc.GetPsqlService[UserToken](GetEnv().GetConnPool())),
        Users:			GetUserService().Users,
        TokenLifeTime:	env.userSessionTime,
        tokenLen:       getTokenLenFromConf(),
    }
}

func getTokenLenFromConf() uint8 {
    conf        := GetEnv().GetConfig()
    tokenLen    := conf.General["token_len"]
    if tokenLen == "" {tokenLen = "0"}
    v, err := ParseUint8(tokenLen)
    if err != nil || v == 0 {
        v = DEF_SESSION_TOKEN_LEN
    }
    return v
}
// try gets logged user as pointer, nil on failure (and boolean false)
func (as *AuthService) GetLoggedUserPtr(ctx context.Context) (*User, bool) {
    // first, try find in current context
    u, _ 		:= ctx.Value(userEntityPtrKey{}).(*User) // pointer to User entity or nil
    tokenStr, _ := ctx.Value(clientTokenKey{}).(string) // token string if exists or empty string
    // check ID
    if u == nil {
        // if it fails, will return pointer to empty struct
        u, _ = as.GetUserByTokenValue(ctx, tokenStr)
    }
    // so must to check this...
    if u.GetID() == 0 {
        // no logged user
        return nil, false
    }
    // rebuild user entity with all related entities (if ther're not already exist)
    GetUserService().OrganizeUserEntity(ctx, u, true)
    // then check token
    if !as.CheckUserToken(ctx, u.TokenData) {
        // looks like "logged" user is no logged anymore, so...
        return nil, false
    }
    // looks like we've got one
    return u, true
}
// gets system user or empty entity if it fails
func (as *AuthService) GetSystemUser(ctx context.Context) (*User, bool) {
    // fetch user service
    us := GetUserService()
    // already loaded? Check globals...
    if SystemUser != nil {
        // small check
        if us.IsSystem(ctx, SystemUser) { return SystemUser, true }        
    }
    // otherwise we have to do it via `standard` way
    sqlStr := `
    SELECT u.* FROM public.users AS u 
    JOIN public.user2roles AS u2r ON u.id = u2r.user_id 
    WHERE u2r.id IN (SELECT id FROM public.roles WHERE is_system = true)
    LIMIT 1 OFFSET 0`
    // do query
    users, err := dbc.DoRawQueryGetStruct[User](ctx, GetEnv().GetConnPool(), sqlStr)
    if err != nil || len(users) != 1 {
        log.Printf("SQL Error while trying to fetch system user entity: [%v]\n", err)
        return &User{}, false
    }
    u := users[len(users)-1]    
    if ! us.IsPrivileged(ctx, u) {
        log.Printf("Unable to fetch system user.")
        return &User{}, false
    }
    us.OrganizeUserEntity(ctx, u, false)
    // set in global scope (same bot for all! ;) ) 
    SystemUser = u
    return u, true
}
// try to get logged user, if fails fetches system user instead (if required)
func (as *AuthService) GetLoggedOrSystemUser(ctx context.Context, forceSystem bool) (*User, error) {
    u, ok := as.GetLoggedUserPtr(ctx)
    if !ok && forceSystem {
        u, ok = as.GetSystemUser(ctx)
        if !ok {
            return &User{}, ErrorFactory(ERR_GET_SYSTEM_USER)
        }
        return u, nil
    }
    if !ok {
        return &User{}, ErrorFactory(ERR_CANT_FETCH_LOGGED_USER)
    }
    return u, nil
}
// checks user token
func (as *AuthService) CheckUserToken(ctx context.Context, ut *UserToken) bool {
    // for now only time
    if ut.ValidTo > time.Now().Unix() {
        return true
    }
    // todo: check value from request against token
    return false
}

func (as *AuthService) TryLoginViaMail(ctx context.Context, email string, phrase string) (*User, error) {
    // todo: sanitize email	
    // find user using `search criteria module`
    var criteria = map[string]map[string]string {
        "where": {"email =": email, "is_active =": "1"},
    }
    results, err := as.Users.FindAllBy(ctx, criteria, 1, 0)
    if err != nil {        
        log.Println(fmt.Errorf("[SQL ERROR] [%w]", err))
        return &User{}, ErrorFactory(ERR_ENTITY_NOT_FOUND, "User")
    }
    if len(results) != 1 {
        // not found or ambiguous
        return &User{}, ErrorFactory(ERR_ENTITY_NOT_FOUND, "User")
    }
    // bingo !
    u 		:= results[len(results)-1]
    u, _ 	= as.FetchUserToken(ctx, u, false)
    // otherwise try login
    return as.TryLoginUser(ctx, u)
}

func (as *AuthService) TryLoginUser(ctx context.Context, u *User) (*User, error) {	
    // check user Token first
    if u.TokenData == nil {
        as.FetchUserToken(ctx, u, false)
    }
    currTokenExp, nowUnix := u.TokenData.ValidTo, time.Now().Unix()
    // check current session
    if nowUnix - currTokenExp < 0 {
        // session is valid, nothing to do
        GetUserService().OrganizeUserEntity(ctx, u, false)
        return u, nil		
    }
    // since now we must start tx
    err := dbc.RunInTx(ctx, GetEnv().GetConnPool(), func(contextWithTx context.Context) error {
        // small clean-up
        err := as.removeOutdatedTokens(contextWithTx)
        if err != nil {
            // just warn and log but not return from func
            log.Printf("WARNING: Couldn't remove outdated tokens before loging process. Reason: %s", err)
        }
        newTokenVal, newTokenExp, err := as.GenerateTokenValueHex(int(as.tokenLen), int(as.TokenLifeTime))
        // build new UserToken entity
        ut := &UserToken{
            UserID: 	u.GetID(),
            Value: 		newTokenVal,
            ValidTo: 	newTokenExp,
            CreatedAt: 	nowUnix,
            UpdatedAt: 	nowUnix,
        }
        // try save
        if err = as.Tokens.Save(contextWithTx, ut); err != nil {
            return ErrorFactory(ERR_ENTITY_SAVE, "UserToken")
        }
        // if success
        // attach token data
        u.TokenData = ut
        // also roles
        err 		= GetUserService().FetchRoles(contextWithTx, u)
        if err != nil {
            return ErrorFactory(ERR_FETCH_ROLES, u.Email)			
        }
        // small update on user entity
        u.LastSeen 	= nowUnix
        u.UpdatedAt = nowUnix
        if err = as.Users.Save(contextWithTx, u); err != nil {
            log.Println("Can't update user entity during login process... [%w]", err)
            return ErrorFactory(ERR_ENTITY_SAVE, "User")
        }
        return nil
    })

    if err != nil {
        return &User{}, err
    }
    // finally success !
    return u, nil
}

func (as *AuthService) Logout(ctx context.Context) error {
    // try fetch logged user in `standard` way
    u, ok := as.GetLoggedUserPtr(ctx)
    if !ok {
        return ErrorFactory(ERR_NOT_LOGGED_IN)
    }
    aRows, err := as.Tokens.ExecQuery(
        ctx,
        "DELETE FROM " + as.Tokens.TableName + " WHERE user_id = $1", u.ID,
    )
    // check for db errors
    if err != nil {
        log.Printf("Unable to remove user tokens: [%v] \n", err)
        return ErrorFactory(ERR_CANT_REMOVE_USER_TOKENS)
    }
    // one more check...
    if aRows < 1 {
        log.Printf("No active user's tokens detected for user ID: [%v] \n", u.ID)
        return ErrorFactory(ERR_NOT_LOGGED_IN)
    }
    // looks ok
    return nil
}

// creates new UserToken entity and saves it into DB
func (as *AuthService) CreateUserToken(ctx context.Context, u *User, hexVal string, lf int64) error {
    // get related user ID
    uID := u.GetID()
    if (uID == 0) {
        return fmt.Errorf("Can't create TOKEN for unsaved (w/o ID) USER entity.")
    }
    // get current time
    now := time.Now().Unix()
    // prepare token entity
    var token = &UserToken{
        UserID: 	uID,
        Value: 		hexVal,
        ValidTo: 	lf,
        CreatedAt: 	now,
        UpdatedAt:  now,
    }
    // then try to save
    if err := as.Tokens.Save(ctx, token); err != nil {
        log.Println("Unable to save user's TOKEN into db. Something went wrong. [%w]", err)
        return ErrorFactory(ERR_ENTITY_SAVE, "UserToken")
    }
    return nil
}

// gets (valid) User's token value
func (as *AuthService) GetUserByTokenValue(ctx context.Context, v string) (*User, bool) {
    // todo: refactor to one query User + token and map on structs (JSONB)
    // for now... must be like below:
    if v == "" {
        return &User{}, false
    }
    sqlStr := `
    SELECT u.* FROM public.users AS u
    JOIN public.token AS t ON u.id = t.user_id
    WHERE t.value = $1 AND t.valid_to >= $2`
    // do query
    users, err := dbc.DoRawQueryGetStruct[User](
        ctx, GetEnv().GetConnPool(),
        sqlStr,
        []any{v,time.Now().Unix()}...,
    )
    if err != nil {
        log.Printf("SQL Error while trying to fetch user via token value: [%v]\n", err)
        return &User{}, false
    }

    // return only valid users
    if len(users) != 1 {
        return &User{}, false
    }
    u := users[len(users)-1]
    // assign token entity 
    as.FetchUserToken(ctx, u, false)
    // then roles
    err = GetUserService().FetchRoles(ctx, u)
    if err != nil {
        return u, false
    }
    // add to `cache` map (by token)
    return u, true
}

// fetches and assign User's token to User entity
func (as *AuthService) FetchUserToken(ctx context.Context, u *User, force bool) (*User, bool) {
    userId 	:= u.GetID()
    if userId == 0 {
        u.TokenData = &UserToken{}; return u, false
    }
    if u.TokenData != nil {
        tokenId	:= u.TokenData.GetID()
        if ! force && tokenId != 0 {
            // already loaded
            return u, true
        }
    }
    // set search criteria
    var criteria = map[string]map[string]string {
        "where": {
            "user_id =": strconv.Itoa(userId),            
        },
    }
    // find all user's tokens
    tokens, err := as.Tokens.FindAllBy(ctx, criteria, 1, 0)
    if err != nil || len(tokens) != 1 {
        // todo msg for invalid
        // ... and assingn empty
        u.TokenData = &UserToken{}
        return u, false
    }
    // in any other case
    u.TokenData = tokens[len(tokens)-1]
    // then
    return u, true
}

// creates new User's token
func (as *AuthService) GenerateTokenValueHex(len int, lifeTime int) (string, int64, error) {
    // small protection
    // for token len...
    if len < 8 {
        len = 8
    }
    // also for token `lifetime`
    if lifeTime < 1 {
        lifeTime = 60 // minute
    }
    // now we are ready
    bytes := make([]byte, len)
    if _, err := rand.Read(bytes); err != nil {
        return "", 0, fmt.Errorf("Crypto rand failed. Unable to create Token. [%w]", err)
    }

    return hex.EncodeToString(bytes), time.Now().Unix() + int64(lifeTime), nil
}

// removes all outdated tokens
func (as *AuthService) removeOutdatedTokens(ctx context.Context) error {
    // try exec query
    _, err := as.Tokens.ExecQuery(
        ctx,
        "DELETE FROM " + as.Tokens.TableName + " WHERE valid_to <= " + strconv.FormatInt(time.Now().Unix(), 10),
    )
    // check for errors
    if err != nil {
        return fmt.Errorf("Unable to remove outdated tokens: [%v]", err)
    }
    return nil
}