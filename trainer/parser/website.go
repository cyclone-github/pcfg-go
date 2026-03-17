package parser

import (
	"strings"
	"unicode"
)

func isURLChar(r rune) bool {
	return unicode.IsLetter(r) ||
		unicode.IsDigit(r) ||
		strings.ContainsRune(".-_/?:#=&%", r)
}

func detectWebsite(section Section) ([]Section, string, string, string) {
	original := section.Value
	working := strings.ToLower(original)

	if !strings.Contains(working, ".") {
		return []Section{section}, "", "", ""
	}

	tldList := getTLDList()

	for _, tld := range tldList {
		searchIndex := 0

		for {
			idx := strings.Index(working[searchIndex:], tld)
			if idx == -1 {
				break
			}

			totalIndex := searchIndex + idx

			// ensure valid boundary after TLD
			if totalIndex+len(tld) < len(working) {
				nextChar := rune(working[totalIndex+len(tld)])
				if unicode.IsLetter(nextChar) || unicode.IsDigit(nextChar) || nextChar == '-' {
					searchIndex = totalIndex + len(tld)
					continue
				}
			}

			endIndex := totalIndex + len(tld)

			// scan forward for full URL
			endOfURL := endIndex
			for endOfURL < len(working) && isURLChar(rune(working[endOfURL])) {
				endOfURL++
			}

			// determine start index
			startIndex := strings.LastIndex(working[:totalIndex], ".") + 1
			if startIndex == 0 {
				startIndex = strings.LastIndex(working[:totalIndex], "/") + 1
			}
			if startIndex == 0 {
				startIndex = strings.LastIndex(working[:totalIndex], ":") + 1
			}
			if startIndex == 0 {
				startIndex = strings.LastIndex(working[:totalIndex], " ") + 1
			}

			host := working[startIndex : totalIndex+len(tld)]

			prefix := ""
			startOfURL := -1

			if startIndex == 0 {
				startOfURL = 0
			}

			if startOfURL == -1 {
				if pi := strings.LastIndex(working[:startIndex+1], "http://www."); pi != -1 {
					prefix = "http://www."
					startOfURL = pi
				}
			}
			if startOfURL == -1 {
				if pi := strings.LastIndex(working[:startIndex+1], "https://www."); pi != -1 {
					prefix = "https://www."
					startOfURL = pi
				}
			}
			if startOfURL == -1 {
				if pi := strings.LastIndex(working[:startIndex], "https://"); pi != -1 {
					prefix = "https://"
					startOfURL = pi
				}
			}
			if startOfURL == -1 {
				if pi := strings.LastIndex(working[:startIndex], "http://"); pi != -1 {
					prefix = "http://"
					startOfURL = pi
				}
			}
			if startOfURL == -1 {
				if pi := strings.LastIndex(working[:startIndex], "www."); pi != -1 {
					prefix = "www."
					startOfURL = pi
				}
			}
			if startOfURL == -1 {
				startOfURL = 0
			}

			fullURL := original[startOfURL:endOfURL]

			var parsing []Section
			if startOfURL != 0 {
				parsing = append(parsing, Section{Value: original[0:startOfURL]})
			}
			parsing = append(parsing, Section{Value: original[startOfURL:endOfURL], Type: "W"})
			if endOfURL != len(original) {
				parsing = append(parsing, Section{Value: original[endOfURL:]})
			}

			return parsing, fullURL, host, prefix
		}
	}

	return []Section{section}, "", "", ""
}

func WebsiteDetection(sectionList []Section) ([]Section, []string, []string, []string) {
	var urlList, hostList, prefixList []string

	index := 0
	for index < len(sectionList) {
		if sectionList[index].Type == "" {
			parsing, url, host, prefix := detectWebsite(sectionList[index])
			if url != "" {
				urlList = append(urlList, url)
				hostList = append(hostList, host)
				prefixList = append(prefixList, prefix)
				sectionList = spliceReplace(sectionList, index, parsing)
			}
		}
		index++
	}
	return sectionList, urlList, hostList, prefixList
}
