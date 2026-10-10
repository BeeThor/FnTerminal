package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// fnOS can create target as 0775 for its dedicated package group. Group write
// is safe only when that group has no members other than this package account.
// SO_PEERCRED is still checked for every connection; filesystem access alone
// never grants authority to supply X-Trim-* headers.
func secureSocketDirectory(path string, st os.FileInfo) error {
	stat, ok := st.Sys().(*syscall.Stat_t)
	if !ok {
		return fmt.Errorf("cannot inspect socket directory owner")
	}
	uid, gid := uint32(os.Geteuid()), uint32(os.Getegid())
	if stat.Uid != 0 && stat.Uid != uid {
		return fmt.Errorf("socket directory owner is not root or package UID")
	}
	if st.Mode().Perm()&0002 != 0 {
		return fmt.Errorf("socket directory is world writable")
	}
	if st.Mode().Perm()&0020 == 0 {
		return nil
	}
	if stat.Gid != gid {
		return fmt.Errorf("socket directory writable by a foreign group")
	}
	group, err := os.ReadFile("/etc/group")
	if err != nil {
		return fmt.Errorf("cannot verify package group membership: %w", err)
	}
	passwd, err := os.ReadFile("/etc/passwd")
	if err != nil {
		return fmt.Errorf("cannot verify package group accounts: %w", err)
	}
	return dedicatedPackageGroup(uid, gid, string(group), string(passwd))
}

func dedicatedPackageGroup(uid, gid uint32, group, passwd string) error {
	name := ""
	for _, line := range strings.Split(passwd, "\n") {
		f := strings.Split(line, ":")
		if len(f) < 4 {
			continue
		}
		accountUID, _ := strconv.ParseUint(f[2], 10, 32)
		accountGID, _ := strconv.ParseUint(f[3], 10, 32)
		if uint32(accountUID) == uid {
			name = f[0]
		}
		if uint32(accountGID) == gid && uint32(accountUID) != uid && accountUID != 0 {
			return fmt.Errorf("package primary group is shared with another account")
		}
	}
	if name == "" {
		return fmt.Errorf("package account is missing from passwd")
	}
	found := false
	for _, line := range strings.Split(group, "\n") {
		f := strings.Split(line, ":")
		if len(f) < 4 {
			continue
		}
		n, e := strconv.ParseUint(f[2], 10, 32)
		if e != nil || uint32(n) != gid {
			continue
		}
		found = true
		for _, member := range strings.Split(f[3], ",") {
			if member != "" && member != name && member != "root" {
				return fmt.Errorf("package group has additional members")
			}
		}
	}
	if !found {
		return fmt.Errorf("package group is missing from group database")
	}
	return nil
}
