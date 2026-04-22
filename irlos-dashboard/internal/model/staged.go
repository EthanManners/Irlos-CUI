// SPDX-License-Identifier: GPL-3.0-or-later
package model

// StagedChanges holds pending edits per tab.  Nothing is written to disk
// until the user selects "Apply Changes" at the bottom of the tab.
type StagedChanges struct {
	OBS    map[string]string
	Noalbs map[string]string
	Config map[string]string
}

func NewStagedChanges() StagedChanges {
	return StagedChanges{
		OBS:    make(map[string]string),
		Noalbs: make(map[string]string),
		Config: make(map[string]string),
	}
}

// Set stages a value for the given tab and field key.
func (s *StagedChanges) Set(tab Tab, key, value string) {
	switch tab {
	case TabOBS:
		s.OBS[key] = value
	case TabNoalbs:
		s.Noalbs[key] = value
	case TabConfig:
		s.Config[key] = value
	}
}

// Clear removes all staged changes for a tab (called after apply).
func (s *StagedChanges) Clear(tab Tab) {
	switch tab {
	case TabOBS:
		s.OBS = make(map[string]string)
	case TabNoalbs:
		s.Noalbs = make(map[string]string)
	case TabConfig:
		s.Config = make(map[string]string)
	}
}
