package main

import (
	"reflect"
	"testing"
)

func TestShouldFindSubsets(t *testing.T) {
	teams := map[string][]string{
		"test-team":   {"test-user", "test-user-2", "test-user-3"},
		"test-team-2": {"test-user", "test-user-2"},
		"test-team-3": {"test-user-3", "test-user-4"},
		"test-team-4": {"test-user-3", "test-user-4"},
	}

	subsets := FindSubsets(teams)

	expected := map[string]string{
		"test-team-2": "test-team",
		"test-team-3": "test-team-4",
		"test-team-4": "test-team-3",
	}

	if !reflect.DeepEqual(subsets, expected) {
		t.Errorf("Expected subsets to be %v, got %v", expected, subsets)
	}
}

func TestShouldFindMembersWithoutTeam(t *testing.T) {
	teams := map[string][]string{
		"test-team":   {"test-user", "test-user-2", "test-user-3"},
		"test-team-2": {"test-user", "test-user-2"},
	}

	members := []string{"test-user", "test-user-2", "test-user-3", "test-user-4"}

	membersWithoutTeam := FindMembersWithoutTeam(teams, members)

	expected := []string{"test-user-4"}

	if !reflect.DeepEqual(membersWithoutTeam, expected) {
		t.Errorf("Expected members without team to be %v, got %v", expected, membersWithoutTeam)
	}
}
