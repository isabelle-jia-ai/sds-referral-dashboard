package main

import "sync"

type TeamRecruiter struct {
	Team           string
	RecruiterName  string
	RecruiterEmail string
	SlackID        string
}

var teamDRIMu sync.RWMutex

var TeamDRIMapping = map[string]TeamRecruiter{
	"Consumer Experience": {
		Team:           "Consumer Experience",
		RecruiterName:  "Melisa Dindoruk",
		RecruiterEmail: "melisa.dindoruk@applied.co",
	},
	"Control Center": {
		Team:           "Control Center",
		RecruiterName:  "Christine Hoang",
		RecruiterEmail: "christine.hoang@applied.co",
	},
	"Core OS": {
		Team:           "Core OS",
		RecruiterName:  "Anthony O'Keefe",
		RecruiterEmail: "anthony.okeefe@applied.co",
	},
	"Cybersecurity": {
		Team:           "Cybersecurity",
		RecruiterName:  "Patrick Ku",
		RecruiterEmail: "patrick.ku@applied.co",
	},
	"Developer Tooling": {
		Team:           "Developer Tooling",
		RecruiterName:  "Christine Hoang",
		RecruiterEmail: "christine.hoang@applied.co",
	},
	"Diagnostics": {
		Team:           "Diagnostics",
		RecruiterName:  "Patrick Ku",
		RecruiterEmail: "patrick.ku@applied.co",
	},
	"Firmware": {
		Team:           "Firmware",
		RecruiterName:  "Andrey Pak",
		RecruiterEmail: "andrey.pak@applied.co",
	},
	"Leadership": {
		Team:           "Leadership",
		RecruiterName:  "Pri Jain",
		RecruiterEmail: "pri.jain@applied.co",
	},
	"Middleware": {
		Team:           "Middleware",
		RecruiterName:  "Melisa Dindoruk",
		RecruiterEmail: "melisa.dindoruk@applied.co",
	},
	"NextGen": {
		Team:           "NextGen",
		RecruiterName:  "Anthony O'Keefe",
		RecruiterEmail: "anthony.okeefe@applied.co",
	},
	"Other": {
		Team:           "Other",
		RecruiterName:  "Gabe McCarty",
		RecruiterEmail: "gabe.mccarty@applied.co",
	},
	"Product": {
		Team:           "Product",
		RecruiterName:  "Patrick Ku",
		RecruiterEmail: "patrick.ku@applied.co",
	},
	"Sensor": {
		Team:           "Sensor",
		RecruiterName:  "Patrick Ku",
		RecruiterEmail: "patrick.ku@applied.co",
	},
	"Systems - Trucking": {
		Team:           "Systems - Trucking",
		RecruiterName:  "Pri Jain",
		RecruiterEmail: "pri.jain@applied.co",
	},
	"Systems - VOS": {
		Team:           "Systems - VOS",
		RecruiterName:  "Sabrina Chan",
		RecruiterEmail: "sabrina.chan@applied.co",
	},
	"TPM": {
		Team:           "TPM",
		RecruiterName:  "Patrick Ku",
		RecruiterEmail: "patrick.ku@applied.co",
	},
	"Vehicle Ops": {
		Team:           "Vehicle Ops",
		RecruiterName:  "Kristy Wang",
		RecruiterEmail: "kristy.wang@applied.co",
	},
	"Vehicle Systems Integration": {
		Team:           "Vehicle Systems Integration",
		RecruiterName:  "Sabrina Chan",
		RecruiterEmail: "sabrina.chan@applied.co",
	},
}

func GetTeamNames() []string {
	teamDRIMu.RLock()
	defer teamDRIMu.RUnlock()
	teams := make([]string, 0, len(TeamDRIMapping))
	for team := range TeamDRIMapping {
		teams = append(teams, team)
	}
	return teams
}

func GetDRIRecruiter(team string) *TeamRecruiter {
	teamDRIMu.RLock()
	defer teamDRIMu.RUnlock()
	if r, ok := TeamDRIMapping[team]; ok {
		return &r
	}
	if r, ok := TeamDRIMapping["Other"]; ok {
		return &r
	}
	return nil
}

func GetAllRecruiters() []TeamRecruiter {
	teamDRIMu.RLock()
	defer teamDRIMu.RUnlock()
	seen := make(map[string]bool)
	var recruiters []TeamRecruiter
	for _, r := range TeamDRIMapping {
		if !seen[r.RecruiterEmail] {
			seen[r.RecruiterEmail] = true
			recruiters = append(recruiters, r)
		}
	}
	return recruiters
}
