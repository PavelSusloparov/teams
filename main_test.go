package main

import (
	"reflect"
	"sort"
	"testing"
)

func TestShouldFindSubsets(t *testing.T) {
	teams := map[string][]string{
		"test-team":   {"test-user", "test-user-2", "test-user-3"},
		"test-team-2": {"test-user", "test-user-2"},
		"test-team-3": {"test-user-3", "test-user-4"},
		"test-team-4": {"test-user-3", "test-user-4"},
		"test-team-5": {"test-user", "test-user-2", "test-user-4"},
	}

	subsets := FindSubsets(teams)

	expected := map[string][]string{
		"test-team-2": {"test-team", "test-team-5"},
		"test-team-3": {"test-team-4"},
		"test-team-4": {"test-team-3"},
	}

	for team, expectedSubsets := range expected {
		teamSubsets := subsets.GetSubsets(team)
		sort.Strings(teamSubsets)

		if !reflect.DeepEqual(teamSubsets, expectedSubsets) {
			t.Errorf("Expected subsets for team %s to be %v, got %v", team, expectedSubsets, subsets.GetSubsets(team))
		}
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
