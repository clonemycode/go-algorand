// Copyright (C) 2019-2025 Algorand, Inc.
// This file is part of go-algorand
//
// go-algorand is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// go-algorand is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with go-algorand.  If not, see <https://www.gnu.org/licenses/>.

package phonebook

import (
	"slices"
	"testing"
	"time"

	"github.com/algorand/go-algorand/test/partitiontest"
	"github.com/stretchr/testify/require"
)

func testPhonebookAll(t *testing.T, set []string, ph Phonebook) {
	actual := ph.GetAddresses(len(set), RelayRole)
	for _, got := range actual {
		ok := slices.Contains(set, got)
		if !ok {
			t.Errorf("get returned junk %#v", got)
		}
	}
	for _, known := range set {
		ok := slices.Contains(actual, known)
		if !ok {
			t.Errorf("get missed %#v; actual=%#v; set=%#v", known, actual, set)
		}
	}
}

func testPhonebookUniform(t *testing.T, set []string, ph Phonebook, getsize int) {
	uniformityTestLength := 250000 / len(set)
	expected := (uniformityTestLength * getsize) / len(set)
	counts := make([]int, len(set))
	for i := 0; i < uniformityTestLength; i++ {
		actual := ph.GetAddresses(getsize, RelayRole)
		for i, known := range set {
			for _, xa := range actual {
				if known == xa {
					counts[i]++
				}
			}
		}
	}
	min := counts[0]
	max := counts[0]
	for i := 1; i < len(counts); i++ {
		if counts[i] > max {
			max = counts[i]
		}
		if counts[i] < min {
			min = counts[i]
		}
	}
	// TODO: what's a good probability-theoretic threshold for good enough?
	if max-min > (expected / 5) {
		t.Errorf("counts %#v", counts)
	}
}

func TestArrayPhonebookAll(t *testing.T) {
	partitiontest.PartitionTest(t)

	set := []string{"a", "b", "c", "d", "e"}
	ph := MakePhonebook(1, 1).(*phonebookImpl)
	for _, e := range set {
		ph.data[e] = makePhonebookEntryData("", RelayRole, false)
	}
	testPhonebookAll(t, set, ph)
}

func TestArrayPhonebookUniform1(t *testing.T) {
	partitiontest.PartitionTest(t)

	set := []string{"a", "b", "c", "d", "e"}
	ph := MakePhonebook(1, 1).(*phonebookImpl)
	for _, e := range set {
		ph.data[e] = makePhonebookEntryData("", RelayRole, false)
	}
	testPhonebookUniform(t, set, ph, 1)
}

func TestArrayPhonebookUniform3(t *testing.T) {
	partitiontest.PartitionTest(t)

	set := []string{"a", "b", "c", "d", "e"}
	ph := MakePhonebook(1, 1).(*phonebookImpl)
	for _, e := range set {
		ph.data[e] = makePhonebookEntryData("", RelayRole, false)
	}
	testPhonebookUniform(t, set, ph, 3)
}

func TestMultiPhonebook(t *testing.T) {
	partitiontest.PartitionTest(t)

	set := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	pha := append([]string{}, set[:5]...)
	phb := append([]string{}, set[5:]...)
	mp := MakePhonebook(1, 1*time.Millisecond)
	mp.ReplacePeerList(pha, "pha", RelayRole)
	mp.ReplacePeerList(phb, "phb", RelayRole)

	testPhonebookAll(t, set, mp)
	testPhonebookUniform(t, set, mp, 1)
	testPhonebookUniform(t, set, mp, 3)
}

// TestMultiPhonebookPersistentPeers validates that the peers added via Phonebook.AddPersistentPeers
// are not replaced when Phonebook.ReplacePeerList is called
func TestMultiPhonebookPersistentPeers(t *testing.T) {
	partitiontest.PartitionTest(t)

	persistentPeers := []string{"a"}
	set := []string{"b", "c", "d", "e", "f", "g", "h", "i", "j", "k"}
	pha := append([]string{}, set[:5]...)
	phb := append([]string{}, set[5:]...)
	mp := MakePhonebook(1, 1*time.Millisecond)
	mp.AddPersistentPeers(persistentPeers, "pha", RelayRole)
	mp.AddPersistentPeers(persistentPeers, "phb", RelayRole)
	mp.ReplacePeerList(pha, "pha", RelayRole)
	mp.ReplacePeerList(phb, "phb", RelayRole)

	testPhonebookAll(t, append(set, persistentPeers...), mp)
	allAddresses := mp.GetAddresses(len(set)+len(persistentPeers), RelayRole)
	for _, pp := range persistentPeers {
		require.Contains(t, allAddresses, pp)
	}

	// check that role of persistent peer gets updated with AddPersistentPeers
	mp2 := MakePhonebook(1, 1*time.Millisecond)
	mp2.AddPersistentPeers(persistentPeers, "phc", RelayRole)
	mp2.AddPersistentPeers(persistentPeers, "phc", ArchivalRole)
	allAddresses = mp2.GetAddresses(len(set)+len(persistentPeers), RelayRole)
	require.Len(t, allAddresses, 1)
	allAddresses = mp2.GetAddresses(len(set)+len(persistentPeers), ArchivalRole)
	require.Len(t, allAddresses, 1)

	// check that role of persistent peer survives
	mp3 := MakePhonebook(1, 1*time.Millisecond)
	mp3.AddPersistentPeers(persistentPeers, "phc", ArchivalRole)
	phc := []string{"a"}
	mp3.ReplacePeerList(phc, "phc", RelayRole)

	allAddresses = mp3.GetAddresses(len(set)+len(persistentPeers), RelayRole)
	require.Len(t, allAddresses, 1)
	allAddresses = mp3.GetAddresses(len(set)+len(persistentPeers), ArchivalRole)
	require.Len(t, allAddresses, 1)
}

func TestMultiPhonebookDuplicateFiltering(t *testing.T) {
	partitiontest.PartitionTest(t)

	set := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}
	pha := append([]string{}, set[:7]...)
	phb := append([]string{}, set[3:]...)
	mp := MakePhonebook(1, 1*time.Millisecond)
	mp.ReplacePeerList(pha, "pha", RelayRole)
	mp.ReplacePeerList(phb, "phb", RelayRole)

	testPhonebookAll(t, set, mp)
	testPhonebookUniform(t, set, mp, 1)
	testPhonebookUniform(t, set, mp, 3)
}

func TestWaitAndAddConnectionTimeLongtWindow(t *testing.T) {
	partitiontest.PartitionTest(t)

	// make the connectionsRateLimitingWindow long enough to avoid triggering it when the
	// test is running in a slow environment
	// The test will artificially simulate time passing
	timeUnit := 2000 * time.Second
	connectionsRateLimitingWindow := 2 * timeUnit
	entries := MakePhonebook(3, connectionsRateLimitingWindow).(*phonebookImpl)
	addr1 := "addrABC"
	addr2 := "addrXYZ"

	// Address not in. Should return false
	addrInPhonebook, _, provisionalTime := entries.GetConnectionWaitTime(addr1)
	require.Equal(t, false, addrInPhonebook)
	require.Equal(t, false, entries.UpdateConnectionTime(addr1, provisionalTime))

	// Test the addresses are populated in the phonebook and a
	// time can be added to one of them
	entries.ReplacePeerList([]string{addr1, addr2}, "default", RelayRole)
	addrInPhonebook, waitTime, provisionalTime := entries.GetConnectionWaitTime(addr1)
	require.Equal(t, true, addrInPhonebook)
	require.Equal(t, time.Duration(0), waitTime)
	require.Equal(t, true, entries.UpdateConnectionTime(addr1, provisionalTime))
	phBookData := entries.data[addr1].recentConnectionTimes
	require.Equal(t, 1, len(phBookData))

	// simulate passing a unit of time
	for rct := range entries.data[addr1].recentConnectionTimes {
		entries.data[addr1].recentConnectionTimes[rct] = entries.data[addr1].recentConnectionTimes[rct].Add(-1 * timeUnit)
	}

	// add another value to addr
	_, waitTime, provisionalTime = entries.GetConnectionWaitTime(addr1)
	require.Equal(t, time.Duration(0), waitTime)
	require.Equal(t, true, entries.UpdateConnectionTime(addr1, provisionalTime))
	phBookData = entries.data[addr1].recentConnectionTimes
	require.Equal(t, 2, len(phBookData))

	// simulate passing a unit of time
	for rct := range entries.data[addr1].recentConnectionTimes {
		entries.data[addr1].recentConnectionTimes[rct] =
			entries.data[addr1].recentConnectionTimes[rct].Add(-1 * timeUnit)
	}

	// the first time should be removed and a new one added
	// there should not be any wait
	_, waitTime, provisionalTime = entries.GetConnectionWaitTime(addr1)
	require.Equal(t, time.Duration(0), waitTime)
	require.Equal(t, true, entries.UpdateConnectionTime(addr1, provisionalTime))
	phBookData2 := entries.data[addr1].recentConnectionTimes
	require.Equal(t, 2, len(phBookData2))

	// make sure the right time was removed
	require.Equal(t, phBookData[1], phBookData2[0])
	require.Equal(t, true, phBookData2[0].Before(phBookData2[1]))

	// try requesting from another address, make sure
	// a separate array is used for these new requests

	// add 3 values to another address. should not wait
	// value 1
	_, waitTime, provisionalTime = entries.GetConnectionWaitTime(addr2)
	require.Equal(t, time.Duration(0), waitTime)
	require.Equal(t, true, entries.UpdateConnectionTime(addr2, provisionalTime))

	// introduce a gap between the two requests so that only the first will be removed later when waited
	// simulate passing a unit of time
	for rct := range entries.data[addr2].recentConnectionTimes {
		entries.data[addr2].recentConnectionTimes[rct] =
			entries.data[addr2].recentConnectionTimes[rct].Add(-1 * timeUnit)
	}

	// value 2
	_, waitTime, provisionalTime = entries.GetConnectionWaitTime(addr2)
	require.Equal(t, time.Duration(0), waitTime)
	require.Equal(t, true, entries.UpdateConnectionTime(addr2, provisionalTime))
	// value 3
	_, waitTime, provisionalTime = entries.GetConnectionWaitTime(addr2)
	require.Equal(t, time.Duration(0), waitTime)
	require.Equal(t, true, entries.UpdateConnectionTime(addr2, provisionalTime))

	phBookData = entries.data[addr2].recentConnectionTimes
	// all three times should be queued
	require.Equal(t, 3, len(phBookData))

	// add another element to trigger wait
	_, waitTime, _ = entries.GetConnectionWaitTime(addr2)
	require.Greater(t, int64(waitTime), int64(0))
	// no element should be removed
	phBookData2 = entries.data[addr2].recentConnectionTimes
	require.Equal(t, phBookData[0], phBookData2[0])
	require.Equal(t, phBookData[1], phBookData2[1])
	require.Equal(t, phBookData[2], phBookData2[2])

	// simulate passing of the waitTime duration
	for rct := range entries.data[addr2].recentConnectionTimes {
		entries.data[addr2].recentConnectionTimes[rct] =
			entries.data[addr2].recentConnectionTimes[rct].Add(-1 * waitTime)
	}

	// The wait should be sufficient
	_, waitTime, provisionalTime = entries.GetConnectionWaitTime(addr2)
	require.Equal(t, time.Duration(0), waitTime)
	require.Equal(t, true, entries.UpdateConnectionTime(addr2, provisionalTime))
	// only one element should be removed, and one added
	phBookData2 = entries.data[addr2].recentConnectionTimes
	require.Equal(t, 3, len(phBookData2))

	// make sure the right time was removed
	require.Equal(t, phBookData[1], phBookData2[0])
	require.Equal(t, phBookData[2], phBookData2[1])
}

func TestWaitAndAddConnectionTimeShortWindow(t *testing.T) {
	partitiontest.PartitionTest(t)

	entries := MakePhonebook(3, 2*time.Millisecond).(*phonebookImpl)
	addr1 := "addrABC"

	// Init the data structures
	entries.ReplacePeerList([]string{addr1}, "default", RelayRole)

	// add 3 values. should not wait
	// value 1
	addrInPhonebook, waitTime, provisionalTime := entries.GetConnectionWaitTime(addr1)
	require.Equal(t, true, addrInPhonebook)
	require.Equal(t, time.Duration(0), waitTime)
	require.Equal(t, true, entries.UpdateConnectionTime(addr1, provisionalTime))
	// value 2
	_, waitTime, provisionalTime = entries.GetConnectionWaitTime(addr1)
	require.Equal(t, time.Duration(0), waitTime)
	require.Equal(t, true, entries.UpdateConnectionTime(addr1, provisionalTime))
	// value 3
	_, waitTime, provisionalTime = entries.GetConnectionWaitTime(addr1)
	require.Equal(t, time.Duration(0), waitTime)
	require.Equal(t, true, entries.UpdateConnectionTime(addr1, provisionalTime))

	// give enough time to expire all the elements
	time.Sleep(10 * time.Millisecond)

	// there should not be any wait
	_, waitTime, provisionalTime = entries.GetConnectionWaitTime(addr1)
	require.Equal(t, time.Duration(0), waitTime)
	require.Equal(t, true, entries.UpdateConnectionTime(addr1, provisionalTime))

	// only one time should be left (the newly added)
	phBookData := entries.data[addr1].recentConnectionTimes
	require.Equal(t, 1, len(phBookData))
}

// TestPhonebookRoles tests that the filtering by roles for different
// phonebooks entries works as expected.
func TestPhonebookRoles(t *testing.T) {
	partitiontest.PartitionTest(t)

	relaysSet := []string{"relay1", "relay2", "relay3"}
	archiverSet := []string{"archiver1", "archiver2", "archiver3"}

	ph := MakePhonebook(1, 1).(*phonebookImpl)
	ph.ReplacePeerList(relaysSet, "default", RelayRole)
	ph.ReplacePeerList(archiverSet, "default", ArchivalRole)
	require.Equal(t, len(relaysSet)+len(archiverSet), len(ph.data))
	require.Equal(t, len(relaysSet)+len(archiverSet), ph.Length())

	for _, role := range []Role{RelayRole, ArchivalRole} {
		for k := 0; k < 100; k++ {
			for l := 0; l < 3; l++ {
				entries := ph.GetAddresses(l, role)
				if role == RelayRole {
					for _, entry := range entries {
						require.Contains(t, entry, "relay")
					}
				} else if role == ArchivalRole {
					for _, entry := range entries {
						require.Contains(t, entry, "archiver")
					}
				}
			}
		}
	}
}

func TestRolesOperations(t *testing.T) {
	partitiontest.PartitionTest(t)
	t.Parallel()

	var tests = []struct {
		role       Role
		otherRoles Role
	}{
		{RelayRole, ArchivalRole},
		{ArchivalRole, RelayRole},
	}

	for _, test := range tests {
		combo := RoleSet{roles: test.role}
		combo.Add(test.otherRoles)
		require.Equal(t, test.role|test.otherRoles, combo.roles)
		require.True(t, combo.Has(test.role))
		require.False(t, combo.Is(test.role))
		require.True(t, combo.Has(test.otherRoles))
		require.False(t, combo.Is(test.otherRoles))

		combo.Remove(test.otherRoles)
		require.Equal(t, test.role, combo.roles)
		require.True(t, combo.Has(test.role))
		require.True(t, combo.Is(test.role))
		require.False(t, combo.Has(test.otherRoles))
		require.False(t, combo.Is(test.otherRoles))
	}
}

func TestReplacePeerList(t *testing.T) {
	partitiontest.PartitionTest(t)
	t.Parallel()

	pb := MakePhonebook(1, 1)
	pb.ReplacePeerList([]string{"a", "b"}, "default", RelayRole)
	res := pb.GetAddresses(4, RelayRole)
	require.ElementsMatch(t, []string{"a", "b"}, res)

	pb.ReplacePeerList([]string{"c"}, "default", ArchivalRole)
	res = pb.GetAddresses(4, ArchivalRole)
	require.ElementsMatch(t, []string{"c"}, res)

	// make b archival in addition to relay
	pb.ReplacePeerList([]string{"b", "c"}, "default", ArchivalRole)
	res = pb.GetAddresses(4, RelayRole)
	require.ElementsMatch(t, []string{"a", "b"}, res)
	res = pb.GetAddresses(4, ArchivalRole)
	require.ElementsMatch(t, []string{"b", "c"}, res)

	// update relays
	pb.ReplacePeerList([]string{"a", "b"}, "default", RelayRole)
	res = pb.GetAddresses(4, RelayRole)
	require.ElementsMatch(t, []string{"a", "b"}, res)
	res = pb.GetAddresses(4, ArchivalRole)
	require.ElementsMatch(t, []string{"b", "c"}, res)

	// exclude b from archival
	pb.ReplacePeerList([]string{"c"}, "default", ArchivalRole)
	res = pb.GetAddresses(4, RelayRole)
	require.ElementsMatch(t, []string{"a", "b"}, res)
	res = pb.GetAddresses(4, ArchivalRole)
	require.ElementsMatch(t, []string{"c"}, res)
}

func TestRoleSetPersistence(t *testing.T) {
	partitiontest.PartitionTest(t)
	t.Parallel()

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{
			name: "MakeRoleSet with persistence",
			fn: func(t *testing.T) {
				rs := MakeRoleSet(RelayRole, true)
				require.True(t, rs.IsPersistent(RelayRole), "Expected RelayRole to be persistent")
				require.True(t, rs.hasPersistentRoles(), "Expected HasPersistentRoles to return true")
			},
		},
		{
			name: "MakeRoleSet without persistence",
			fn: func(t *testing.T) {
				rs := MakeRoleSet(RelayRole, false)
				require.False(t, rs.IsPersistent(RelayRole), "Expected RelayRole to not be persistent")
				require.False(t, rs.hasPersistentRoles(), "Expected HasPersistentRoles to return false")
			},
		},
		{
			name: "AddPersistent",
			fn: func(t *testing.T) {
				rs := MakeRoleSet(RelayRole, false)
				rs.AddPersistent(ArchivalRole)

				require.True(t, rs.IsPersistent(ArchivalRole), "Expected ArchivalRole to be persistent")
				require.False(t, rs.IsPersistent(RelayRole), "Expected RelayRole to not be persistent")
				require.True(t, rs.Has(ArchivalRole), "Expected to have ArchivalRole")
				require.True(t, rs.Has(RelayRole), "Expected to have RelayRole")
				require.True(t, rs.hasPersistentRoles(), "Expected HasPersistentRoles to return true")
			},
		},
		{
			name: "Multiple roles with different persistence",
			fn: func(t *testing.T) {
				rs := MakeRoleSet(RelayRole, true)
				rs.Add(ArchivalRole)

				require.True(t, rs.IsPersistent(RelayRole), "Expected RelayRole to be persistent")
				require.False(t, rs.IsPersistent(ArchivalRole), "Expected ArchivalRole to not be persistent")
				require.True(t, rs.Has(RelayRole), "Expected to have RelayRole")
				require.True(t, rs.Has(ArchivalRole), "Expected to have ArchivalRole")
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, test.fn)
	}
}
