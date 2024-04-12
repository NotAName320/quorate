/*
XML structs for QUORATE's NSAPI client
Copyright (C) 2024 Nota

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package ns_client

type ApiRootNode interface {
	WaProposalsOuter | RegionOuter | NationRegion
}

type WaProposalsOuter struct {
	Inner WaProposalList `xml:"PROPOSALS"`
}

type WaProposalList struct {
	Proposals []Proposal `xml:"PROPOSAL"`
}

type Proposal struct {
	Id        string `xml:"id,attr"`
	Approvals string `xml:"APPROVALS"`
}

type RegionOuter struct {
	EndoRanks EndoCensusRanks `xml:"CENSUSRANKS"`
	Tags      RegionTags      `xml:"TAGS"`
	LastMajor int64           `xml:"LASTMAJORUPDATE"`
	LastMinor int64           `xml:"LASTMINORUPDATE"`
}

type RegionTags struct {
	TagArray []string `xml:"TAG"`
}

type EndoCensusRanks struct {
	Inner NationList `xml:"NATIONS"`
}

type NationList struct {
	Nations []NationEndos `xml:"NATION"`
}

type NationEndos struct {
	Name  string `xml:"NAME"`
	Endos int    `xml:"SCORE"`
}

type NationRegion struct {
	Region string `xml:"REGION"`
}
