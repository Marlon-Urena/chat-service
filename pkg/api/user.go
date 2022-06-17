package api

import "errors"

type UserService interface {
	GetUserByIds(userIds []string) ([]*UserModel, error)
	GetUsersByUsernameContaining(query string) ([]*UserModel, error)
}

type UserRepository interface {
	GetUserByIds(userIds []string) ([]*UserModel, error)
	GetUsersByUsernameContaining(query string) ([]*UserModel, error)
}

type userService struct {
	storage UserRepository
}

func NewUserService(repository UserRepository) UserService {
	return &userService{storage: repository}
}

func (u userService) GetUserByIds(userIds []string) ([]*UserModel, error) {
	if len(userIds) == 0 {
		return nil, errors.New("userId array is empty")
	}

	users, err := u.storage.GetUserByIds(userIds)

	if err != nil {
		return nil, err
	}

	return users, nil
}

func (u userService) GetUsersByUsernameContaining(username string) ([]*UserModel, error) {
	if username == "" {
		return nil, errors.New("username is empty")
	}

	user, err := u.storage.GetUsersByUsernameContaining(username)

	if err != nil {
		return nil, err
	}

	return user, nil
}
