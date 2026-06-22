package internal

import (
	"encoding/json"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
    // admin and system bot
    USER_ROLE_SYSTEM string     = "USER_ROLE_SYSTEM"
    USER_ROLE_ADMIN string      = "USER_ROLE_ADMIN"
    // `common` users
    USER_ROLE_TRUSTED string    = "USER_ROLE_TRUSTED"
    USER_ROLE_COMMON string     = "USER_ROLE_COMMON"
    USER_ROLE_GUEST string      = "USER_ROLE_GUEST"
    // ...also `virtual multi-role`` for admin and / or system
    USER_ROLE_PRIVILEGED string = "USER_ROLE_PRIVILEGED"

    PARAM_TYPE_NUMBER string    = "PARAM_TYPE_NUMBER"
    PARAM_TYPE_BOOL string      = "PARAM_TYPE_BOOL"
    PARAM_TYPE_STRING string    = "PARAM_TYPE_STRING"
)
// general
type (
    ApiEnvironment struct {
        connectionPool      *pgxpool.Pool
        router              *chi.Mux
        logFile             *os.File
        userSessionTime     uint16
        loadedConfig        ApiConfig
    }

    ApiConfig struct {
	    General  	map[string]string     `json:"general"`
	    Db			map[string]string     `json:"db"`
    }
)
// user related
type (
	User struct {
        ID          int                 `db:"id"`
        FirstName   string              `db:"first_name"`
        LastName    string              `db:"last_name"`        
        Name 		string              `db:"-"`        // todo REMOVE
        Email       string              `db:"email"`
        NickName    string              `db:"nickname"`
        PassPhrase  string              `db:"passphrase"`
        LastSeen    int64               `db:"last_seen"`
        IsVerified  bool                `db:"is_verified"`
        IsActive    bool                `db:"is_active"`
        CreatedAt   int64               `db:"created_at"`
        UpdatedAt   int64               `db:"updated_at"`        
        Roles       map[string]*Role    `db:"-"`
        TokenData   *UserToken          `db:"-"`
	}
    // user session token
	UserToken struct {
        ID          int                 `db:"id"`
        UserID      int                 `db:"user_id"`
		Value 		string              `db:"value"`
		ValidTo 	int64               `db:"valid_to"`
        CreatedAt   int64               `db:"created_at"`
        UpdatedAt   int64               `db:"updated_at"`        
	}
    // user's role in the system
	Role struct {
        ID          int                 `db:"id"`
        Name        string              `db:"name"`
        Desc        string              `db:"description"`
        IsAdmin	    bool                `db:"is_admin"`
        IsSystem    bool                `db:"is_system"`
        Trusted     bool                `db:"-"`
        CanLogin    bool                `db:"can_login"`
        CreatedAt   int64               `db:"created_at"`
        UpdatedAt   int64               `db:"updated_at"`
	}
    // relation user -> roles
    User2Role struct {
        ID          int                 `db:"id"`
        UserID      int                 `db:"user_id"`
        RoleID      int                 `db:"role_id"`
        CreatedAt   int64               `db:"created_at"`
        UpdatedAt   int64               `db:"updated_at"`
    }
)
// router
type (
    // custom handler
    CustomHandler func(rw http.ResponseWriter, r *http.Request) error
    // Route spec.
    Route struct {
        Name            string                      `json:"name"`
        Path            string                      `json:"path"`
        Method          string                      `json:"method"`
        Restricted      bool                        `json:"restricted"`
        RequiredRoles   []string                    `json:"required_roles"`
        Handler         CustomHandler               `json:"handler"`
    }
    // Incoming param DTO
    IncomingParam struct {
        Name        string                          `json:"name"`
        Type        string                          `json:"type"`
        IsRequired  bool                            `json:"is_required"`
        UrlOnly     bool                            `json:"url_only"`
        CanBeEmpty  bool                            `json:"can_be_empty"`
        Default     string                          `json:"default"`
    }
    // DTO for Route and params (maybe it's little bit redundant /JSON-conf. dedicated/)
    ParamDTO struct {
	    Name       string                           `json:"name"`
	    Type       string                           `json:"type"`
	    IsRequired bool                             `json:"is_required"`
	    UrlOnly    bool                             `json:"url_only"`
	    CanBeEmpty bool                             `json:"can_be_empty"`
	    Default    string                           `json:"default"`
    }
    RouteDTO struct {
	    Name          string                        `json:"name"`
	    Path          string                        `json:"path"`
	    Method        string                        `json:"method"`
	    Restricted    bool                          `json:"restricted"`
	    RequiredRoles []string                      `json:"required_roles"`
	    Handler       string                        `json:"handler"`
	    Params        []ParamDTO                    `json:"params"` // nested
    }
)
// response DTO
type (
    GeneralResponse struct {
        Status              string                  `json:"status"`
        Code                int                     `json:"code"`    
        Body                json.RawMessage         `json:"body,omitempty"`
        ErrorOccured        bool                    `json:"error_occurred"`
        IErrorCode          int                     `json:"i_error_code"`    
        Msg                 string                  `json:"msg,omitempty"`
    }
    ResponseErrorInfo struct {
        HttpCode            int
        InternalCode        int
        Message             string
        DebugData           []string
    }
)