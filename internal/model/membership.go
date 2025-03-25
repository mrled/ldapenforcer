package model

// Member represents a member of a group
type Member struct {
	// DN is the distinguished name of the member
	DN string

	// Type is the type of member (person, svcacct, group)
	Type string

	// UID is the uid attribute of the member
	UID string

	// IsPosix indicates if the member is a POSIX account
	IsPosix bool
}

// GetGroupMembers returns all members of a group, including members of nested groups
func GetGroupMembers(groupname string, groups map[string]*Group, people map[string]*Person, svcaccts map[string]*SvcAcct,
	enforcedPeopleOU, enforcedSvcAcctOU, enforcedGroupOU string) ([]*Member, error) {

	// Get the group
	group, ok := groups[groupname]
	if !ok {
		return nil, nil
	}

	// Track processed groups to avoid cycles
	processedGroups := make(map[string]bool)

	// Get all members
	var members []*Member

	// Process direct people members
	for _, uid := range group.People {
		person, ok := people[uid]
		if !ok {
			continue
		}

		members = append(members, &Member{
			DN:      createPersonDN(uid, enforcedPeopleOU),
			Type:    "person",
			UID:     uid,
			IsPosix: person.IsPosix(),
		})
	}

	// Process direct service account members
	for _, uid := range group.SvcAccts {
		svcacct, ok := svcaccts[uid]
		if !ok {
			continue
		}

		members = append(members, &Member{
			DN:      createSvcAcctDN(uid, enforcedSvcAcctOU),
			Type:    "svcacct",
			UID:     uid,
			IsPosix: svcacct.IsPosix(),
		})
	}

	// Process nested groups
	processedGroups[groupname] = true
	for _, nestedGroupName := range group.Groups {
		if processedGroups[nestedGroupName] {
			continue // Avoid cycles
		}

		// Get members of nested group (recursive)
		nestedMembers, err := getNestedGroupMembers(
			nestedGroupName,
			groups,
			people,
			svcaccts,
			enforcedPeopleOU,
			enforcedSvcAcctOU,
			enforcedGroupOU,
			processedGroups,
		)
		if err != nil {
			return nil, err
		}

		// Add members from nested group
		members = append(members, nestedMembers...)
	}

	return members, nil
}

// getNestedGroupMembers is a recursive helper function for GetGroupMembers
func getNestedGroupMembers(groupname string, groups map[string]*Group, people map[string]*Person, svcaccts map[string]*SvcAcct,
	enforcedPeopleOU, enforcedSvcAcctOU, enforcedGroupOU string, processedGroups map[string]bool) ([]*Member, error) {

	// Get the group
	group, ok := groups[groupname]
	if !ok {
		return nil, nil
	}

	// Mark this group as processed
	processedGroups[groupname] = true

	// Get all members
	var members []*Member

	// Process direct people members
	for _, uid := range group.People {
		person, ok := people[uid]
		if !ok {
			continue
		}

		members = append(members, &Member{
			DN:      createPersonDN(uid, enforcedPeopleOU),
			Type:    "person",
			UID:     uid,
			IsPosix: person.IsPosix(),
		})
	}

	// Process direct service account members
	for _, uid := range group.SvcAccts {
		svcacct, ok := svcaccts[uid]
		if !ok {
			continue
		}

		members = append(members, &Member{
			DN:      createSvcAcctDN(uid, enforcedSvcAcctOU),
			Type:    "svcacct",
			UID:     uid,
			IsPosix: svcacct.IsPosix(),
		})
	}

	// Process nested groups (recursively)
	for _, nestedGroupName := range group.Groups {
		if processedGroups[nestedGroupName] {
			continue // Avoid cycles
		}

		// Get members of nested group (recursive)
		nestedMembers, err := getNestedGroupMembers(
			nestedGroupName,
			groups,
			people,
			svcaccts,
			enforcedPeopleOU,
			enforcedSvcAcctOU,
			enforcedGroupOU,
			processedGroups,
		)
		if err != nil {
			return nil, err
		}

		// Add members from nested group
		members = append(members, nestedMembers...)
	}

	return members, nil
}

// Helper functions to create DNs
func createPersonDN(uid, enforcedPeopleOU string) string {
	return "uid=" + uid + "," + enforcedPeopleOU
}

func createSvcAcctDN(uid, enforcedSvcAcctOU string) string {
	return "uid=" + uid + "," + enforcedSvcAcctOU
}
