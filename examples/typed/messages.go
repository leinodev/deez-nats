package main

type UserCreatedEvent struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UserRenamedEvent struct {
	ID      string `json:"id"`
	NewName string `json:"new_name"`
}

type GetUserRequest struct {
	ID string `json:"id"`
}

type GetUserResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UpdateUserRequest struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type UpdateUserResponse struct {
	Updated bool `json:"updated"`
}
