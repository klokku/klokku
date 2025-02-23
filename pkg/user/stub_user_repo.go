package user

import (
	"context"
	"errors"
)

type StubUserRepository struct {
	nextId int
	data   map[int]User
}

func NewStubUserRepository() *StubUserRepository {
	nextId := 2
	data := map[int]User{}
	return &StubUserRepository{nextId: nextId, data: data}
}

func (s *StubUserRepository) CreateUser(ctx context.Context, user User) (int, error) {
	s.nextId++
	user.Id = s.nextId
	s.data[s.nextId] = user
	return s.nextId, nil
}

func (s *StubUserRepository) GetUser(ctx context.Context, id int) (User, error) {
	user, ok := s.data[id]
	if !ok {
		return User{}, errors.New("user not found")
	}
	return user, nil
}

func (s *StubUserRepository) UpdateUser(ctx context.Context, userId int, user User) (User, error) {
	if _, ok := s.data[userId]; !ok {
		return User{}, errors.New("user not found")
	}
	s.data[userId] = user
	return user, nil
}

func (s *StubUserRepository) DeleteUser(ctx context.Context, id int) error {
	delete(s.data, id)
	return nil
}

func (s *StubUserRepository) GetAllUsers(ctx context.Context) ([]User, error) {
	var users []User
	for _, user := range s.data {
		users = append(users, user)
	}
	return users, nil
}
