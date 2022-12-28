package main

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/google/go-github/v48/github"
	"github.com/stretchr/testify/mock"
)

type mockOrganizationsService struct {
	mock.Mock
}

func (m *mockOrganizationsService) Get(ctx context.Context, org string) (*github.Organization, *github.Response, error) {
	args := m.Called(ctx, org)
	return args.Get(0).(*github.Organization), args.Get(1).(*github.Response), args.Error(2)
}

func (m *mockOrganizationsService) ListMembers(ctx context.Context, org string, opt *github.ListMembersOptions) ([]*github.User, *github.Response, error) {
	args := m.Called(ctx, org, opt)
	return args.Get(0).([]*github.User), args.Get(1).(*github.Response), args.Error(2)
}

type mockTeamsService struct {
	mock.Mock
}

func (m *mockTeamsService) ListTeams(ctx context.Context, org string, opt *github.ListOptions) ([]*github.Team, *github.Response, error) {
	args := m.Called(ctx, org, opt)
	return args.Get(0).([]*github.Team), args.Get(1).(*github.Response), args.Error(2)
}

func (m *mockTeamsService) ListTeamMembersByID(ctx context.Context, orgID, teamID int64, opts *github.TeamListTeamMembersOptions) ([]*github.User, *github.Response, error) {
	args := m.Called(ctx, orgID, teamID, opts)
	return args.Get(0).([]*github.User), args.Get(1).(*github.Response), args.Error(2)
}

func TestShoudCheckOrganizationAccess(t *testing.T) {
	mockOS := new(mockOrganizationsService)
	mockOS.On("Get", mock.Anything, "test-org").Return(&github.Organization{ID: github.Int64(123)}, &github.Response{}, nil)
	mockOS.On("Get", mock.Anything, "forbidden-org").Return(&github.Organization{}, &github.Response{}, fmt.Errorf("forbidden"))

	processor := Processor{
		Context:              context.Background(),
		OrganizationsService: mockOS,
	}

	orgID, err := processor.GetOrganizationID("test-org")
	if err != nil {
		t.Errorf("Error creating processor: %v", err)
	}
	if orgID != 123 {
		t.Errorf("Expected 123, got %v", orgID)
	}

	if _, err := processor.GetOrganizationID("forbidden-org"); err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestShoudListMembers(t *testing.T) {
	mockOS := new(mockOrganizationsService)
	mockOS.On("ListMembers", mock.Anything, "test-org", mock.Anything).Return([]*github.User{
		{Login: github.String("test-user")},
		{Login: github.String("test-user-2")},
	}, &github.Response{}, nil)
	mockOS.On("ListMembers", mock.Anything, "bad-org", mock.Anything).Return([]*github.User{}, &github.Response{}, fmt.Errorf("error"))

	processor := Processor{
		Context:              context.Background(),
		OrganizationsService: mockOS,
	}

	members, err := processor.Members("test-org")
	if err != nil {
		t.Errorf("Error creating processor: %v", err)
	}

	expected := []string{"test-user", "test-user-2"}

	if !reflect.DeepEqual(members, expected) {
		t.Errorf("Expected %v, got %v", expected, members)
	}

	_, err = processor.Members("bad-org")
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}

func TestShouldGetTeams(t *testing.T) {
	mockTS := new(mockTeamsService)
	mockTS.On("ListTeams", mock.Anything, "test-org", mock.Anything).Return([]*github.Team{
		{ID: github.Int64(1), Name: github.String("test-team")},
		{ID: github.Int64(2), Name: github.String("test-team-2"), Parent: &github.Team{ID: github.Int64(1), Name: github.String("test-team")}},
	}, &github.Response{}, nil)
	mockTS.On("ListTeams", mock.Anything, "bad-org", mock.Anything).Return([]*github.Team{}, &github.Response{}, fmt.Errorf("error"))
	mockTS.On("ListTeams", mock.Anything, "bad-org-2", mock.Anything).Return([]*github.Team{
		{ID: github.Int64(3), Name: github.String("test-team")},
	}, &github.Response{}, nil)

	mockTS.On("ListTeamMembersByID", mock.Anything, int64(123), int64(1), mock.Anything).Return([]*github.User{
		{Login: github.String("test-user")},
		{Login: github.String("test-user-2")},
	}, &github.Response{}, nil)
	mockTS.On("ListTeamMembersByID", mock.Anything, int64(123), int64(2), mock.Anything).Return([]*github.User{
		{Login: github.String("test-user-3")},
		{Login: github.String("test-user-4")},
	}, &github.Response{}, nil)
	mockTS.On("ListTeamMembersByID", mock.Anything, int64(125), int64(3), mock.Anything).Return([]*github.User{}, &github.Response{}, fmt.Errorf("error"))

	processor := Processor{
		Context:      context.Background(),
		TeamsService: mockTS,
	}

	teams, parents, err := processor.Teams("test-org", 123)
	if err != nil {
		t.Errorf("Error creating processor: %v", err)
	}

	expected := map[string][]string{
		"test-team":   {"test-user", "test-user-2"},
		"test-team-2": {"test-user-3", "test-user-4"},
	}

	if !reflect.DeepEqual(teams, expected) {
		t.Errorf("Expected teams to be %v, got %v", expected, teams)
	}

	expectedParents := map[string]string{
		"test-team-2": "test-team",
	}

	if !reflect.DeepEqual(parents, expectedParents) {
		t.Errorf("Expected parents to be %v, got %v", expectedParents, parents)
	}

	_, _, err = processor.Teams("bad-org", 124)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	_, _, err = processor.Teams("bad-org-2", 125)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
}
