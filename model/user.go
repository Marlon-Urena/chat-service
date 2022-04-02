package model

import (
	"messageService/dto"
	"time"
)

type UserModel struct {
	UID          string
	FirstName    *string
	LastName     *string
	Username     string
	Email        string
	Address      *string
	City         *string
	State        *string
	Country      *string
	ZipCode      *string
	PhotoUrl     *string
	PhoneNumber  *string
	Status       string
	LastActivity time.Time
}

func (u *UserModel) ConvertToDTO() dto.UserDTO {
	var name string
	if u.FirstName != nil && u.LastName != nil {
		name = *u.FirstName + " " + *u.LastName
	}
	return dto.UserDTO{
		Id:           u.UID,
		Email:        u.Email,
		Username:     u.Username,
		Name:         &name,
		Avatar:       u.PhotoUrl,
		Status:       u.Status,
		Position:     nil,
		PhoneNumber:  u.PhoneNumber,
		Address:      u.Address,
		LastActivity: u.LastActivity,
	}
}
