package utils

// 域名匹配
import (
	"regexp"
	"sort"
	"strings"
	"sync"
)

var (
	// 预编译的正则表达式
	regexSyncMap = sync.Map{}
)

func loadRegex(pattern string) (*regexp.Regexp, error) {
	r, ok := regexSyncMap.Load(pattern)
	if !ok {
		r, err := regexp.Compile(pattern)
		if err != nil {
			return nil, err
		}
		regexSyncMap.Store(pattern, r)
		return r, nil
	}
	return r.(*regexp.Regexp), nil
}

// IsRegexValid 判断字符串是否为合法的 /pattern/ 正则格式。
func IsRegexValid(pattern string) bool {
	regexBody, ok := unwrapRegexPattern(pattern)
	if !ok {
		return false
	}
	_, err := regexp.Compile(regexBody)
	return err == nil
}

// MatchDomainByRegex 判断域名是否匹配规则，/pattern/ 按正则匹配，否则按普通域名匹配。
func MatchDomainByRegex(pattern, domain string) bool {
	rule := strings.TrimSpace(pattern)
	domain = strings.TrimSpace(domain)

	regexBody, isRegex := unwrapRegexPattern(rule)
	if !isRegex {
		return strings.EqualFold(rule, domain)
	}
	re, err := loadRegex(regexBody)
	if err != nil {
		return false
	}
	return re.MatchString(domain)
}

// IsDomainRegex reports whether a domain rule uses the /pattern/ syntax.
// Keeping this classification in one place prevents callers from repeatedly
// treating ordinary domains as regular expressions on the request path.
func IsDomainRegex(pattern string) bool {
	_, ok := unwrapRegexPattern(pattern)
	return ok
}

// IsRegexListMutuallyExclusive 判断规则数组是否互斥且无包含关系，并返回有问题的规则。
func IsRegexListMutuallyExclusive(patterns []string) (bool, []string) {
	if len(patterns) <= 1 {
		return true, nil
	}

	matchers := make([]ruleMatcher, len(patterns))
	conflicts := make(map[string]struct{})

	for i, pattern := range patterns {
		matchers[i] = buildRuleMatcher(pattern)
		if matchers[i].invalid {
			conflicts[pattern] = struct{}{}
		}
	}

	allSamples := collectSamplesFromMatchers(matchers)

	// 通过候选域名样本近似判断冲突与包含关系。
	for i := 0; i < len(patterns); i++ {
		if matchers[i].invalid {
			continue
		}
		for j := i + 1; j < len(patterns); j++ {
			if matchers[j].invalid {
				continue
			}

			patternI := patterns[i]
			patternJ := patterns[j]

			if strings.EqualFold(patternI, patternJ) {
				conflicts[patternI] = struct{}{}
				conflicts[patternJ] = struct{}{}
				continue
			}

			if hasRuleIntersection(matchers[i], matchers[j], allSamples) {
				conflicts[patternI] = struct{}{}
				conflicts[patternJ] = struct{}{}
				continue
			}

			if hasRuleInclusion(matchers[i], matchers[j], allSamples) ||
				hasRuleInclusion(matchers[j], matchers[i], allSamples) {
				conflicts[patternI] = struct{}{}
				conflicts[patternJ] = struct{}{}
			}
		}
	}

	if len(conflicts) == 0 {
		return true, nil
	}

	problems := make([]string, 0, len(conflicts))
	for pattern := range conflicts {
		problems = append(problems, pattern)
	}
	sort.Strings(problems)
	return false, problems
}

func hasIntersection(a, b *regexp.Regexp, samples []string) bool {
	for _, s := range samples {
		if a.MatchString(s) && b.MatchString(s) {
			return true
		}
	}
	return false
}

func isIncludedBySamples(src, target *regexp.Regexp, samples []string) bool {
	matched := false
	for _, s := range samples {
		if src.MatchString(s) {
			matched = true
			if !target.MatchString(s) {
				return false
			}
		}
	}
	return matched
}

type ruleMatcher struct {
	isRegex bool
	domain  string
	regex   *regexp.Regexp
	invalid bool
}

func buildRuleMatcher(pattern string) ruleMatcher {
	rule := strings.TrimSpace(pattern)
	regexBody, isRegex := unwrapRegexPattern(rule)
	if !isRegex {
		return ruleMatcher{domain: strings.ToLower(rule)}
	}
	re, err := loadRegex(regexBody)
	if err != nil {
		return ruleMatcher{isRegex: true, invalid: true}
	}
	return ruleMatcher{isRegex: true, regex: re}
}

func unwrapRegexPattern(pattern string) (string, bool) {
	s := strings.TrimSpace(pattern)
	if len(s) >= 2 && strings.HasPrefix(s, "/") && strings.HasSuffix(s, "/") {
		body := strings.TrimSpace(s[1 : len(s)-1])
		if body == "" {
			return "", false
		}
		return body, true
	}
	return "", false
}

func hasRuleIntersection(a, b ruleMatcher, samples []string) bool {
	if !a.isRegex && !b.isRegex {
		return strings.EqualFold(a.domain, b.domain)
	}
	if !a.isRegex && b.isRegex {
		return b.regex.MatchString(a.domain)
	}
	if a.isRegex && !b.isRegex {
		return a.regex.MatchString(b.domain)
	}
	return hasIntersection(a.regex, b.regex, samples)
}

func hasRuleInclusion(src, target ruleMatcher, samples []string) bool {
	if !src.isRegex || !target.isRegex {
		return false
	}
	return isIncludedBySamples(src.regex, target.regex, samples)
}

func collectSamplesFromMatchers(matchers []ruleMatcher) []string {
	samples := make([]string, 0, len(matchers)*3)
	for _, matcher := range matchers {
		if matcher.invalid {
			continue
		}
		if matcher.isRegex {
			samples = append(samples, buildDomainSamples(matcher.regex.String())...)
			continue
		}
		samples = append(samples, matcher.domain)
	}
	return uniqueStrings(samples)
}

func buildDomainSamples(pattern string) []string {
	samples := make([]string, 0, 4)
	plain := guessPlainDomain(pattern)
	if plain != "" {
		samples = append(samples, plain, "sub."+plain, "api."+plain)
	}

	samples = append(samples, extractDomainLiterals(pattern)...)
	return uniqueStrings(samples)
}

func guessPlainDomain(pattern string) string {
	s := strings.TrimSpace(pattern)
	s = strings.TrimPrefix(s, "^")
	s = strings.TrimSuffix(s, "$")
	s = strings.ReplaceAll(s, `\.`, ".")
	s = strings.ReplaceAll(s, `\-`, "-")

	if strings.ContainsAny(s, `*+?()[]{}|\\`) {
		return ""
	}
	return s
}

func extractDomainLiterals(pattern string) []string {
	normalized := strings.ReplaceAll(pattern, `\.`, ".")
	re, _ := loadRegex(`[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)+`)
	return re.FindAllString(normalized, -1)
}

func uniqueStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, item := range in {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}
