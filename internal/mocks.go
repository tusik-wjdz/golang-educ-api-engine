package internal

import ()

var MockRoles = map[string]*Role {
    USER_ROLE_SYSTEM: {        
        Name:       "System",
        Desc:       "System virtual user for internal tasks.",
        IsAdmin:    true,
        IsSystem:   true,
        Trusted:    true,
        CanLogin:   false, // blocked
    },
    USER_ROLE_ADMIN: {        
        Name:       "Admin",
        Desc:       "API Administrator",
        IsAdmin:    true,
        IsSystem:   true,
        Trusted:    true,
        CanLogin:   true,
    },
    USER_ROLE_GUEST: {        
        Name:       "Guest",
        Desc:       "Restricted access to resources.",
        IsAdmin:    true,
        IsSystem:   true,
        Trusted:    false,
        CanLogin:   false, // bacause logged guest becomes typical logged user (id 4)
    },
    USER_ROLE_TRUSTED: {        
        Name:       "Trusted",
        Desc:       "Typical (trusted) client of the API",
        IsAdmin:    true,
        IsSystem:   true,
        Trusted:    true,
        CanLogin:   true,
    },
}

var MockUserToken = UserToken{
    Value: "ABCDE12345678",
    ValidTo: -1,
}

var MockUser = User{
    Name: "Wojciech Dzieciol",
    Email: "user@gmail.com",
    Roles: map[string]*Role {
        USER_ROLE_TRUSTED: MockRoles[USER_ROLE_TRUSTED],
        USER_ROLE_GUEST: MockRoles[USER_ROLE_GUEST],
        //USER_ROLE_ADMIN: MockRoles[USER_ROLE_ADMIN],
    },
    TokenData: &MockUserToken,
}

// routes

