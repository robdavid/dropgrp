package main

import (
	"os/user"
	"strconv"
	"syscall"
	"testing"
)

func groupInfo(t *testing.T) []*user.Group {
	gids, err := syscall.Getgroups()
	if err != nil {
		t.Fatal(err)
	}
	result := make([]*user.Group, len(gids))
	for i, gid := range gids {
		result[i], err = user.LookupGroupId(strconv.Itoa(gid))
		if err != nil {
			t.Fatal(err)
		}
	}
	return result
}

func compareGroups(t *testing.T, expected []*user.Group, actual []int) {
	t.Helper()
	if len(actual) != len(expected) {
		t.Errorf("Expected %d groups(s) but got %d", len(expected), len(actual))
		return
	}
	for i, g := range expected {
		if strconv.Itoa(actual[i]) != g.Gid {
			t.Errorf("Expected gid %s (%s) at index %d but got %d", g.Gid, g.Name, i, actual[i])
		}
	}
}

func pickTestGroup(t *testing.T, groups []*user.Group, skip int) (index int) {
	t.Helper()
	gid := syscall.Getgid()
	var group *user.Group
	for index, group = range groups {
		if strconv.Itoa(gid) == group.Gid {
			continue
		} else if skip == 0 {
			return
		}
		skip--
	}
	t.Log("Insufficient groups for test")
	t.SkipNow()
	return
}

func pickSupplemental(groups []*user.Group) []*user.Group {
	gid := syscall.Getgid()
	result := make([]*user.Group, 0, len(groups))
	for _, g := range groups {
		if strconv.Itoa(gid) != g.Gid {
			result = append(result, g)
		}
	}
	return result
}

func TestFilterGroupsByName(t *testing.T) {
	mygroups := groupInfo(t)
	i := pickTestGroup(t, mygroups, 0)
	filtered, err := filteredGroups([]string{mygroups[0].Name})
	if err != nil {
		t.Fatal(err)
	}
	expected := append(mygroups[:i], mygroups[i+1:]...)
	compareGroups(t, expected, filtered)
}

func TestFilterGroupsById(t *testing.T) {
	mygroups := groupInfo(t)
	i := pickTestGroup(t, mygroups, 1)
	filtered, err := filteredGroups([]string{mygroups[i].Gid})
	if err != nil {
		t.Fatal(err)
	}
	expected := append(mygroups[:i], mygroups[i+1:]...)
	compareGroups(t, expected, filtered)
}

func TestFilterAllGroups(t *testing.T) {
	mygroups := groupInfo(t)
	suppl := pickSupplemental(mygroups)
	names := make([]string, len(suppl))
	for i := range suppl {
		names[i] = suppl[i].Gid
	}
	filtered, err := filteredGroups(names)
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) > 1 {
		t.Fatalf("Expected at most 1 item, got %d", len(filtered))
	} else if len(filtered) == 1 {
		if gid := syscall.Getgid(); filtered[0] != gid {
			t.Errorf("Expected remaining item would be primary group (%d) but is %d", gid, filtered[0])
		}
	}
}

func TestPrimaryGroup(t *testing.T) {
	gid := syscall.Getgid()
	_, err := filteredGroups([]string{strconv.Itoa(gid)})
	if err == nil {
		t.Fatal("Expected error did not occur")
	}
}
