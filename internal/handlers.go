package internal

/**
 * Handlers
 * few fundamental rules:
 *
 * All handlers MUST return "error"
 * All handlers MUST use Respond / ErrorFactory methods
 * All errors SHOULD be prepared in services and
 * ... also ALL ERRORS MUST contains NON-SENSITIVE informations
 * e.g. "User not found.", "Can't read data. Try later..." or something like that.
 * All sensitive data MUST be logged in services or OPTIONALLY 
 * may be use in response but ONLY IN DEBUG MODE!
 *
 * Well, thats it.
 */

import (
    "fmt"
    "net/http"
    "strconv"
)

//==============================
// USERS
//==============================

// Route: /users/login
func Login(rw http.ResponseWriter, r *http.Request) error {
    ctx		:= r.Context()
    data 	:= GetValidatedDataFromContext(ctx)
    login, phrase := data["email"].GetText(), data["phrase"].GetText()
    u, err := GetAuthService().TryLoginViaMail(ctx, login, phrase)
    if(err != nil) {
        return ErrorFactory(ERR_LOGIN_ERROR)
    }
    var result = map[string]any {
        "user":			u,
        "token": 		u.TokenData.Value,
    }
    return Respond(rw, r, result)	
}

// Route: /users/logout
func Logout(rw http.ResponseWriter, r *http.Request) error {
    err := GetAuthService().Logout(r.Context())
    if err != nil {
        return err
    }
    return Respond(rw, r, "Logged out! See you next time.")
}

// Route: /users/register
func RegisterUser(rw http.ResponseWriter, r *http.Request) error {
    ctx 	:= r.Context()
    // grab incoming data container and also UserService
    data 	:= GetValidatedDataFromContext(ctx)
    // todo: validator for input data
    us 		:= GetUserService()
    // type assertion (safe because of schema validator in middleware)
    fName, lName, email, phrase, nick :=
    data["first_name"].GetText(),
    data["last_name"].GetText(),
    data["email"].GetText(),
    data["phrase"].GetText(),
    data["nickname"].GetText()
    // create
    user, err := us.Create(ctx, fName, lName, email, phrase, nick)
    if err != nil {
        return err
    }
    var result = map[string]any {
        "created_user": user,
    }
    return Respond(rw, r, result)
}

// Route: /users/update
func UpdateUser(rw http.ResponseWriter, r *http.Request) error {
    u := &User{}
    ctx				:= r.Context()
    loggedUser, _ 	:= GetAuthService().GetLoggedUserPtr(ctx)
    data			:= GetValidatedDataFromContext(ctx)
    // get service
    us 	:= GetUserService()
    // for admins / bots
    if us.IsPrivileged(ctx, loggedUser) {
        // assign admin for now..
        u = loggedUser
        // then check for id of "foreign" user
        uID := FetchNum[int](data["uid"])
        if uID != 0 {
            user, exists := us.Find(ctx, uID)
            if exists {
                // time to switch and operate on "foreign" user
                u = user
            }
            // or update "myself"
        }
    } else {
        u = loggedUser
    }
    updatedUser, err := us.UpdatePersonalInfo(ctx, u, data)
    if err != nil {
        return err
    }
    return Respond(rw, r, updatedUser)	
}

// removes specified user
// Route: /users/delete
func DeleteUser(rw http.ResponseWriter, r *http.Request) error {
    ctx 		:= r.Context()
    // just for sure
    us			:= GetUserService()
    admin, ok 	:= GetAuthService().GetLoggedUserPtr(ctx)
    if ! ok {
        // somehow there's no logged user / error while trying to get from context or via token
        return ErrorFactory(ERR_CANT_FETCH_LOGGED_USER)
    }
    if ! us.IsPrivileged(ctx, admin) {
        return ErrorFactory(ERR_OPERATION_NOT_PERMITTED)
    }
    // then
    id  := FetchNum[int](GetValidatedDataFromContext(ctx)["id"])
    // try find user
    u, exists := us.Find(ctx, id)
    if ! exists {
        return ErrorFactory(ERR_USER_NOT_FOUND, strconv.Itoa(id))
    }
    err := us.Remove(ctx, u)
    if err == nil {
        return Respond(rw, r, []string{strconv.Itoa(id), "Deleted"})
    }
    return err
}

// shows users collection by limit, offset
// Route: /users/list
func ListUsers(rw http.ResponseWriter, r *http.Request) error {
    ctx			:= r.Context()
    data  		:= GetValidatedDataFromContext(ctx)
    list, err 	:= GetUserService().List(
        ctx,		
        FetchNum[int](data["limit"]),
        FetchNum[int](data["offset"]),
    )
    if err == nil { // looks ok, so...
        return Respond(rw, r, list) 
    }
    return err
}

//==============================
// USERS END
//==============================

// region experiments / tests
func IndexAction(rw http.ResponseWriter, r *http.Request) error {
    data 		:= GetValidatedDataFromContext(r.Context())
    fName 		:= data["first-name"].GetText()
    lName  		:= data["last-name"].GetText()
    age 		:= data["age"].GetInt()
    u, isLogged := GetAuthService().GetLoggedUserPtr(r.Context())
    result := make(map[string]any)
    result["Logged in?"] = isLogged
    if (isLogged) {
        result["logged_user"] = u.FirstName + " " + u.LastName + " " + " with id: " + strconv.Itoa(u.GetID()) + " :)"
    }
    result["hello_message"] = fmt.Sprintf("Your are: %s %s, and your age is: %d", fName, lName, age)
    return Respond(rw, r, result)	
}

func HelloAction(rw http.ResponseWriter, r *http.Request) error {
    ctx 				:= r.Context()
    data 				:= GetValidatedDataFromContext(ctx)
    u, isLogged 		:= GetAuthService().GetLoggedUserPtr(ctx)
    result := make(map[string]any)
    result["Logged in?"] = isLogged
    if (isLogged) {
        result["logged_user"] = u.FirstName + " " + u.LastName + " " + " with id: " + strconv.Itoa(u.GetID()) + " :)"
    }

    name 		                := data["who"].GetText()
    option  	                := data["some-option"].GetBool()
    age			                := data["age"].GetText()
    result["first_message"]     = fmt.Sprintf("Hello %s! You are %v", name, age)

    if(option) {
        result["second_message"] = "Second message has been passed!"
    } else {
        result["second_message"] = "No second message. Default value will be used."
    }

    return Respond(rw,r,result)	
}


func MemberZoneAction(rw http.ResponseWriter, r *http.Request) error {
    data := GetValidatedDataFromContext(r.Context())
    type Happy func(V bool)	
    feelsLike := func(isHappy bool) string {
        if (isHappy) { return "great!"}
        return "... dump coder"
    }
    return Respond(rw,r, map[string]any {
        "first_name": 	data["firstname"].GetText(),
        "last_name": 	data["lastname"].GetText(),
        "Email:": 		data["email"].GetText(),
        "Feels like": 	feelsLike(data["feels_happy"].GetBool()),
        "Your age:": 	data["age"].GetInt(),
        "Your gender:": data["gender"].GetText(),
    })	
}

func AdminAction(rw http.ResponseWriter, r *http.Request) error {
    data := GetValidatedDataFromContext(r.Context())
    result := make(map[string]string)
    result["admin_message"] = "You are ADMIN with name: " + (data["name"].GetText())	
    return Respond(rw,r,result)	
}

func EditHandler(rw http.ResponseWriter, r *http.Request) error {
    data := GetValidatedDataFromContext(r.Context())
    result := make(map[string]any)
    result["email_edit"] = "Your new email is: " + data["email"].GetText()
    result["ID"] = data["id"]
    return Respond(rw,r,result)	
}
// endregion