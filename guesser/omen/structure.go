package omen

type GuessStructure struct {
	cp          map[string]map[int][]string
	maxLevel    int
	ip          string
	cpLength    int
	targetLevel int
	optimizer   *Optimizer
	parseTree   []ParseTreeNode
}

func NewGuessStructure(cp map[string]map[int][]string, maxLevel int, ip string, cpLength, targetLevel int, opt *Optimizer) *GuessStructure {
	return &GuessStructure{
		cp:          cp,
		maxLevel:    maxLevel,
		ip:          ip,
		cpLength:    cpLength,
		targetLevel: targetLevel,
		optimizer:   opt,
		parseTree:   nil,
	}
}

func (gs *GuessStructure) NextGuess() string {
	if gs.parseTree == nil {
		gs.parseTree = gs.fillOutParseTree(gs.ip, gs.cpLength, gs.targetLevel)
		if gs.parseTree == nil {
			return ""
		}
		return gs.formatGuess()
	}

	last := &gs.parseTree[len(gs.parseTree)-1]
	cpChars := gs.cp[last.IP][last.Level]
	if last.Index+1 < len(cpChars) {
		last.Index++
		return gs.formatGuess()
	}

	element := *last
	gs.parseTree = gs.parseTree[:len(gs.parseTree)-1]

	if len(gs.parseTree) == 0 {
		return ""
	}

	reqLength := 1
	reqLevel := element.Level + gs.parseTree[len(gs.parseTree)-1].Level

	for len(gs.parseTree) > 0 {
		last = &gs.parseTree[len(gs.parseTree)-1]
		last.Index++

		depthLevel := last.Level

		for {
			cpChars := gs.cp[last.IP][depthLevel]
			for last.Index < len(cpChars) {
				newIP := element.IP[:len(element.IP)-1] + cpChars[last.Index]
				newElements := gs.fillOutParseTree(newIP, reqLength, reqLevel-depthLevel)
				if newElements != nil {
					gs.parseTree = append(gs.parseTree, newElements...)
					return gs.formatGuess()
				}
				last.Index++
			}

			if depthLevel == 0 {
				break
			}

			cpIndex, newLevel := gs.findCP(last.IP, depthLevel-1, 0)
			if cpIndex == nil {
				break
			}
			depthLevel = newLevel
			last.Level = depthLevel
			last.Index = 0
		}

		element = gs.parseTree[len(gs.parseTree)-1]
		gs.parseTree = gs.parseTree[:len(gs.parseTree)-1]
		reqLength++

		if len(gs.parseTree) > 0 {
			reqLevel += gs.parseTree[len(gs.parseTree)-1].Level
		}
	}

	return ""
}

func (gs *GuessStructure) formatGuess() string {
	guess := gs.ip
	for _, item := range gs.parseTree {
		chars := gs.cp[item.IP][item.Level]
		if item.Index < len(chars) {
			guess += chars[item.Index]
		}
	}
	return guess
}

func (gs *GuessStructure) fillOutParseTree(ip string, length, targetLevel int) []ParseTreeNode {
	if length == 1 {
		cpIndex, cpLevel := gs.findCP(ip, targetLevel, targetLevel)
		if cpIndex == nil {
			return nil
		}
		prefix := ip
		if len(ip) > 1 {
			prefix = ip[:len(ip)-1]
		}
		return []ParseTreeNode{{IP: prefix, Level: cpLevel, Index: 0}}
	}

	if length <= gs.optimizer.MaxLength {
		if found, result := gs.optimizer.Lookup(ip, length, targetLevel); found {
			return result
		}
	}

	optimizeLevelTarget := targetLevel
	curLevel := targetLevel

	for curLevel >= 0 {
		cpIndex, cpLevel := gs.findCP(ip, curLevel, 0)
		if cpIndex == nil {
			if length <= gs.optimizer.MaxLength {
				gs.optimizer.Update(ip, length, optimizeLevelTarget, nil)
			}
			return nil
		}

		nextLength := length - 1
		prefix := ip
		if len(ip) > 1 {
			prefix = ip[:len(ip)-1]
		}
		for curIndex := 0; curIndex < len(cpIndex); curIndex++ {
			nextIP := ip[1:] + cpIndex[curIndex]
			workingParseTree := gs.fillOutParseTree(nextIP, nextLength, targetLevel-cpLevel)
			if workingParseTree != nil {
				result := append([]ParseTreeNode{{IP: prefix, Level: cpLevel, Index: curIndex}}, workingParseTree...)
				if length <= gs.optimizer.MaxLength {
					gs.optimizer.Update(ip, length, optimizeLevelTarget, result)
				}
				return result
			}
		}

		curLevel = cpLevel - 1
	}

	if length <= gs.optimizer.MaxLength {
		gs.optimizer.Update(ip, length, optimizeLevelTarget, nil)
	}
	return nil
}

func (gs *GuessStructure) findCP(ip string, topLevel, bottomLevel int) ([]string, int) {
	prefix := ip
	if len(ip) > 1 {
		prefix = ip[:len(ip)-1]
	}
	if gs.maxLevel < topLevel {
		topLevel = gs.maxLevel
	}
	for topLevel >= bottomLevel {
		if chars, ok := gs.cp[prefix][topLevel]; ok {
			return chars, topLevel
		}
		topLevel--
	}
	return nil, 0
}
