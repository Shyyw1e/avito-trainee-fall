package domain

import "fmt"

type Team struct {
	Name    string
	Members []User
}

func NewTeam(name string, members []User) (*Team, error) {
	if name == "" {
		return nil, fmt.Errorf("empty teamName")
	}

	validatedMembers := make([]User, 0, len(members))
	for _, m := range members {
		if m.TeamName == name {
			validatedMembers = append(validatedMembers, m)
		}
	}

	return &Team{
		Name:    name,
		Members: validatedMembers,
	}, nil
}

func (t *Team) ActiveMembers() []User {
	active := make([]User, 0, len(t.Members))
	for _, m := range t.Members {
		if m.IsActive {
			active = append(active, m)
		}
	}
	return active
}

func (t *Team) FindMember(userID string) (*User, bool) {
	for i := range t.Members {
		if t.Members[i].ID == userID {
			return &t.Members[i], true
		}
	}
	return nil, false
}

func (t *Team) HasMember(userID string) bool {
	for _, m := range t.Members {
		if m.ID == userID {
			return true
		}
	}
	return false
}
