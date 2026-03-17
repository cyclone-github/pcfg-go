package parser

func getTLDList() []string {
	return []string{
		// legacy / core
		".com", ".org", ".edu", ".gov", ".mil", ".net",

		// major ccTLDs
		".us", ".uk", ".ca", ".de", ".jp", ".fr", ".au", ".ru", ".ch", ".it",
		".nl", ".se", ".no", ".es",
		".cn", ".in", ".br", ".mx", ".kr", ".za", ".pl", ".tr", ".ir", ".id",
		".sg", ".hk", ".tw", ".vn", ".ar", ".cl", ".nz", ".be", ".fi", ".dk",

		// modern gTLDs
		".info", ".biz", ".xyz", ".online", ".site", ".top", ".club", ".live",
		".shop", ".store", ".tech", ".app", ".dev", ".blog", ".cloud",

		// short / popular in passwords
		".co", ".io", ".ai", ".me", ".gg", ".tv", ".cc",

		// optional extras
		".pw", ".name", ".pro", ".win", ".loan", ".click",
	}
}
