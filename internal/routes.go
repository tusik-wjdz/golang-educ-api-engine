package internal

import (
    "encoding/json"
    "log"
    "os"
)

var HandlerRegistry = map[string]CustomHandler{
    // `almost`` friendly name  | unfriendly handler func ;)
    // admin
    "assignRole":               AssignRole,
    "revokeRole":               RevokeRole,
    "deleteUser":               DeleteUser,
    "listUser":                 ListUsers,
    // users
    "login":                    Login,
    "logout":                   Logout,
    "register":                 RegisterUser,
    "updateUser":               UpdateUser, 
    // domain handlers
    // weather stations
    "findWeatherStation":       FindWeatherStation,
    "createWeatherStation":     CreateWeatherStation,
    "updateWeatherStation":     UpdateWeatherStation,
    "listWeatherStation":       ListWeatherStations,
    "deleteWeatherStation":     DeleteWeatherStation,
    // products
    "createProduct":            CreateProduct,
    "updateProduct":            UpdateProduct,
    "listProduct":              ListProducts,
    "listProductUserCtx":       ListProductsInUserCtx,
    "deleteProduct":            DeleteProduct,
    // ...
    // misc / testing
    "hello":                    HelloAction,
    "index":                    IndexAction,
    "member-zone":              MemberZoneAction,
    "admin":                    AdminAction,
    "edit":                     EditHandler,
}

var Routes      = []Route{}
var ParamList   = make(map[string][]IncomingParam)

func importConfig() bool {
    data, err := os.ReadFile("./conf/routes.json")
    if err != nil {
        log.Println("Routes conf. file (endpoints w. params specifications) NOT FOUND!")
        return false
    }
    // routes with nested params schema in DTO
    var dtos []RouteDTO
    if err := json.Unmarshal(data, &dtos); err != nil {
        log.Println(err)        
        return false
    }
    // iterate over data transfer objects
    for i, dto := range dtos {
        if dto.Name == "" || dto.Path == "" {
            log.Printf("[WARN] Invalid endpoint specification at index: [%d]\n", i)
        }
        // time to find our handler in registered handlers
        customHandlerFunc, exists := HandlerRegistry[dto.Name]
        if ! exists {
            log.Printf("[WARN] Handler function for action: %s, path: %s, NOT FOUND!\n", dto.Name, dto.Path)
            continue
        }
        r := Route{
            Name:               dto.Name,
            Path:               dto.Path,
            Method:             dto.Method,
            Restricted:         dto.Restricted,
            RequiredRoles:      dto.RequiredRoles,            
            Handler:            customHandlerFunc,
        }
        // add endpoint specs to routes slice
        Routes = append(Routes, r)
        // time to parse params specs.
        var incParams []IncomingParam
        for _, parDto := range dto.Params {            
            if parDto.Type == "" {
                parDto.Type = PARAM_TYPE_STRING
            }
            // `schema` for incoming param (defined in conf/routes.json for each one param)
            incParamSpecs := IncomingParam{
                Name:           parDto.Name,
                Type:           parDto.Type,
                IsRequired:     parDto.IsRequired,
                UrlOnly:        parDto.UrlOnly,
                CanBeEmpty:     parDto.CanBeEmpty,
                Default:        parDto.Default,
            }
            // add param
            incParams = append(incParams, incParamSpecs)
        }
        // put param list into dedicated map (paramList)
        ParamList[dto.Name] = incParams
    }
    return true
}