package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"slices"
	"strconv"
	"strings"
	"syscall"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Printf("usage: %s <group>[,<group>...] <command_plus_args>\n", os.Args[0])
		os.Exit(1)
	}
	groupNames := strings.Split(os.Args[1], ",")
	err := dropgrp(groupNames, os.Args[2:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", os.Args[0], err.Error())
		os.Exit(2)
	}
}

// filteredGroups takes a list of group strings, which may be names or GIDs, and returns
// a list of the current process' supplementary groups with those groups removed if present.
func filteredGroups(dropGroups []string) (trimmed []int, err error) {
	var gids []int

	// Convert provided group names to group ids
	pgrp := syscall.Getgid() // primary group
	dropGids := make([]int, 0, len(dropGroups))
	for _, group := range dropGroups {
		var userGroup *user.Group
		var gid int

		if gid, err = strconv.Atoi(group); err != nil { // does not parse as number, look up by name
			if userGroup, err = user.LookupGroup(group); err != nil {
				return
			} else if gid, err = strconv.Atoi(userGroup.Gid); err != nil {
				return trimmed, fmt.Errorf("%s: invalid gid returned from group lookup: %w", userGroup.Gid, err)
			}
		}
		if gid == pgrp {
			return trimmed, fmt.Errorf("%s: not a supplementary group\n", group)
		}
		dropGids = append(dropGids, gid)
	}

	// Find current supplementary groups
	if gids, err = syscall.Getgroups(); err != nil {
		return
	}

	// Filter out the groups whose names match the list of groups to drop
	trimmed = make([]int, 0, len(gids))
	for _, gid := range gids {
		if !slices.Contains(dropGids, gid) {
			trimmed = append(trimmed, gid)
		}
	}
	return
}

// dropgrp runs the command provided in commandAndArgs with all of the supplementary groups
// listed in groups removed. Any group in groups not held by the current user is silent ignored,
// as is the users primary group if given.
func dropgrp(groups []string, commandAndArgs []string) error {
	var err error
	var gids []int

	// Get the desired list of groups ids
	if gids, err = filteredGroups(groups); err != nil {
		return err
	}

	// Set the groups
	if err = setgroups(gids); err != nil {
		return err
	}

	// Use Exec syscall to run the provided command
	var binary string
	if binary, err = exec.LookPath(commandAndArgs[0]); err != nil {
		return err
	}
	return syscall.Exec(binary, commandAndArgs, os.Environ())
}

// setgroups calls the Setgroups syscall to apply a set of groups
// to the current process, elevating privileges as required, if
// possible. Privileges are set to normal once this function returns.
func setgroups(gids []int) (err error) {
	uid := syscall.Geteuid()
	if uid != 0 {
		// Become the root user to allow groups to be set
		if err = syscall.Seteuid(0); err != nil {
			return err
		}
		// Deferred function to undo the above however the
		// function exits.
		defer func() {
			syserr := syscall.Seteuid(uid)
			err = errors.Join(err, syserr)
		}()
	}
	return syscall.Setgroups(gids)
}
