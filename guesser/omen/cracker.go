package omen

type MarkovCracker struct {
	grammar     *Grammar
	optimizer   *Optimizer
	maxLevel    int
	targetLevel int
	startIP     int
	startLength int
	curLen      [2]int

	curIP [2]int

	curGuess *GuessStructure
}

func NewMarkovCracker(grammar *Grammar, targetLevel int, optimizer *Optimizer) *MarkovCracker {
	mc := &MarkovCracker{
		grammar:     grammar,
		optimizer:   optimizer,
		maxLevel:    grammar.MaxLevel,
		targetLevel: targetLevel,
	}
	mc.startIP = mc.findFirstObject(grammar.IP)
	mc.startLength = mc.findFirstObjectMap(grammar.LN)
	return mc
}

func (mc *MarkovCracker) findFirstObject(ip map[int][]string) int {
	for level := 0; level < mc.maxLevel; level++ {
		if len(ip[level]) != 0 {
			return level
		}
	}
	return mc.maxLevel
}

func (mc *MarkovCracker) findFirstObjectMap(ln map[int][]int) int {
	for level := 0; level < mc.maxLevel; level++ {
		if len(ln[level]) != 0 {
			return level
		}
	}
	return mc.maxLevel
}

func (mc *MarkovCracker) NextGuess() string {
	if mc.curGuess == nil {
		mc.curLen = [2]int{mc.startLength, 0}
		mc.curIP = [2]int{mc.startIP, 0}
		mc.curGuess = NewGuessStructure(
			mc.grammar.CP,
			mc.maxLevel,
			mc.grammar.IP[mc.curIP[0]][mc.curIP[1]],
			mc.grammar.LN[mc.curLen[0]][mc.curLen[1]],
			mc.targetLevel-mc.curLen[0]-mc.curIP[0],
			mc.optimizer,
		)
	}

	guess := mc.curGuess.NextGuess()
	for guess == "" {
		if !mc.increaseIPForTarget(mc.targetLevel - mc.curLen[0]) {
			if !mc.increaseLenForTarget() {
				mc.curGuess = nil
				return ""
			}
		}
		guess = mc.curGuess.NextGuess()
	}
	return guess
}

func (mc *MarkovCracker) increaseLenForTarget() bool {
	level := mc.curLen[0]
	index := mc.curLen[1] + 1
	ln := mc.grammar.LN

	for level <= mc.maxLevel {
		size := len(ln[level])
		if size > index {
			mc.curLen = [2]int{level, index}
			mc.curIP = [2]int{mc.startIP, 0}
			mc.curGuess = NewGuessStructure(
				mc.grammar.CP,
				mc.maxLevel,
				mc.grammar.IP[mc.curIP[0]][mc.curIP[1]],
				mc.grammar.LN[mc.curLen[0]][mc.curLen[1]],
				mc.targetLevel-mc.curLen[0]-mc.curIP[0],
				mc.optimizer,
			)
			return true
		}
		level++
		index = 0
		if level > mc.maxLevel || level > mc.targetLevel {
			return false
		}
	}
	return false
}

func (mc *MarkovCracker) increaseIPForTarget(workingTarget int) bool {
	level := mc.curIP[0]
	index := mc.curIP[1] + 1
	ip := mc.grammar.IP

	for level <= mc.maxLevel {
		size := len(ip[level])
		if size > index {
			mc.curIP = [2]int{level, index}
			mc.curGuess = NewGuessStructure(
				mc.grammar.CP,
				mc.maxLevel,
				mc.grammar.IP[mc.curIP[0]][mc.curIP[1]],
				mc.grammar.LN[mc.curLen[0]][mc.curLen[1]],
				mc.targetLevel-mc.curLen[0]-mc.curIP[0],
				mc.optimizer,
			)
			return true
		}
		level++
		index = 0
		if level > mc.maxLevel || level > workingTarget {
			return false
		}
	}
	return false
}
