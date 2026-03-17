package omen

type ParseTreeNode struct {
	IP    string
	Level int
	Index int
}

type Optimizer struct {
	MaxLength  int
	tmtoLookup []map[string]map[int][]ParseTreeNode
}

func NewOptimizer(maxLength int) *Optimizer {
	o := &Optimizer{MaxLength: maxLength}
	o.tmtoLookup = make([]map[string]map[int][]ParseTreeNode, maxLength+1)
	for i := 0; i <= maxLength; i++ {
		o.tmtoLookup[i] = make(map[string]map[int][]ParseTreeNode)
	}
	return o
}

func (o *Optimizer) Lookup(ipNgram string, length int, targetLevel int) (bool, []ParseTreeNode) {
	if length > o.MaxLength {
		return false, nil
	}
	levelMap, ok := o.tmtoLookup[length][ipNgram]
	if !ok {
		return false, nil
	}
	pt, ok := levelMap[targetLevel]
	if !ok {
		return false, nil
	}
	return true, customCopy(pt)
}

func (o *Optimizer) Update(ipNgram string, length int, targetLevel int, pt []ParseTreeNode) {
	if length > o.MaxLength {
		return
	}
	if o.tmtoLookup[length][ipNgram] == nil {
		o.tmtoLookup[length][ipNgram] = make(map[int][]ParseTreeNode)
	}
	o.tmtoLookup[length][ipNgram][targetLevel] = customCopy(pt)
}

func customCopy(pt []ParseTreeNode) []ParseTreeNode {
	if pt == nil {
		return nil
	}
	result := make([]ParseTreeNode, len(pt))
	copy(result, pt)
	return result
}
