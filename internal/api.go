package internal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand/v2"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
)

// non-const context keys
type contextKey string
type userEntityPtrKey struct{}
type clientTokenKey struct{}

const (
    // context keys
    requestIDKey contextKey     = "RequestID"
    validatedDataKey contextKey = "validatedData"
    userEntityKey contextKey    = "userEntity"    
    endpointName contextKey     = "endpointName"
    endpointSpec contextKey     = "endpointSpec"
    errorInfoKey contextKey     = "errorInfo"
    // strings
    panicMsg string             = "Abnormal program termination... "
    // paths
    mainLogPath string          = "./logs"
)

// globals
var apiEnv *ApiEnvironment
var UserRoles map[string]*Role
var SystemUser *User

// region core
// init api (ctor)
func InitApi(minMode bool) (*ApiEnvironment, error) {
    // init ApiEnv struct
    apiEnv                      = &ApiEnvironment{}
    // also new context
    ctx                         := context.Background()

    // try load main config for API
    config, err                 := loadApiConfig()
    if err != nil {
        return apiEnv, fmt.Errorf(panicMsg + "Cant load main conf. file. Reason: %w", err)
    }
    // set loaded config
    apiEnv.loadedConfig = config

    // try setup connection
    pool, err                   := setupConnection(ctx)
    if err != nil {
        // critical error (not recoverable at this point)
        return apiEnv, fmt.Errorf(panicMsg + "Unable to establish connection with DB.")
    }
    fmt.Println("Connection with DB established.")
    // set connection pool
    apiEnv.connectionPool   = pool

    // setup (and open) log file 
    logFile, err            := setLogFile("")
    if err != nil {
        return apiEnv, fmt.Errorf(panicMsg + "Can't create main API log file.")
    }
    // setup logger w. logfile stream
    apiEnv.logger           = setupLogger(logFile, "")
    apiEnv.logFile          = logFile
    if minMode {
        // if minimalistic mode then we've got what we need so...
        return apiEnv, nil
    }
    // since now...
    defer CleanExit()
    // try load endpoints configuration from json
    log.Println("Loading route map / endpoints configuration... (file ./conf/routes.json)")
    if !importConfig() {
        errMsg := ErrorFactory(ERR_LOAD_ROUTES_CONF)
        log.Println(errMsg)
        return apiEnv, errMsg
    }
    fmt.Println("Done.")
    // then...
    // set up router
    apiEnv.router           = setupRouter(apiEnv.logger)
    // fetch user roles end set in global scope 
    UserRoles, err          = getUserRoles(ctx)
    if err != nil {
        return apiEnv, fmt.Errorf(panicMsg + "Can't fetch user roles.")
    }
    // def. user session time
    ust, err := ParseUint16(config.General["session_time"])
    if err != nil {
        return apiEnv, fmt.Errorf(panicMsg + "Invalid value for session_time. Check configuration.")
    }
    apiEnv.userSessionTime  = ust
    // return (just for transparency)
    return apiEnv, nil
}
// main func for API-listen-loop
func Listen() bool {
    apiEnv := GetEnv()
    if nil == apiEnv {
        // API is misconfigured
        log.Println("Api is not configured properly. Can't continue...")
        return false
    }
    defer CleanExit()

    // get config
    srvCfg      := apiEnv.loadedConfig.General
    hostname    := srvCfg["api_hostname"]
    port        := srvCfg["api_port"]
    // check host
    if hostname == "" {
        log.Println("Warning, listen host address is NOT set - localhost will be used instead.")
    }    
    // check port
    if port == "" {
        log.Println("Warning, listen port is NOT set - HTTP default will be used instead.")
    }
    // create HTTP server instance
    srv := &http.Server{
        Addr:       hostname + ":" + port,
        Handler:    apiEnv.GetRouter(),
    }
    // then we need system signal channel to catch (KILL, etc)
    sysSig := make(chan os.Signal, 1)
    // notify on CTRL-C (interrupt) or KILL (SIGTERM)
    signal.Notify(sysSig, os.Interrupt, syscall.SIGTERM)

    // time to start server
    log.Println("Trying to start HTTP server for the REST API ...")
    go func() {
        if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
            // exit !
            log.Fatalf("Couldn't start the API server: %v", err)
        }
    }()
    log.Println("Looks ok. Ready to serve!")
    // block sig chan
    <-sysSig // waits for interrupt or SIGTERM
    log.Println(" CTRL-C / KILL Captured. Shutting down... ")

    // set context with timeout for the rest tasks (5 second should be enough)
    sdCtx, cancel := context.WithTimeout(context.Background(), 5 * time.Second)
    // cancel() must be executed so...
    defer cancel()
    // stop listening...
    if err := srv.Shutdown(sdCtx); err != nil {
        log.Fatalf("SERVER SHUTDOWN ERROR: %v", err)
    }

    // looks fine...
    log.Println("Done. See you next time.")
    return true
}
// endregion

// region getters
// API Env getter
func GetEnv() *ApiEnvironment {
    return apiEnv
}
// getter for config 
func (*ApiEnvironment) GetConfig() ApiConfig {
    return GetEnv().loadedConfig
}
// safe getters
func (ae *ApiEnvironment) GetRouter() *chi.Mux {
    return ae.router
}
// connection pool
func (ae *ApiEnvironment) GetConnPool() *pgxpool.Pool {
    return ae.connectionPool
}
// for validated incoming data (by schema_validator) transferred via context
// (very important in handlers)
func GetValidatedDataFromContext(ctx context.Context) map[string]IncomingValue {
    if data, ok := ctx.Value(validatedDataKey).(map[string]IncomingValue); ok {
        return data
    }
    return nil
}
// for endpoint specification described in conf. file
func GetTargetEndpointSpecsFromContext(ctx context.Context) Route {
    if specs, ok := ctx.Value(endpointSpec).(Route); ok {
        return specs
    }
    // empty route
    return Route{}
}
// for User entity (ptr) via ctx (easy way to fetch logged user)
func GetUserPtrFromContext(ctx context.Context) *User {
    if uPtr, ok := ctx.Value(userEntityPtrKey{}).(*User); ok {
        return uPtr
    }
    return &User{}
}
// endregion

// region boot-api-cfg
// main config loader
func loadApiConfig() (ApiConfig, error) {
	conf := ApiConfig{}
	data, err := os.ReadFile("./conf/settings.json")
    if err != nil {
        return conf, fmt.Errorf("Main config file not found. Check your file structure.")
    }
    if err := json.Unmarshal(data, &conf); err != nil {
        return conf, fmt.Errorf("Conf. file parse error: %w", err)
    }
    var reqFields   = []string{}
    var section     = map[string]string{}
    verifySection   := func() (bool, string) {
        for _, field := range reqFields {
            _, exists := section[field]
            if !exists {
                return false, field
            }
        }
        return true, ""
    }
    // section `general`
    reqFields, section = []string{"api_hostname", "api_port", "api_log_pfx", "session_time"}, conf.General
    if ok, missedField := verifySection(); ! ok {
        return conf, fmt.Errorf("Conf. file error. Required setting [%s] is missing (section `general`).", missedField)
    }
    // section `db`
    reqFields, section = []string{"db_user", "db_pass", "db_host", "db_port", "db_name"}, conf.Db
    if ok, missedField := verifySection(); ! ok {
        return conf, fmt.Errorf("Conf. file error. Required setting [%s] is missing (section `db`).", missedField)
    }
    return conf, nil
}
// setup Chi Router
func setupRouter(logger *log.Logger) *chi.Mux {
    r := chi.NewRouter()
    r.Use(middleware.RequestLogger(&middleware.DefaultLogFormatter{
        Logger: logger,
        NoColor: true,
    }))
    // native recoverer
    r.Use(middleware.Recoverer)
    // request reg.
    r.Use(RegisterRequestMiddleware)
    // response interceptor 
    r.Use(CustomResponseMiddleware)
    // checking route against schema
    r.Use(CheckRouteMiddleware)
    // Auth
    r.Use(AuthMiddleware)
    // by def
    r.NotFound(func(w http.ResponseWriter, r *http.Request) {
        // set header
        w.Header().Set("Content-Type", "application/json")
        // write status
        w.WriteHeader(http.StatusNotFound)
        // json data
        errMsg := fmt.Sprintf(`{"error": "%s"}`, ErrorFactory(ERR_RESOURCE_NOT_FOUND).Error())
        w.Write([]byte(errMsg))
    })

    for _, route := range Routes {
        customHandler := ErrorInterceptor(route.Handler)
        r.With(
            SchemaValidator{}.Build(&route, true).Validate(),
        ).Method(route.Method, route.Path, customHandler)
    }
    return r
}
// setup pg connection (using PGX)
func setupConnection(ctx context.Context) (*pgxpool.Pool, error) {    
    cfg := apiEnv.loadedConfig.Db
    DSNStr := fmt.Sprintf(
        "postgres://%s:%s@%s:%s/%s",
        cfg["db_user"],
        cfg["db_pass"],
        cfg["db_host"],
        cfg["db_port"],
        cfg["db_name"],
    )    
    // if everything is ok try connect...
    pool, err := pgxpool.New(ctx, DSNStr)
    if err != nil {
        return nil, err
    }
    return pool, nil
}
// setup logger (log package)
func setupLogger(logFile *os.File, prefix string) *log.Logger {
    multiOutput := io.MultiWriter(os.Stdout, logFile)
    log.SetOutput(multiOutput)
    return log.New(multiOutput, prefix, log.LstdFlags)
}
// opens main log file and returns resource (also error)
func setLogFile(logName string) (*os.File, error) {
    createdAt := time.Unix(time.Now().Unix(), 0).Format("2006-01-02_150405")
    if logName == "" {
        logName = "api_main_" + createdAt
    }
    logPath, err := filepath.Abs(mainLogPath)
    if err != nil {
        return nil, err
    }
    pathOk, err := FPathExists(logPath)
    if err != nil { return nil, err } // should panic and stop execution
    // create path for logs if not exists
    if ! pathOk {
        if err := os.Mkdir(logPath, 0755); err != nil {
            return nil, err // panic!
        }
    }
    // build full path to log file
    logFilePath := filepath.Join(logPath, logName + ".log")
    // try create file for write
    file, err := os.OpenFile(logFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
    if err != nil {
        return nil, err
    }
    return file, nil
}
// endregion

// region middleware
func RegisterRequestMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
        // fetch context from request
        ctx := r.Context()
        // register ID
        ctx = context.WithValue(ctx, requestIDKey, generateRequestID())
        // cretes new copy of request with new context and go forward        
        next.ServeHTTP(rw, r.WithContext(ctx))
    })
}
// checks base route rules
func CheckRouteMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
        // try to find target route in predefined array
        // r.use(middleware.StripSlashes(next)) todo: // trailing slashes
        ctx := r.Context()
        for _,route := range Routes {
            // 1) try find route
            if route.Path != r.URL.Path {
                continue
            }
            // 2) check method
            if route.Method != r.Method {
                // at this point we can break with error
                http.Error(rw, "Method not allowed.", http.StatusMethodNotAllowed)
                return
            }
            ctx = context.WithValue(r.Context(), endpointSpec, route)
            break
        }
        next.ServeHTTP(rw, r.WithContext(ctx))
    })
}
// prepares custom response
func CustomResponseMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
        interceptor := ResponseInterceptor{}.Get(rw)            
        next.ServeHTTP(interceptor, r)

        returnedData        := interceptor.BodyBuffer.Bytes()
        returnedCode        := interceptor.StatusCode
        internalErrCode     := 0
    
        response := GeneralResponse{
            Code:           returnedCode, 
            ErrorOccured:   returnedCode >= 400,
            IErrorCode:     internalErrCode,
        }

        if response.ErrorOccured {
            response.Status = "error"
            // remove '\n' and other trailing white space
            response.Msg    = strings.TrimSpace(string(returnedData)) // cast bytes to native string (for JSON encoder)
            response.Body   = nil
        } else {
            response.Status = "success"
            response.Msg    = ""
            // check if body isn't empty
            if len(returnedData) > 0 {
                response.Body = json.RawMessage(returnedData)
            } else {
                response.Body = json.RawMessage(`{}`)
            }
        }
        // setting header and status code
        rw.Header().Set("Content-Type", "application/json")
        rw.WriteHeader(returnedCode)
        // generate and sent final JSON response
        err := json.NewEncoder(rw).Encode(response)

        if err != nil {
            log.Printf("[CRITICAL] Failed to encode final response (JSON): %v\n", err)
        }
    })
}
// auth
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
        endpointSpec := GetTargetEndpointSpecsFromContext(r.Context())
        ctx := r.Context()
        // try fetch token from request
        tokenStr, tokenExists := fetchBearerToken(r)
        var u *User;
        isLogged := false
        // get currently logged user (by VALID token)
        if tokenExists {
            u, isLogged = getUserByToken(ctx, tokenStr)
            // also add token to context of current request
            ctx = context.WithValue(ctx, clientTokenKey{}, tokenStr)
        }
        // otherwise, create anonymous
        if !isLogged {
            u = createAnonymous()
        }
        // is it restricted endpoint ?
        if !endpointSpec.Restricted {
            // nothing to do
            ctx = context.WithValue(ctx, userEntityPtrKey{}, u)
            next.ServeHTTP(rw, r.WithContext(ctx))
            return
        }
        // if yes... w have to do few things
        if !isLogged {
            err := ErrorFactory(ERR_ACCESS_DENIED_LOGIN_REQ)
            http.Error(rw, err.Error(), err.HttpCode)
            return
        }
        canLogin := false
        // roles (special cases)
        if GetUserService().IsPrivileged(ctx, u) {
            canLogin = true
        // check commons
        } else {
            for _, roleId := range endpointSpec.RequiredRoles {                
                role, exists := u.Roles[roleId]
                if !exists {
                    err := ErrorFactory(ERR_HIGHER_PRIVILEGES)
                    http.Error(rw, err.Error(), err.HttpCode)
                    return
                }
                // at least min. one role must give access to login into the system
                if !canLogin {
                    if role.CanLogin == true  {
                        canLogin = true;
                    }
                }
            }
        }
        // check: user is verified ?
        if !canLogin {
            err := ErrorFactory(ERR_CANT_LOGIN); http.Error(rw, err.Error(), err.HttpCode)
            return
        }
        // add to context
        ctx = context.WithValue(ctx, userEntityPtrKey{}, u)
        next.ServeHTTP(rw, r.WithContext(ctx))
    })
}
// endregion

// region error-intercepting
// error interceptor for handling custom / API-related errors
func ErrorInterceptor(next CustomHandler) http.HandlerFunc {
    return func(rw http.ResponseWriter, r *http.Request) {
        err := next(rw, r)
        if err == nil {
            return
        }
        var apiErr *ApiResponseError
        if errors.As(err, &apiErr) {
            log.Println(fmt.Errorf("[ERROR] %w", err))
            http.Error(rw, apiErr.Error(), apiErr.HttpCode)            
            return
        }
        http.Error(rw, "Something went wrong...", http.StatusBadRequest)
    }
}
// endregion

// region custom-responds
// general respond
func Respond[T any] (rw http.ResponseWriter, r *http.Request, data T) error {
    // todo: consider format custom err (also err param)
    jsonBytes, err := json.Marshal(data)
    if err != nil {
        log.Println("Json Marshal issue: %w", err)
        return ErrorFactory(ERR_GENERAL_WRITE_RESPONSE, "JSON-encoding failure")
    }
    rw.Write([]byte(jsonBytes))
    return nil
}
// also w. error
func RespondWithError(rw http.ResponseWriter, r *http.Request, e error, cMsg string, cCode int) {
    log.Println(e)
    // consider passing errorinfo
    if cMsg != "" && cCode >= 400 {
        // ignore error interface in response to user
        http.Error(rw, cMsg, cCode)
    } else {
        http.Error(rw, e.Error(), http.StatusInternalServerError)
    }
}
// deprecated
func RespondWithErrorCtx[T any] (rw http.ResponseWriter, r *http.Request, data T, e error) {
    if e != nil {
        var errInfo = ResponseErrorInfo{
            HttpCode: 401,
            Message: "",
            InternalCode: 5801,
        }
        ctx := r.Context()        
        r = r.WithContext(context.WithValue(ctx, errorInfoKey, errInfo))
    }
    jsonBytes, err := json.Marshal(data)
    if err != nil {
        // todo: return general error (status 500)
    }
    rw.Write([]byte(jsonBytes))
}
// endregion

// region other-methods
func GetLoggedUser(ctx context.Context) (*User, bool) {
    u, ok := ctx.Value(userEntityKey).(User);
    if ! ok {
        return createAnonymous(), false
    }
    return &u, true
}
// gets user by token string (or return Guest - token expired / invalid / not passed)
func getUserByToken(ctx context.Context, token string) (*User, bool) {
    u, ok := GetAuthService().GetUserByTokenValue(ctx, token)
    if !ok {
        u = createAnonymous()
        return u, false
    }
    // in any other case
    return u, true
}
// creates anonymous user instance (deprecated)
func createAnonymous() *User {
    // mock
    return &User{
        Name:       "Anonymous",
        Email:      "",
        Roles:      map[string]*Role {USER_ROLE_GUEST: UserRoles[USER_ROLE_GUEST]},
    }
}
// gen. request ID
func generateRequestID() string {
    randomVal := strconv.Itoa(int(rand.UintN(2*10^8)))
    return string(requestIDKey) + randomVal
}
// for `bearer` token
func fetchBearerToken(r *http.Request) (string, bool) {
    rawToken := r.Header.Get("Authorization")
    if rawToken == "" {
        return "", false
    }
    // use regexp to extract token from header string
    re, err := regexp.Compile(`Bearer\s+(\S+)`)
    if err != nil  {
        // handle error
        return "", false
    }
    // try find valid token in string
    matches := re.FindStringSubmatch(rawToken)
    if len(matches) > 1 {
        // bingo !
        return matches[1], true
    }
    // not found
    return "", false
}
// endregion

// region roles-related
// refresh user roles in global scope
func RefreshRoles(ctx context.Context) bool {
    var err error
    // fetch user roles end set in global scope
    UserRoles, err = getUserRoles(ctx)    
    if err != nil {
        log.Println(ErrorFactory(ERR_REFRESH_USER_ROLES, err))
        return false
    }
    return true
}
// gets roles for specified user
func getRolesForUser(u User) map[string]*Role {
    roles := u.Roles
    return roles
}
// get all roles
func getUserRoles(ctx context.Context) (map[string]*Role, error) {
    roles, err := GetUserService().GetRolesFromDb(ctx)
    if err != nil {
        // just return error
        return nil, err
    }
    return roles, nil
}
// endregion

// region shutdown
// Graceful shutdown
func CleanExit() {
    if apiEnv == nil {
        // nothing to do
        return
    }
    // close connection pool
    log.Println("Closing all connections in the pool...")
    apiEnv.GetConnPool().Close()
    // close log file
    log.Println("Done.")
    apiEnv.logFile.Close()
}
// endregion