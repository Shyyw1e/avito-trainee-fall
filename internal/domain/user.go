package domain

import "fmt"

type User struct {
	ID			string
	Name		string
	TeamName	string
	IsActive	bool
}

func NewUser(id, username, teamName string, isActive bool) (*User, error) {
	if id == "" || username == "" || teamName == "" {
		return nil, fmt.Errorf("empty parameter")
	}
	return &User{
		ID: id,
		Name: username,
		TeamName: teamName,
		IsActive: isActive,
	}, nil
}

func (u *User) Deactivate() {
	u.IsActive = false
}

func (u *User) Activate() {
	u.IsActive = true
}