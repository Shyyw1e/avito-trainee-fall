package domain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewTeam_FiltersMembersByTeamName(t *testing.T) {
	members := []User{
		{ID: "u1", Name: "Alice", TeamName: "backend", IsActive: true},
		{ID: "u2", Name: "Bob", TeamName: "backend", IsActive: false},
		{ID: "u3", Name: "Charlie", TeamName: "frontend", IsActive: true},
	}

	team, err := NewTeam("backend", members)
	require.NoError(t, err)
	require.NotNil(t, team)

	require.Equal(t, "backend", team.Name)
	require.Len(t, team.Members, 2)

	for _, m := range team.Members {
		require.Equal(t, "backend", m.TeamName)
	}
}

func TestNewTeam_EmptyName_Error(t *testing.T) {
	_, err := NewTeam("", nil)
	require.Error(t, err)
}

func TestTeam_ActiveMembers(t *testing.T) {
	team := &Team{
		Name: "backend",
		Members: []User{
			{ID: "u1", Name: "Alice", TeamName: "backend", IsActive: true},
			{ID: "u2", Name: "Bob", TeamName: "backend", IsActive: false},
			{ID: "u3", Name: "Charlie", TeamName: "backend", IsActive: true},
		},
	}

	active := team.ActiveMembers()
	require.Len(t, active, 2)
	require.ElementsMatch(t,
		[]string{active[0].ID, active[1].ID},
		[]string{"u1", "u3"},
	)
}

func TestTeam_FindMember(t *testing.T) {
	team := &Team{
		Name: "backend",
		Members: []User{
			{ID: "u1", Name: "Alice", TeamName: "backend", IsActive: true},
			{ID: "u2", Name: "Bob", TeamName: "backend", IsActive: true},
		},
	}

	u, ok := team.FindMember("u2")
	require.True(t, ok)
	require.NotNil(t, u)
	require.Equal(t, "u2", u.ID)
	require.Equal(t, "Bob", u.Name)

	u.Name = "Bobby"
	require.Equal(t, "Bobby", team.Members[1].Name)

	u, ok = team.FindMember("u3")
	require.False(t, ok)
	require.Nil(t, u)
}

func TestTeam_HasMember(t *testing.T) {
	team := &Team{
		Name: "backend",
		Members: []User{
			{ID: "u1", Name: "Alice", TeamName: "backend", IsActive: true},
			{ID: "u2", Name: "Bob", TeamName: "backend", IsActive: true},
		},
	}

	require.True(t, team.HasMember("u1"))
	require.True(t, team.HasMember("u2"))
	require.False(t, team.HasMember("u3"))
}
