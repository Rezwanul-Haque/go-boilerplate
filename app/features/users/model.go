package users

import "go-boilerplate/app/shared/model"

type User struct {
	model.Base
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
}
