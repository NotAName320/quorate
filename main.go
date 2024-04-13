/*
QUORATE main file
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

package main

import (
	"bufio"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	nsclient "quorate/internal/ns-client"
	"sort"
	"strconv"
	"strings"
	"time"
)

const Gpl = `QUORATE v1.0.4, Copyright (C) 2024 Nota
This program comes with ABSOLUTELY NO WARRANTY.
This is free software, and you are welcome to redistribute it
under certain conditions. Check the LICENSE file for more info.
`

type Hit struct {
	Name          string
	Delegate      string
	SecondNation  string
	IsDeltip      bool
	UpdateTime    int64
	TriggerRegion string
	TriggerTime   int64
}

type RegionsDumpList struct {
	Regions []RegionDump `xml:"REGION"`
}

type RegionDump struct {
	Name      string `xml:"NAME"`
	LastMinor int64  `xml:"LASTMINORUPDATE"`
	LastMajor int64  `xml:"LASTMAJORUPDATE"`
}

func flagPassed(name string, set flag.FlagSet) bool {
	found := false
	set.Visit(func(f *flag.Flag) {
		if f.Name == name {
			found = true
		}
	})
	return found
}

func main() {
	var maxEndoCount int
	var minimumTrigger int
	var isMinor bool
	var reDownDump bool
	var proposalId string
	var userAgent string

	flagSet := flag.FlagSet{}
	flagSet.StringVar(&userAgent, "useragent", "", "Your user agent")
	flagSet.StringVar(&proposalId, "proposal", "", "The proposal ID")
	flagSet.IntVar(&maxEndoCount, "endos", -1, "The maximum endorsement count for a target")
	flagSet.IntVar(&minimumTrigger, "mintrig", -1, "The minimum trigger time")
	flagSet.BoolVar(&isMinor, "minor", false, "Use if generating times for minor")
	flagSet.BoolVar(&reDownDump, "redownload", false, "Use to redownload the daily dump if it's already present")

	err := flagSet.Parse(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		return
	} else if err != nil {
		log.Fatal(err)
	}

	fmt.Println(Gpl)

	scanner := bufio.NewScanner(os.Stdin)
	for userAgent == "" {
		fmt.Print("Enter your main nation: ")
		scanner.Scan()
		userAgent = scanner.Text()
		time.Sleep(50 * time.Millisecond) //avoids weird behavior on ctrl C
	}
	nsclient.SetUserAgent(userAgent)
	log.Println("User agent set to " + userAgent)
	time.Sleep(500 * time.Millisecond)

	for proposalId == "" {
		fmt.Print("Enter a World Assembly Proposal ID (e.g. proposal_id_12312312): ")
		scanner.Scan()
		proposalId = scanner.Text()
		time.Sleep(50 * time.Millisecond)
	}
	log.Println("Proposal set to " + proposalId)
	time.Sleep(500 * time.Millisecond)

	for endoCountString := ""; err != nil || maxEndoCount < 1; maxEndoCount, err = strconv.Atoi(endoCountString) {
		fmt.Print("Enter the endo count: ")
		scanner.Scan()
		endoCountString = strings.ToLower(scanner.Text())
		time.Sleep(50 * time.Millisecond)
	}
	log.Printf("Endo count set to %d!\n", maxEndoCount)
	time.Sleep(500 * time.Millisecond)

	for minTrigString := ""; err != nil || minimumTrigger < 1; minimumTrigger, err = strconv.Atoi(minTrigString) {
		fmt.Print("Enter the minimum trigger time: ")
		scanner.Scan()
		minTrigString = strings.ToLower(scanner.Text())
		time.Sleep(50 * time.Millisecond)
	}
	log.Printf("Minimum trigger set to %d!\n", minimumTrigger)
	time.Sleep(500 * time.Millisecond)

	if !flagPassed("minor", flagSet) {
		var choice string
		for choice != "major" && choice != "minor" {
			fmt.Print("Which update do you want to search for? (minor/major) ")
			scanner.Scan()
			choice = strings.ToLower(scanner.Text())
			time.Sleep(50 * time.Millisecond)
		}
		isMinor = choice == "minor"
	}

	getNewDump := true
	if _, err := os.Stat("regions.xml"); err == nil {
		if flagPassed("redownload", flagSet) {
			getNewDump = reDownDump
		} else {
			choice := "qwerty"
			for choice != "y" && choice != "n" && choice != "" {
				fmt.Print("Daily regions dump already downloaded! Download again? (Y/n) ")
				scanner.Scan()
				choice = strings.ToLower(scanner.Text())
				time.Sleep(50 * time.Millisecond)
			}
			if choice == "n" {
				getNewDump = false
			}
		}
	}
	if getNewDump {
		log.Println("Getting region dump...")
		err := nsclient.GetRegionDump()
		if err != nil {
			log.Fatal(err)
		}
		log.Println("Region dump saved!")
	}

	log.Printf("Getting approvals on proposal %s...\n", proposalId)
	approvals, err := nsclient.GetProposalApprovals(proposalId)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("%d approvals found!\n", len(approvals))
	log.Println("Checking which regions can be hit (this may take a while)...")
	var hittable []Hit
	for _, approval := range approvals {
		//we do this by api to see if delbumps are possible. we also get update times just to sort them here.
		region, err := nsclient.GetNationRegion(approval)
		if err != nil {
			log.Fatal(err)
		}

		regionInfo, err := nsclient.GetRegioninfo(region)
		if err != nil {
			log.Fatal(err)
		}

		if regionInfo.Password {
			continue
		}
		region = strings.Replace(strings.ToLower(region), " ", "_", -1)
		var updateTime int64
		if isMinor {
			updateTime = regionInfo.LastMinor
		} else {
			updateTime = regionInfo.LastMajor
		}

		if regionInfo.DelEndos < maxEndoCount {
			hit := Hit{Name: region, Delegate: approval, IsDeltip: false, UpdateTime: updateTime}
			hittable = append(hittable, hit)
			log.Printf("Region %s with delegate %s can be hit!\n", region, approval)
		} else if regionInfo.DelEndos < regionInfo.SecondEndos+maxEndoCount {
			hit := Hit{Name: region, Delegate: approval, SecondNation: regionInfo.SecondNation, IsDeltip: true,
				UpdateTime: updateTime}
			hittable = append(hittable, hit)
			log.Printf("Region %s with delegate %s can be deltipped by nation %s!\n", region, approval,
				regionInfo.SecondNation)
		}
		time.Sleep(1500 * time.Millisecond) //courtesy sleep even though we have ratelimiting
	}
	if len(hittable) == 0 {
		log.Print("No regions are hittable! Press enter to exit...")
		scanner.Scan()
	}

	log.Printf("Checks done! %d regions are hittable!\n", len(hittable))

	//sort hittable targets by update time
	log.Println("Sorting regions by update time...")
	sort.Slice(hittable[:], func(i, j int) bool {
		return hittable[i].UpdateTime < hittable[j].UpdateTime
	})
	log.Println("Regions sorted!")

	log.Println("Loading regions dump...")
	dumpFile, err := os.ReadFile("regions.xml")
	if err != nil {
		log.Fatal(err)
	}

	var regionsList RegionsDumpList
	err = xml.Unmarshal(dumpFile, &regionsList)
	if err != nil {
		log.Fatal(err)
	}
	regions := regionsList.Regions

	dumpFile = nil //remove from memory
	log.Println("Regions dump loaded!")

	updateTimes := make(map[int64]string)
	firstUpdateRegion := regions[0].Name
	var firstUpdateTime int64
	if isMinor {
		firstUpdateTime = regions[0].LastMinor
	} else {
		firstUpdateTime = regions[0].LastMajor
	}

	hitIndex := 0
	log.Println("Getting triggers for regions...")
	for _, region := range regions {
		canonName := strings.Replace(strings.ToLower(region.Name), " ", "_", -1)

		if hitIndex == len(hittable) {
			break
		}

		var regionUpdate int64
		if isMinor {
			regionUpdate = region.LastMinor
		} else {
			regionUpdate = region.LastMajor
		}

		if _, exists := updateTimes[regionUpdate]; !exists {
			updateTimes[regionUpdate] = canonName
		}

		//edge case where region doesn't exit in daily dump
		if hitIndex != len(hittable)-1 && canonName == hittable[hitIndex+1].Name {
			hitIndex++
		}

		if canonName == hittable[hitIndex].Name {
			hittable[hitIndex].UpdateTime = regionUpdate
			for i := 0; true; i++ {
				trigTime := regionUpdate - int64(minimumTrigger+i)
				if trigRegion, exists := updateTimes[trigTime]; exists {
					hittable[hitIndex].TriggerTime = trigTime
					hittable[hitIndex].TriggerRegion = trigRegion
					hitIndex++
					break
				} else if trigTime <= firstUpdateTime {
					hittable[hitIndex].TriggerTime = firstUpdateTime
					hittable[hitIndex].TriggerRegion = firstUpdateRegion
					hitIndex++
					break
				}
			}
		}
	}
	log.Println("Triggers obtained!")

	log.Println("Creating trigger_list.txt and raidFile.txt...")
	var triggerFileBuilder strings.Builder
	var raidFileBuilder strings.Builder

	for i, hit := range hittable {
		if hit.TriggerRegion == "" {
			continue
		}

		firstRegionTimeDiff := (time.Duration(hit.UpdateTime-firstUpdateTime) * time.Second).String()
		triggerTimeDiff := time.Duration(hit.UpdateTime-hit.TriggerTime) * time.Second

		triggerFileBuilder.WriteString(hit.TriggerRegion + "\n")
		raidFileBuilder.WriteString(fmt.Sprintf("%d) https://www.nationstates.net/region=%s (%s)\n", i+1, hit.Name,
			firstRegionTimeDiff))
		if hit.IsDeltip {
			raidFileBuilder.WriteString(fmt.Sprintf("ENDORSE: https://www.nationstates.net/nation=%s\n", hit.SecondNation))
		}
		raidFileBuilder.WriteString(fmt.Sprintf("\ta) https://www.nationstates.net/template-overall=none/region=%s (%s)\n\n",
			hit.TriggerRegion, triggerTimeDiff))
	}

	triggerFile, err := os.Create("trigger_list.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer triggerFile.Close()
	_, err = triggerFile.WriteString(triggerFileBuilder.String())
	if err != nil {
		log.Fatal(err)
	}

	raidFile, err := os.Create("raidFile.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer raidFile.Close()
	_, err = raidFile.WriteString(raidFileBuilder.String())
	if err != nil {
		log.Fatal(err)
	}

	log.Print("Files created! Press enter to exit...")
	scanner.Scan()
}
