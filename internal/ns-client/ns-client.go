/*
QUORATE's NationStates API client
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

import (
	"compress/gzip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
)

const apiUrl = "https://www.nationstates.net/cgi-bin/api.cgi"
const version = "1.0.0"

var httpClient = http.DefaultClient
var userAgent = ""

type RegionInfo struct {
	DelEndos     int
	SecondEndos  int
	SecondNation string
	Password     bool
	LastMajor    int64
	LastMinor    int64
}

func SetUserAgent(mainNation string) {
	userAgent = url.QueryEscape("quorate v" + version + " developed by nation=Notanam, in use by nation=" + mainNation)
}

func makeAPIRequest[T ApiRootNode](data url.Values) (returned T, error error) {
	var zero T

	if userAgent == "" {
		return zero, errors.New("no user agent set")
	}

	req, err := http.NewRequest("POST", apiUrl, strings.NewReader(data.Encode()))
	if err != nil {
		return zero, err
	}

	req.Header.Add("User-Agent", userAgent)
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

	for {
		result, err := httpClient.Do(req)
		if err != nil {
			return zero, err
		}

		if result.StatusCode == http.StatusTooManyRequests {
			retryAfter := result.Header.Get("Retry-After")
			log.Print("Hit rate limit, trying again in " + retryAfter)
			intRetry, _ := strconv.Atoi(retryAfter)
			time.Sleep(time.Duration(intRetry) * time.Second)
		} else {
			if remaining, _ := strconv.Atoi(result.Header.Get("RateLimit-Remaining")); remaining <= 7 {
				log.Print("Getting close to rate limit... slowing down")
				reset, _ := strconv.Atoi(result.Header.Get("RateLimit-Reset"))
				time.Sleep(time.Duration(reset/remaining+1) * time.Second)
			}
			if result.StatusCode != http.StatusOK {
				return zero, fmt.Errorf("bad status: %s", result.Status)
			}
			bodyBytes, err := io.ReadAll(result.Body)
			if err != nil {
				return zero, err
			}

			k := zero
			err = xml.Unmarshal(bodyBytes, &k)
			if err != nil {
				return zero, err
			}

			return k, nil
		}
	}
}

func GetProposalApprovals(id string) (delegates []string, error error) {
	data := url.Values{}
	data.Add("wa", "2")
	data.Add("q", "proposals")

	xmlProposals, err := makeAPIRequest[WaProposalsOuter](data)
	if err != nil {
		return nil, err
	}

	for _, proposal := range xmlProposals.Inner.Proposals {
		if proposal.Id == id {
			return strings.Split(proposal.Approvals, ":"), nil
		}
	}

	data.Set("wa", "1")
	xmlProposals, err = makeAPIRequest[WaProposalsOuter](data)
	if err != nil {
		return nil, err
	}

	for _, proposal := range xmlProposals.Inner.Proposals {
		if proposal.Id == id {
			return strings.Split(proposal.Approvals, ":"), nil
		}
	}

	return nil, errors.New("proposal not found")
}

func GetRegioninfo(region string) (regionInfo RegionInfo, error error) {
	var zero RegionInfo

	data := url.Values{}
	data.Add("region", region)
	data.Add("q", "censusranks+tags+lastmajorupdate+lastminorupdate")
	data.Add("scale", "66")

	xmlRegion, err := makeAPIRequest[RegionOuter](data)
	if err != nil {
		return zero, err
	}

	endoRanks := xmlRegion.EndoRanks.Inner.Nations
	if len(endoRanks) < 2 {
		return zero, errors.New("how tf was this even triggered")
	}

	return RegionInfo{
		DelEndos:     endoRanks[0].Endos,
		SecondEndos:  endoRanks[1].Endos,
		SecondNation: endoRanks[1].Name,
		Password:     slices.Contains(xmlRegion.Tags.TagArray, "Password"),
		LastMinor:    xmlRegion.LastMinor,
		LastMajor:    xmlRegion.LastMajor,
	}, nil
}

func GetNationRegion(nation string) (regionName string, error error) {
	data := url.Values{}
	data.Add("nation", nation)
	data.Add("q", "region")

	xmlRegion, err := makeAPIRequest[NationRegion](data)
	if err != nil {
		return "", err
	}

	return xmlRegion.Region, nil
}

func GetRegionDump() (error error) {
	err := downloadRegionDump()
	if err != nil {
		return err
	}

	err = unzipRegionDump()
	if err != nil {
		return err
	}

	_ = os.Remove("regions.xml.gz")

	return nil
}

func downloadRegionDump() (error error) {
	if userAgent == "" {
		return errors.New("no user agent set")
	}

	out, err := os.Create("regions.xml.gz")
	if err != nil {
		return err
	}
	defer out.Close()

	req, err := http.NewRequest("GET", "https://www.nationstates.net/pages/regions.xml.gz", nil)
	if err != nil {
		return err
	}

	req.Header.Add("User-Agent", userAgent)
	result, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	if result.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", result.Status)
	}
	defer result.Body.Close()

	_, err = io.Copy(out, result.Body)
	if err != nil {
		return err
	}

	return nil
}

func unzipRegionDump() (error error) {
	uncompressed, err := os.Create("regions.xml")
	if err != nil {
		return err
	}
	defer uncompressed.Close()

	zippedDump, err := os.Open("regions.xml.gz")
	if err != nil {
		return err
	}
	defer zippedDump.Close()

	gzReader, err := gzip.NewReader(zippedDump)
	defer gzReader.Close()

	_, err = io.Copy(uncompressed, gzReader)
	if err != nil {
		return err
	}

	return nil
}
