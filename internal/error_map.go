package internal

// err list
const (
    // system
    ERR_UNKNOWN uint16					= iota
    ERR_GENERAL_FAILURE uint16			= iota
    ERR_GENERAL_WRITE_RESPONSE uint16	= iota
    ERR_GET_SYSTEM_USER uint16          = iota
    ERR_LOAD_ROUTES_CONF uint16         = iota
    ERR_LOAD_ROLESCONF_DB uint16        = iota
    ERR_INV_ALIAS_CFG_FOR_ROLES uint16  = iota
    ERR_OPERATION_NOT_PERMITTED uint16  = iota
    ERR_EMPTY_PARAM uint16 				= iota
    ERR_ENTITY_SAVE uint16				= iota
    ERR_ENTITY_UDPATE uint16            = iota
    ERR_ENTITY_REMOVE uint16			= iota
    ERR_ENTITY_UNSAVED uint16			= iota	
    ERR_ASSIGN_ROLE uint16				= iota
    ERR_UNASSIGN_ROLE uint16			= iota
    ERR_ASSIGN_ROLES uint16				= iota
    ERR_REVOKE_ROLES uint16				= iota
    ERR_REMOVE_ROLES uint16				= iota
    ERR_REFRESH_USER_ROLES uint16       = iota
    ERR_FETCH_ROLES uint16				= iota
    ERR_SETUP_ROLES_HANDLER_LVL uint16  = iota
    ERR_INV_ALIAS_FOR_ROLE uint16       = iota
    ERR_ROLE_ALREADY_ASSIGNED uint16    = iota
    ERR_ROLE_REVOKE_NOT_ASSIGNED uint16 = iota
    // request
    ERR_HIGHER_PRIVILEGES uint16		= iota
    ERR_ACCESS_DENIED_LOGIN_REQ uint16  = iota
    ERR_CANT_FETCH_LOGGED_USER uint16   = iota
    ERR_CANT_LOGIN	uint16			    = iota
    ERR_NOT_LOGGED_IN uint16            = iota
    ERR_CANT_REMOVE_USER_TOKENS uint16  = iota  
    ERR_RESOURCE_NOT_FOUND uint16       = iota
    // misc. user	
    ERR_LOGIN_ERROR uint16				= iota
    ERR_INVALID_CREDS uint16			= iota
    ERR_ENTITY_NOT_FOUND uint16			= iota
    ERR_USER_NOT_FOUND uint16			= iota
    ERR_LOGIN_ALREADY_IN_USE uint16     = iota
    ERR_GET_LIST uint16					= iota
    ERR_REVERT_AFTER_CREATE	uint16		= iota
)

var mainErrorsDict = map[uint16][]string{
    ERR_GENERAL_FAILURE: 			{"General error occurred. Can't continue.", 					"G001", "500"},
    ERR_GENERAL_WRITE_RESPONSE:		{"General error while trying to send response. Reason: [%s]",	"G301", "500"},
    ERR_GET_SYSTEM_USER:            {"Unable to get system (dummy) user.",                          "G302", "500"},
    ERR_LOAD_ROUTES_CONF:           {"Can't load (or parse) routes conf. (routes.json) file!",      "G304", "500"},
    ERR_LOAD_ROLESCONF_DB:          {"Can't load roles from DB. Check logs.",                       "G005", "500"},
    ERR_INV_ALIAS_CFG_FOR_ROLES:    {
        "Alias is valid but related role doesn't exist or not found. Check configuration.",         "G006", "500",
    },
    // requests
    ERR_LOGIN_ERROR: 				{"Unable to login. Something went wrong. Try again later...", 	"R010", "401"},
    ERR_NOT_LOGGED_IN:              {"You are not logged in or your session has expired.",          "R013", "401"},
    ERR_CANT_REMOVE_USER_TOKENS:    {"Can't remove user's tokens. Try again later.",                "RO14", "500"},
    ERR_ACCESS_DENIED_LOGIN_REQ:	{
        "Access denied. You must be logged in for access to this resource",                         "R011", "403",
    },
    ERR_HIGHER_PRIVILEGES: 			{"This endpoint requires higher privileges", 					"R012", "403"},
    ERR_CANT_FETCH_LOGGED_USER:     {"Unable to fetch currently logged user. Can't continue",       "G310", "401"}, // no logged user?
    ERR_CANT_LOGIN: 				{
        "Your account is trusted but you can't login at the moment. Contact with support",          "R013", "401",
    },
    ERR_RESOURCE_NOT_FOUND:			{"Resource / path not found.", 									"R003", "404"},
    ERR_OPERATION_NOT_PERMITTED:    {"Well... forget about that. ;) Not pemitted.",                 "R101", "401"},
    // domain
    ERR_EMPTY_PARAM: 				{"Empty param [%s]. Expected value", 							"D001", "400"},
    ERR_GET_LIST:					{"Unable to list entities of type: [%s]", 						"D003", "500"},
    ERR_ENTITY_NOT_FOUND:			{"%s not found.", 												"D002", "404"},
    // more detailed
    ERR_USER_NOT_FOUND:				{"User [%s] not found.", 										"D004", "404"},
    ERR_REVERT_AFTER_CREATE:		{"Unable to ROLLBACK after user CREATE action! Check logs...", 	"D005", "500"},
    // db
    ERR_ENTITY_SAVE: 				{"Can't save: [%s] type.", 							            "P002", "500"},
    ERR_ENTITY_UDPATE:              {"Can't update: [%s] type.",                                    "P003", "500"},
    ERR_ENTITY_REMOVE:				{"Can't remove: [%s] type.", 						            "P004", "500"},
    ERR_ENTITY_UNSAVED:				{"Can't operate on unsaved entity: [%s] type.", 	            "P005", "500"},
    // other `general` / `system`
    ERR_ASSIGN_ROLE: 				{"Can't assign role [%s] to user: [%s]", 			            "P006", "500"},
    ERR_UNASSIGN_ROLE: 				{"Can't unassing role [%s] to user: [%s]", 			            "P007", "500"},
    ERR_ASSIGN_ROLES:				{"Can't assing roles to user [%s]", 				            "P008", "500"},
    ERR_REVOKE_ROLES:				{"Unable to revoke roles. User: [%s]", 				            "P009", "500"},
    ERR_REMOVE_ROLES:				{"Unable to remove roles related to user: [%s]", 	            "P010", "500"},
    ERR_INVALID_CREDS: 				{"Invalid login / password. Login failed", 			            "G202", "403"},
    ERR_LOGIN_ALREADY_IN_USE:       {
        "This email address / login is already in use. Please, choose another one.",                "G203", "400"},
    ERR_FETCH_ROLES:				{"Unable to fetch roles for: %s", 					            "G004", "500"},
    ERR_REFRESH_USER_ROLES:         {"Unable to refresh user's roles. Reason: %s\n",                "P006", "500"},
    ERR_SETUP_ROLES_HANDLER_LVL:    {"Failed to change roles on user ID: %d",                       "G005", "500"},
    ERR_INV_ALIAS_FOR_ROLE:         {"Invalid alias [%s] for role.",                                "G007", "400"},
    ERR_ROLE_ALREADY_ASSIGNED:      {"This user already has this role (%s). Nothig to do.",         "G008", "400"},
    ERR_ROLE_REVOKE_NOT_ASSIGNED:   {"Cannot revoke unassigned role (%s). Nothing to do.",          "G009", "400"},
    ERR_UNKNOWN: 					{"Unknown error occurred. Can't continue", 			            "G000", "500"},
}
